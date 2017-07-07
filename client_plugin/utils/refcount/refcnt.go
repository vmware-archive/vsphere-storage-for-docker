// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build linux

//
// Refcount discovery from local docker.
//
// Docker issues mount/unmount to a volume plugin each time container
// using this volume is started or stopped(or killed). So if multiple  containers
// use the same volume, it is the responsibility of the plugin to track it.
// This file implements this tracking via refcounting volume usage, and recovering
// refcounts and mount states on plugin or docker restart
//
// When  Docker is killed (-9). Docker may forget
// all about volume usage and consider clean slate. This could lead to plugin
// locking volumes which Docker does not need so we need to recover correct
// refcounts.
//
// Note: when Docker Is properly restarted ('service docker restart'), it shuts
// down accurately  and sends unmounts so refcounts stays in sync. In this case
// there is no need to do anything special in the plugin. Thus all discovery
// code is mainly for crash recovery and cleanup.
//
// Refcounts are changed in Mount/Unmount. The code in this file provides
// generic refcnt API and also supports refcount discovery on restarts:
// - Connects to Docker over unix socket, enumerates Volume Mounts and builds
//   "volume mounts refcount" map as Docker sees it.
// - Gets actual mounts from /proc/mounts, and makes sure the refcounts and
//   actual mounts are in sync.
//
// The process is initiated on plugin start,and ONLY if Docker is already
// running and thus answering client.Info() request.
//
// After refcount discovery, results are compared to /proc/mounts content.
//
// We rely on all plugin mounts being in /mnt/vmdk/<volume_name>, and will
// unount stuff there at will - this place SHOULD NOT be used for manual mounts.
//
// If a volume IS mounted, but should not be (refcount = 0)
//   - we assume there was a restart of VM or even ESX, and
//     the mount is stale (since Docker does not need it)
//   - we unmount / detach it
//
// If a volume is NOT mounted, but should be (refcount > 0)
//   - this should never happen since Docker using volume means bind mount is
//     active so the disk should not have been unmounted
//   - we just log an error and keep going. Recovery in this case is manual
//
// We assume that mounted (in Docker VM) and attached (to Docker VM) is the
// same. If something is attached to VM but not mounted (so from refcnt and
// mountspoint of view the volume is not used, but the VMDK is still attached
// to the VM) - we leave it to manual recovery.
//
// The RefCountsMap is safe to be used by multiple goroutines and has a single
// RWMutex to serialize operations on the map and refCounts.
// The serialization of operations per volume is assured by the volume/store
// of the docker daemon.
//

package refcount

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/plugin_utils"
	"golang.org/x/net/context"
)

const (
	ApiVersion              = "v1.24" // docker engine 1.12 and above support this api version
	DockerUSocket           = "unix:///var/run/docker.sock"
	defaultSleepIntervalSec = 1
	dockerConnTimeoutSec    = 2
	refCountDelayStartSec   = 2
	refCountRetryAttempts   = 20

	photonDriver = "photon"
)

// info about individual volume ref counts and mount
type refCount struct {
	// refcount for the given volume.
	count uint

	// Is the volume mounted from OS point of view
	// (i.e. entry in /proc/mounts exists)
	mounted bool

	// Volume is mounted from this device. Used on recovery only , for info
	// purposes. Value is empty during normal operation
	dev string
}

// RefCountsMap struct
type RefCountsMap struct {
	refMap map[string]*refCount // Map of refCounts
	mtx    *sync.RWMutex        // Synchronizes RefCountsMap ops

	refcntInitSuccess bool        // save refcounting success
	isDirty           bool        // flag to check reconciling has been interrupted
	StateMtx          *sync.Mutex // (Exported) Synchronizes refcounting between mount/unmount and refcounting thread
}

var (
	// vmdk or local. We use "vmdk" only in production, but need "local" to
	// allow no-ESX test. sanity_test.go '-d' flag allows to switch it to local
	driverName string

	// header for Docker Remote API
	defaultHeaders map[string]string

	// root dir for mounted volumes
	mountRoot string
)

// local init() for initializing stuff in before running any code in this file
func init() {
	defaultHeaders = map[string]string{"User-Agent": "engine-api-client-1.0"}
}

// NewRefCountsMap - creates a new RefCountsMap
func NewRefCountsMap() *RefCountsMap {
	return &RefCountsMap{
		refMap: make(map[string]*refCount),
		mtx:    &sync.RWMutex{},

		StateMtx:          &sync.Mutex{},
		isDirty:           false,
		refcntInitSuccess: false,
	}
}

// Creates a new refCount
func newRefCount() *refCount {
	return &refCount{
		count: 0,
	}
}

// return if refcount initialization has been successful
func (r *RefCountsMap) IsInitialized() bool {
	return r.refcntInitSuccess
}

// dirty the background refcount process
// this flag is marked dirty from the driver
// caller acquires lock on state as appropriate
func (r *RefCountsMap) MarkDirty() {
	r.isDirty = true
}

// tries to calculate refCounts for dvs volumes. If failed, triggers a timer
// based reattempt to schedule scan after a delay
func (r *RefCountsMap) Init(d drivers.VolumeDriver, mountDir string, name string) {
	err := r.calculate(d, mountDir, name)
	// If refcounting wasn't successful, schedule one again
	if err != nil {
		log.Infof("Refcounting failed: (%v).", err)
		go func() {
			r.retryCalculate(d, mountDir, name)
		}()
	}
}

// create a timer to calculate refcount after a delay. If failed, retry again
// until retry attempt limit reached
func (r *RefCountsMap) retryCalculate(d drivers.VolumeDriver, mountDir string, name string) {
	attemptLeft := refCountRetryAttempts
	delay := refCountDelayStartSec
	for attemptLeft > 0 {
		// generate a random delay everytime
		log.Infof("Scheduling again after %d seconds", delay)
		timer := time.NewTimer(time.Duration(delay) * time.Second)

		<-timer.C
		err := r.calculate(d, mountDir, driverName)
		if err != nil {
			log.Infof("Refcounting failed: (%v). Attempts left: %d ", err, attemptLeft)
			attemptLeft--
			// exponential backoff
			delay += delay
		} else {
			return // all good
		}
	}
	// couldn't complete refcounting even after retries.
	// docker logs artifical panic and restarts the plugin.
	panic(fmt.Sprintf("Failed to talk to docker to calculate volumes usage. Please restart docker"))
}

// calculate Refcounts. Discover volume usage refcounts from Docker.
func (r *RefCountsMap) calculate(d drivers.VolumeDriver, mountDir string, name string) error {
	c, err := client.NewClient(DockerUSocket, ApiVersion, nil, defaultHeaders)
	if err != nil {
		log.Panicf("Failed to create client for Docker at %s.( %v)",
			DockerUSocket, err)
	}
	mountRoot = mountDir
	driverName = name

	log.Infof("Getting volume data from %s", DockerUSocket)

	ctx, cancel := context.WithTimeout(context.Background(), dockerConnTimeoutSec*time.Second)
	defer cancel()
	info, err := c.Info(ctx)
	if err != nil {
		log.Infof("Can't connect to %s due to (%v), skipping discovery", DockerUSocket, err)
		return err
	}
	log.Debugf("Docker info: version=%s, root=%s, OS=%s",
		info.ServerVersion, info.DockerRootDir, info.OperatingSystem)

	// connects (and polls if needed) and then calls discovery
	err = r.discoverAndSync(c, d)
	if err != nil {
		log.Errorf("Failed to discover mount refcounts(%v)", err)
		return err
	}

	// RLocks the RefCountsMap
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	log.Infof("Discovered %d volumes in use.", len(r.refMap))
	for name, cnt := range r.refMap {
		log.Infof("Volume name=%s count=%d mounted=%t device='%s'",
			name, cnt.count, cnt.mounted, cnt.dev)
	}

	log.Infof("Refcounting successfully completed")
	return nil
}

// Returns ref count for the volume.
// If volume is not referred (not in the map), return 0
func (r *RefCountsMap) GetCount(vol string) uint {
	// RLocks the RefCountsMap
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	rc := r.refMap[vol]
	if rc == nil {
		return 0
	}
	return rc.count
}

// Incr refCount for the volume vol. Creates new entry if needed.
func (r *RefCountsMap) Incr(vol string) uint {
	// Locks the RefCountsMap
	r.mtx.Lock()
	defer r.mtx.Unlock()

	rc := r.refMap[vol]
	if rc == nil {
		rc = newRefCount()
		r.refMap[vol] = rc
	}
	rc.count++
	return rc.count
}

// Decr recfcount for the volume vol and returns the new count
// returns -1  for error (and resets count to 0)
// also deletes the node from the map if refcount drops to 0
func (r *RefCountsMap) Decr(vol string) (uint, error) {
	// Locks the RefCountsMap
	r.mtx.Lock()
	defer r.mtx.Unlock()

	rc := r.refMap[vol]
	if rc == nil {
		return 0, fmt.Errorf("Decr: Missing refcount. name=%s", vol)
	}

	if rc.count == 0 {
		// we should NEVER get here. Even if Docker sends Unmount before Mount,
		// it should be caught in previous check. So delete the entry (in case
		// someone upstairs does 'recover', and panic.
		delete(r.refMap, vol)
		log.Warning("Decr: refcnt already 0 (rc.count=0), name=%s", vol)
		return 0, nil
	}

	rc.count--

	if rc.count < 0 {
		log.Warningf("Decr: Internal error, refcnt is negative. Trying to recover, deleting the counter - name=%s refcnt=%d", vol, rc.count)
	}
	// Deletes the refcount only if there are no references
	if rc.count <= 0 {
		delete(r.refMap, vol)
	}
	return rc.count, nil
}

// check if volume with source as mount_source belongs to vmdk plugin
func isVMDKMount(mount_source string) bool {
	managedPluginMountStart := "/var/lib/docker/plugins/"

	// if plugin is used as a service
	if strings.HasPrefix(mount_source, mountRoot) {
		return true
	}
	// if plugin is used as managed plugin
	// managed plugin has mount source in format:
	// '/var/lib/docker/plugins/{plugin uuid}/rootfs/mnt/vmdk/{volume name}'
	if strings.HasPrefix(mount_source, managedPluginMountStart) && strings.Contains(mount_source, mountRoot) {
		return true
	}

	// mounted volume doesn't belong to vmdk_plugin
	return false
}

// check if refcounting has been made dirty by mounts/unmounts
func (r *RefCountsMap) checkDirty() bool {
	r.StateMtx.Lock()
	defer r.StateMtx.Unlock()
	return r.isDirty
}

// enumerates volumes and  builds RefCountsMap, then sync with mount info
func (r *RefCountsMap) discoverAndSync(c *client.Client, d drivers.VolumeDriver) error {
	// we assume to  have empty refcounts. Let's enforce

	r.StateMtx.Lock()
	r.isDirty = false
	r.StateMtx.Unlock()

	filters := filters.NewArgs()
	filters.Add("status", "running")
	filters.Add("status", "paused")
	filters.Add("status", "restarting")

	ctx, cancel := context.WithTimeout(context.Background(), dockerConnTimeoutSec*time.Second)
	defer cancel()
	containers, err := c.ContainerList(ctx, types.ContainerListOptions{
		All:    true,
		Filter: filters,
	})
	if err != nil {
		log.Errorf("ContainerList failed (err: %v)", err)
		return err
	}

	// use same datastore for all volumes with short names
	datastoreName := ""

	log.Infof("Found %d running or paused containers", len(containers))
	for _, ct := range containers {

		if r.checkDirty() {
			return fmt.Errorf("refcounting wasn't clean.")
		}

		ctx_inspect, cancel_inspect := context.WithTimeout(context.Background(), dockerConnTimeoutSec*time.Second)
		defer cancel_inspect()
		containerJSONInfo, err := c.ContainerInspect(ctx_inspect, ct.ID)
		if err != nil {
			log.Errorf("ContainerInspect failed for %s (err: %v)", ct.Names, err)
			// We intentionally don't cleanup refMap because whatever refCounts(if any) we were able to
			// populate are valid.
			return err
		}
		log.Debugf("  Mounts for %v", ct.Names)
		for _, mount := range containerJSONInfo.Mounts {
			// check if the mount location belongs to vmdk plugin
			if isVMDKMount(mount.Source) != true {
				continue
			}

			volumeInfo, err := plugin_utils.GetVolumeInfo(mount.Name, datastoreName, d)
			if err != nil {
				log.Errorf("Unable to get volume info for volume %s. err:%v", mount.Name, err)
				return err
			}
			datastoreName = volumeInfo.DatastoreName
			r.Incr(volumeInfo.VolumeName)
			log.Debugf("name=%v (driver=%s source=%s) (%v)",
				mount.Name, mount.Driver, mount.Source, mount)
		}
	}

	// lock and check if the background refcount was dirtied.
	// get mounts, remove unncessary mounts and set refcntInitSuccess
	// under same lock to avoid races with parallel mount/unmount
	r.StateMtx.Lock()
	defer r.StateMtx.Unlock()
	if r.isDirty == true {
		// refcounting was dirtied by parallel mount/unmount.
		return fmt.Errorf("refcounting wasn't clean.")
	}

	// Check that refcounts and actual mount info from Linux match
	// If they don't, unmount unneeded stuff, or yell if something is
	// not mounted but should be (it's error. we should not get there)
	r.updateRefMap()
	r.syncMountsWithRefCounters(d)
	// mark reconciling success so that further unmounts can instantly be processed
	r.refcntInitSuccess = true
	return nil
}

// syncronize mount info with refcounts - and unmounts if needed
func (r *RefCountsMap) syncMountsWithRefCounters(d drivers.VolumeDriver) {
	// Lock the RefCountsMap
	r.mtx.Lock()
	defer r.mtx.Unlock()

	for vol, cnt := range r.refMap {
		f := log.Fields{
			"name":    vol,
			"refcnt":  cnt.count,
			"mounted": cnt.mounted,
			"dev":     cnt.dev,
		}

		log.WithFields(f).Debug("Refcnt record: ")
		if cnt.mounted == true {
			if cnt.count == 0 {
				// Volume mounted but not used - UNMOUNT and DETACH !
				log.WithFields(f).Info("Initiating recovery unmount. ")
				err := d.UnmountVolume(vol)
				if err != nil {
					log.Warning("Failed to unmount - manual recovery may be needed")
				}
			}
		} else {
			if cnt.count == 0 {
				// volume unmounted AND refcount 0.  We should NEVER get here
				// since unmounted and recount==0 volumes should have no record
				// in the map. Something went seriously wrong in the code.
				log.WithFields(f).Panic("Internal failure: record should not exist. ")
			} else {
				// No mounts, but Docker tells we have refcounts.
				// It could happen when Docker runs a container with a volume
				// but not using files on the volumes, and the volume is (manually?)
				// unmounted. Unlikely but possible. Mount !
				log.WithFields(f).Warning("Initiating recovery mount. ")
				status, err := d.GetVolume(vol)
				if err != nil {
					log.Warning("Failed to mount - manual recovery may be needed")
				} else {
					//Ensure the refcount map has this disk ID
					id := ""
					exists := false
					if driverName == photonDriver {
						if id, exists = status["ID"].(string); !exists {
							log.Warning("Failed to disk ID for photon disk cannot mount in use disk")
						}
					}

					isReadOnly := false
					if access, exists := status["access"]; exists {
						if access == "read-only" {
							isReadOnly = true
						}
					}
					_, err = d.MountVolume(vol, status["fstype"].(string), id, isReadOnly, false)
					if err != nil {
						log.Warning("Failed to mount - manual recovery may be needed")
					}
				}
			}
		}
	}
}

// updates refcount map with mounted volumes using mount info
func (r *RefCountsMap) updateRefMap() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	volumeMap, err := plugin_utils.GetMountInfo(mountRoot)

	if err != nil {
		return err
	}

	for volName, dev := range volumeMap {
		refInfo := r.refMap[volName]
		if refInfo == nil {
			refInfo = newRefCount()
		}
		refInfo.mounted = true
		refInfo.dev = dev
		r.refMap[volName] = refInfo
		log.Debugf("Found '%s' in /proc/mount, ref=(%#v)", volName, refInfo)
	}

	return nil
}
