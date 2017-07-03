// Copyright 2017 VMware, Inc. All Rights Reserved.
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

package global

//
// VMWare Global Docker Data Volume plugin.
//
// Provide support for --driver=global in Docker.
//
// Serves requests from Docker Engine related to global volume operations.
///

import (
	"fmt"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/plugin_utils"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/refcount"
)

const (
	version = "Global Volume Driver v0.1"
)

// VolumeDriver - global volume driver struct
type VolumeDriver struct {
	refCounts     *refcount.RefCountsMap
	mountIDtoName map[string]string // map of mountID -> full volume name
}

var mountRoot string

// NewVolumeDriver creates driver instance
func NewVolumeDriver(mountDir string, driverName string) *VolumeDriver {
	var d *VolumeDriver

	mountRoot = mountDir

	d = &VolumeDriver{
		refCounts: refcount.NewRefCountsMap(),
	}

	d.mountIDtoName = make(map[string]string)
	d.refCounts.Init(d, mountDir, driverName)

	log.WithFields(log.Fields{
		"version": version,
	}).Info("Docker Global plugin started ")

	return d
}

// In following three operations on refcount, if refcount
// map hasn't been initialized, return 1 to prevent detach and remove.

// Return the number of references for the given volume
func (d *VolumeDriver) getRefCount(vol string) uint {
	if d.refCounts.IsInitialized() != true {
		return 1
	}
	return d.refCounts.GetCount(vol)
}

// Increment the reference count for the given volume
func (d *VolumeDriver) incrRefCount(vol string) uint {
	if d.refCounts.IsInitialized() != true {
		return 1
	}
	return d.refCounts.Incr(vol)
}

// Decrement the reference count for the given volume
func (d *VolumeDriver) decrRefCount(vol string) (uint, error) {
	if d.refCounts.IsInitialized() != true {
		return 1, nil
	}
	return d.refCounts.Decr(vol)
}

// Returns the given volume mountpoint
func getMountPoint(volName string) string {
	return filepath.Join(mountRoot, volName)
}

// Get info about a single volume
func (d *VolumeDriver) Get(r volume.Request) volume.Response {
	log.Errorf("VolumeDriver Get to be implemented")
	return volume.Response{Err: ""}
}

// List volumes known to the driver
func (d *VolumeDriver) List(r volume.Request) volume.Response {
	log.Errorf("VolumeDriver List to be implemented")
	return volume.Response{Err: ""}
}

// GetVolume - return volume meta-data.
func (d *VolumeDriver) GetVolume(name string) (map[string]interface{}, error) {
	var statusMap map[string]interface{}
	statusMap = make(map[string]interface{})
	log.Errorf("VolumeDriver GetVolume to be implemented")
	return statusMap, nil
}

// MountVolume - Request attach and them mounts the volume.
func (d *VolumeDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	log.Errorf("VolumeDriver MountVolume to be implemented")
	mountpoint := getMountPoint(name)
	return mountpoint, nil
}

// UnmountVolume - Unmounts the volume and then requests detach
func (d *VolumeDriver) UnmountVolume(name string) error {
	log.Errorf("VolumeDriver UnmountVolume to be implemented")
	return nil
}

// Create - create a volume.
func (d *VolumeDriver) Create(r volume.Request) volume.Response {
	log.Errorf("VolumeDriver Create to be implemented")
	return volume.Response{Err: ""}
}

// Remove - removes individual volume. Docker would call it only if is not using it anymore
func (d *VolumeDriver) Remove(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Removing volume ")

	// Cannot remove volumes till plugin completely initializes (refcounting is complete)
	// because we don't know if it is being used or not
	if d.refCounts.IsInitialized() != true {
		msg := fmt.Sprintf(plugin_utils.PluginInitError+" Cannot remove volume=%s", r.Name)
		log.Error(msg)
		return volume.Response{Err: msg}
	}

	// Docker is supposed to block 'remove' command if the volume is used.
	if d.getRefCount(r.Name) != 0 {
		msg := fmt.Sprintf("Remove failure - volume is still mounted. "+
			" volume=%s, refcount=%d", r.Name, d.getRefCount(r.Name))
		log.Error(msg)
		return volume.Response{Err: msg}
	}

	log.Errorf("VolumeDriver Remove to be implemented")
	return volume.Response{Err: ""}
}

// Path - give docker a reminder of the volume mount path
func (d *VolumeDriver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: getMountPoint(r.Name)}
}

// Mount - Provide a volume to docker container - called once per container start.
// We need to keep refcount and unmount on refcount drop to 0
//
// The serialization of operations per volume is assured by the volume/store
// of the docker daemon.
// As long as the refCountsMap is protected is unnecessary to do any locking
// at this level during create/mount/umount/remove.
//
func (d *VolumeDriver) Mount(r volume.MountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")

	// lock the state
	d.refCounts.StateMtx.Lock()
	defer d.refCounts.StateMtx.Unlock()

	log.Errorf("VolumeDriver Mount to be implemented")
	return volume.Response{Err: ""}
}

// Unmount request from Docker. If mount refcount is drop to 0.
func (d *VolumeDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")

	// lock the state
	d.refCounts.StateMtx.Lock()
	defer d.refCounts.StateMtx.Unlock()

	if d.refCounts.IsInitialized() != true {
		// if refcounting hasn't been succesful,
		// no refcounting, no unmount. All unmounts are delayed
		// until we succesfully populate the refcount map
		d.refCounts.MarkDirty()
		return volume.Response{Err: ""}
	}

	log.Errorf("VolumeDriver Unmount to be implemented")
	return volume.Response{Err: ""}
}

// Capabilities - Report plugin scope to Docker
func (d *VolumeDriver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}
