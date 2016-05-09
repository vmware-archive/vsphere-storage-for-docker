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
// there is no need to do anything special in the plugin. Thus all this code is
// mainly for crash recovery and cleanup.
//
// Refcounts are tracked in Mount/Unmount. The code in this file provides
// generic reccnt API and also supports refcount discovery on restarts:
// - Connects to Docker over unix socket, enumerates Volume Mounts and builds
//   "volume mounts refcount" map as Docker sees it.
// - Gets actual mounts from /proc/mounts, and makes sure the refcounts and
//   actual mounts are in sync.
//
// The process is initiated on plugin start.  If it fails (e.g. docker is not
//    running), we sleep/retry until Docker answers.
//
// After refcount discovery, results are compared to /proc/mounts content.
//
// We rely on all plugin mounts being in /mnt/vmdk/<volume_name>, and will
// unount stuff there at will - this place SHOULD NOT be used for manual mounts.
//
// If a volume IS mounted, but should not be (refcount = 0)
//   - we assume there was a restart of either Docker or VM or even ESX, and
//     the mount is stale (since Docker does not need it)
//   - we unmount / detach it
//
// If a volume is NOT mounted, but should be (refcount > 0)
//   - this should never happen since Docker using volume means bind mount is
//     active so the disk could not have been unmounted
//   - we just log an error and keep going. Recovery in this case is manual
//
// We assume that mounted (in Docker VM) and attached (to Docker VM) is the
// same. If something is attached to VM but not mounted (so from refcnt and
// mountspoint of view the volume is not used, but the VMDK is still attached
// to the VM) - we leave it to manual recovery.
//

package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

const (
	apiVersion              = "v1.22"
	dockerUSocket           = "unix:///var/run/docker.sock"
	defaultSleepIntervalSec = 3

	// consts for finding and parsing linux mount information
	linuxMountsFile = "/proc/mounts"
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

// volume name -> {count, mounted, device}
type refCountsMap map[string]*refCount

var (
	// vmdk or local. We use "vmdk" only in production, but need "local" to
	// allow no-ESX test. sanity_test.go '-d' flag allows to switch it to local
	driverName string

	// header for Docker Remote API
	defaultHeaders map[string]string

	// wait time for reconnect
	reconnectSleepInterval time.Duration
)

// file init() these things before running any code
func init() {
	defaultHeaders = map[string]string{"User-Agent": "engine-api-client-1.0"}

	// defaults: in test they can be overwritten (see sanity_test.go)
	reconnectSleepInterval = defaultSleepIntervalSec * time.Second
	driverName = "vmdk"
}

// Init Refcounts. Discover volume usage refcounts from Docker.
// This will poll until replied to.
func (r refCountsMap) Init(d *vmdkDriver) {

	c, err := client.NewClient(dockerUSocket, apiVersion, nil, defaultHeaders)
	if err != nil {
		log.Panicf("Failed to create client for Docker at %s.( %v)",
			dockerUSocket, err)
	}

	// connects (and polls if needed) and then calls discovery
	log.Infof("Getting volume data from to %s", dockerUSocket)
	err = r.discoverAndSync(c, d)
	attempt := 1
	for err != nil {
		// stay here forever, until connected to Docker or killed
		attempt++
		if (attempt-2)%10 == 0 { // throttle to each 10th attempt
			log.Warningf("Failed to get data from Docker, retrying. (err: %v)", err)
		}
		time.Sleep(reconnectSleepInterval)
		err = r.discoverAndSync(c, d)
	}

	log.Infof("Discovered %d volumes in use or mounted.", len(r))
	if log.GetLevel() == log.DebugLevel {
		for name, cnt := range r {
			log.Debugf("Volume name=%s count=%d mounted=%t device='%s'",
				name, cnt.count, cnt.mounted, cnt.dev)
		}
	}
}

// Returns ref count for the volume.
// If volume is not referred (not in the map), return 0
// NOTE: this assumes the caller holds the lock if we run concurrently
func (r refCountsMap) getCount(vol string) uint {
	rc := r[vol]
	if rc == nil {
		return 0
	}
	return rc.count
}

// incr refCount for the volume vol. Creates new entry if needed.
func (r refCountsMap) incr(vol string) uint {
	rc := r[vol]
	if rc == nil {
		rc = &refCount{}
		r[vol] = rc
	}
	rc.count++
	return rc.count
}

// decr recfcount for the volume vol and returns the new count
// returns -1  for error (and resets count to 0)
// also deletes the node from the map if refcount drops to 0
func (r refCountsMap) decr(vol string) (uint, error) {
	rc := r[vol]
	if rc == nil {
		return 0, fmt.Errorf("decr: refcount is already 0. name=%s", vol)
	}

	if rc.count == 0 {
		// we should NEVER get here. Even if Docker sends Unmount before Mount,
		// it should be caught in precious check. So delete the entry (in case
		// someone upstairs does 'recover', and panic.
		delete(r, vol)
		log.Panicf("decr: refcnt corruption (rc.count=0), name=%s", vol)
	}

	rc.count--
	if rc.count == 0 {
		defer delete(r, vol)
	}
	return rc.count, nil
}

// enumberates volumes and  builds refCountsMap, then sync with mount info
func (r refCountsMap) discoverAndSync(c *client.Client, d *vmdkDriver) error {
	// we assume to  have empty refcounts. Let's enforce
	for name := range r {
		delete(r, name)
	}

	filters := filters.NewArgs()
	filters.Add("status", "running")
	filters.Add("status", "paused")
	containers, err := c.ContainerList(types.ContainerListOptions{
		All:    true,
		Filter: filters,
	})
	if err != nil {
		return err
	}

	log.Debugf("Found %d running or paused containers", len(containers))
	for _, ct := range containers {
		containerJSONInfo, err := c.ContainerInspect(ct.ID)
		if err != nil {
			log.Errorf("ContainerInspect failed for %s (err: %v)", ct.Names, err)
			continue
		}
		log.Debugf("  Mounts for %v", ct.Names)

		for _, mount := range containerJSONInfo.Mounts {
			if mount.Driver == driverName {
				r.incr(mount.Name)
				log.Debugf("  name=%v (driver=%s source=%s)",
					mount.Name, mount.Driver, mount.Source)
			}
		}
	}

	// Check that refcounts and actual mount info from Linux match
	// If they don't, unmount unneeded stuff, or yell if something is
	// not mounted but should be (it's error. we should not get there)

	r.getMountInfo()
	r.syncMountsWithRefCounters(d)

	return nil
}

// syncronize mount info with refcounts - and unmounts if needed
func (r refCountsMap) syncMountsWithRefCounters(d *vmdkDriver) {
	for vol, cnt := range r {
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
				err := d.unmountVolume(vol)
				if err != nil {
					log.Warning("Failed to unmount - manual recovery may be needed")
				}
			}
			// else: all good, nothing to do - volume mounted and used.

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
				_, err := d.mountVolume(vol)
				if err != nil {
					log.Warning("Failed to mount - manual recovery may be needed")
				}
			}
		}
	}
}

// scans /proc/mounts and updates refcount map witn mounted volumes
func (r refCountsMap) getMountInfo() error {
	data, err := ioutil.ReadFile(linuxMountsFile)
	if err != nil {
		log.Errorf("Can't get info from %s (%v)", linuxMountsFile, err)
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		field := strings.Fields(line)
		if len(field) < 2 {
			continue // skip empty line and lines too short to have our mount
		}
		// fields format: [/dev/sdb /mnt/vmdk/vol1 ext2 rw,relatime 0 0]
		if filepath.Dir(field[1]) != mountRoot {
			continue
		}
		volName := filepath.Base(field[1])
		refInfo := r[volName]
		if refInfo == nil {
			refInfo = &refCount{count: 0}
		}
		refInfo.mounted = true
		refInfo.dev = field[0]
		r[volName] = refInfo
		log.Debugf("Found a mounted volume %s (%#v)", volName, refInfo)
	}

	return nil
}
