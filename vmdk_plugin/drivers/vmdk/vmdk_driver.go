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

package vmdk

//
// VMDK VolumeImpl Driver.
//
// Provide support for VMDK backed volumes, when Docker VM is running under ESX.
//
// Serves requests from Docker Engine related to VMDK volume operations.
// Depends on vmdk-opsd service to be running on hosting ESX
// (see ./esx_service)
//

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/drivers/vmdk/vmdkops"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/fs"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/plugin_utils"
)

const (
	devWaitTimeout   = 1 * time.Second
	sleepBeforeMount = 1 * time.Second
	watchPath        = "/dev/disk/by-path"
)

// VolumeImplDriver - VMDK driver struct
type VolumeImplDriver struct {
	useMockEsx bool
	ops        vmdkops.VmdkOps
}

var mountRoot string

// Init creates Driver which to real ESX (useMockEsx=False) or a mock
func Init(port int, useMockEsx bool, mountDir string) (*VolumeImplDriver, error) {
	var d *VolumeImplDriver

	vmdkops.EsxPort = port
	mountRoot = mountDir

	if useMockEsx {
		d = &VolumeImplDriver{
			useMockEsx: true,
			ops:        vmdkops.VmdkOps{Cmd: vmdkops.MockVmdkCmd{}},
		}
	} else {
		d = &VolumeImplDriver{
			useMockEsx: false,
			ops: vmdkops.VmdkOps{
				Cmd: vmdkops.EsxVmdkCmd{
					Mtx: &sync.Mutex{},
				},
			},
		}
	}

	log.WithFields(log.Fields{
		"port":     vmdkops.EsxPort,
		"mock_esx": useMockEsx,
	}).Info("Docker VMDK plugin started ")

	return d, nil
}

// GetMountPoint - Returns the given volume mountpoint
func (d *VolumeImplDriver) GetMountPoint(volName string) string {
	return filepath.Join(mountRoot, volName)
}

// Get info about a single volume
func (d *VolumeImplDriver) Get(r volume.Request) volume.Response {
	status, err := d.GetVolume(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	mountpoint := d.GetMountPoint(r.Name)
	return volume.Response{Volume: &volume.Volume{Name: r.Name,
		Mountpoint: mountpoint,
		Status:     status}}
}

// List volumes known to the driver
func (d *VolumeImplDriver) List(r volume.Request) ([]*volume.Volume, error) {
	volumes, err := d.ops.List()
	if err != nil {
		return nil, err
	}
	responseVolumes := make([]*volume.Volume, 0, len(volumes))
	for _, vol := range volumes {
		mountpoint := d.GetMountPoint(vol.Name)
		responseVol := volume.Volume{Name: vol.Name, Mountpoint: mountpoint}
		responseVolumes = append(responseVolumes, &responseVol)
	}
	return responseVolumes, nil
}

// GetVolume - return volume meta-data.
func (d *VolumeImplDriver) GetVolume(name string) (map[string]interface{}, error) {
	return d.ops.Get(name)
}

// MountVolume - Request attach and them mounts the volume.
// Actual mount - send attach to ESX and do the in-guest magic
// Returns mount point and  error (or nil)
func (d *VolumeImplDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	mountpoint := d.GetMountPoint(name)

	// First, make sure  that mountpoint exists.
	err := fs.Mkdir(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"name": name, "dir": mountpoint},
		).Error("Failed to make directory for volume mount ")
		return mountpoint, err
	}

	watcher, skipInotify := fs.DevAttachWaitPrep(name, watchPath)

	// Have ESX attach the disk
	dev, err := d.ops.Attach(name, nil)
	if err != nil {
		return mountpoint, err
	}

	if d.useMockEsx {
		return mountpoint, fs.Mount(mountpoint, fstype, string(dev[:]), false)
	}

	device, err := fs.GetDevicePath(dev)
	if err != nil {
		return mountpoint, err
	}

	if skipInotify {
		time.Sleep(sleepBeforeMount)
		return mountpoint, fs.Mount(mountpoint, fstype, device, false)
	}

	fs.DevAttachWait(watcher, name, device)

	// May have timed out waiting for the attach to complete,
	// attempt the mount anyway.
	return mountpoint, fs.Mount(mountpoint, fstype, device, isReadOnly)
}

// UnmountVolume - Unmounts the volume and then requests detach
func (d *VolumeImplDriver) UnmountVolume(name string) error {
	mountpoint := d.GetMountPoint(name)
	err := fs.Unmount(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"mountpoint": mountpoint, "error": err},
		).Error("Failed to unmount volume. Now trying to detach... ")
		// Do not return error. Continue with detach.
	}
	return d.ops.Detach(name, nil)
}

// No need to actually manifest the volume on the filesystem yet
// (until Mount is called).
// Name and driver specific options passed through to the ESX host

// Create - create a volume.
func (d *VolumeImplDriver) Create(r volume.Request) volume.Response {

	if r.Options == nil {
		r.Options = make(map[string]string)
	}
	// If cloning a existent volume, create and return
	if _, result := r.Options["clone-from"]; result == true {
		errClone := d.ops.Create(r.Name, r.Options)
		if errClone != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": errClone}).Error("Clone volume failed ")
			return volume.Response{Err: errClone.Error()}
		}
		return volume.Response{Err: ""}
	}

	// Use default fstype if not specified
	if _, result := r.Options["fstype"]; result == false {
		r.Options["fstype"] = fs.FstypeDefault
	}

	// Get existent filesystem tools
	supportedFs := fs.MkfsLookup()

	// Verify the existence of fstype mkfs
	mkfscmd, result := supportedFs[r.Options["fstype"]]
	if result == false {
		msg := "Not found mkfs for " + r.Options["fstype"]
		msg += "\nSupported filesystems found: "
		validfs := ""
		for fs := range supportedFs {
			if validfs != "" {
				validfs += ", " + fs
			} else {
				validfs += fs
			}
		}
		log.WithFields(log.Fields{"name": r.Name,
			"fstype": r.Options["fstype"]}).Error("Not found ")
		return volume.Response{Err: msg + validfs}
	}

	errCreate := d.ops.Create(r.Name, r.Options)
	if errCreate != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errCreate}).Error("Create volume failed ")
		return volume.Response{Err: errCreate.Error()}
	}

	// Handle filesystem creation
	log.WithFields(log.Fields{"name": r.Name,
		"fstype": r.Options["fstype"]}).Info("Attaching volume and creating filesystem ")

	watcher, skipInotify := fs.DevAttachWaitPrep(r.Name, watchPath)

	dev, errAttach := d.ops.Attach(r.Name, nil)
	if errAttach != nil {
		log.WithFields(log.Fields{"name": r.Name,
			"error": errAttach}).Error("Attach volume failed, removing the volume ")
		// An internal error for the attach may have the volume attached to this client,
		// detach before removing below.
		d.ops.Detach(r.Name, nil)
		errRemove := d.ops.Remove(r.Name, nil)
		if errRemove != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": errRemove}).Warning("Remove volume failed ")
		}
		return volume.Response{Err: errAttach.Error()}
	}

	device, errGetDevicePath := fs.GetDevicePath(dev)
	if errGetDevicePath != nil {
		log.WithFields(log.Fields{"name": r.Name,
			"error": errGetDevicePath}).Error("Could not find attached device, removing the volume ")
		errDetach := d.ops.Detach(r.Name, nil)
		if errDetach != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": errDetach}).Warning("Detach volume failed ")
		}
		errRemove := d.ops.Remove(r.Name, nil)
		if errRemove != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": errRemove}).Warning("Remove volume failed ")
		}
		return volume.Response{Err: errGetDevicePath.Error()}
	}

	if skipInotify {
		time.Sleep(sleepBeforeMount)
	} else {
		// Wait for the attach to complete, may timeout
		// in which case we continue creating the file system.
		fs.DevAttachWait(watcher, r.Name, device)
	}
	errMkfs := fs.Mkfs(mkfscmd, r.Name, device)
	if errMkfs != nil {
		log.WithFields(log.Fields{"name": r.Name,
			"error": errMkfs}).Error("Create filesystem failed, removing the volume ")
		errDetach := d.ops.Detach(r.Name, nil)
		if errDetach != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": errDetach}).Warning("Detach volume failed ")
		}
		errRemove := d.ops.Remove(r.Name, nil)
		if errRemove != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": errRemove}).Warning("Remove volume failed ")
		}
		return volume.Response{Err: errMkfs.Error()}
	}

	errDetach := d.ops.Detach(r.Name, nil)
	if errDetach != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errDetach}).Error("Detach volume failed ")
		return volume.Response{Err: errDetach.Error()}
	}

	log.WithFields(log.Fields{"name": r.Name,
		"fstype": r.Options["fstype"]}).Info("Volume and filesystem created ")
	return volume.Response{Err: ""}
}

// Remove - removes individual volume. Docker would call it only if is not using it anymore
func (d *VolumeImplDriver) Remove(r volume.Request) volume.Response {
	err := d.ops.Remove(r.Name, r.Options)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err},
		).Error("Failed to remove volume ")
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Err: ""}

}

// Path - give docker a reminder of the volume mount path
func (d *VolumeImplDriver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: d.GetMountPoint(r.Name)}
}

// Mount - Provide a volume to docker container - called once per container start.
// We need to keep refcount and unmount on refcount drop to 0
//
// The serialization of operations per volume is assured by the volume/store
// of the docker daemon.
// As long as the refCountsMap is protected is unnecessary to do any locking
// at this level during create/mount/umount/remove.
//
func (d *VolumeImplDriver) Mount(r volume.MountRequest, volInfo *plugin_utils.VolumeInfo) volume.Response {
	fname := volInfo.VolumeName
	log.WithFields(log.Fields{"name": fname}).Info("Mounting volume ")

	// get volume metadata if required
	volumeMeta := volInfo.VolumeMeta
	var err error
	if volumeMeta == nil {
		if volumeMeta, err = d.ops.Get(fname); err != nil {
			return volume.Response{Err: err.Error()}
		}
	}

	fstype := fs.FstypeDefault
	isReadOnly := false

	// Check access type.
	value, exists := volumeMeta["access"].(string)
	if !exists {
		msg := fmt.Sprintf("Invalid access type for %s, assuming read-write access.", fname)
		log.WithFields(log.Fields{"name": fname, "error": msg}).Error("")
		isReadOnly = false
	} else if value == "read-only" {
		isReadOnly = true
	}

	// Check file system type.
	value, exists = volumeMeta["fstype"].(string)
	if !exists {
		msg := fmt.Sprintf("Invalid filesystem type for %s, assuming type as %s.",
			fname, fstype)
		log.WithFields(log.Fields{"name": fname, "error": msg}).Error("")
		// Fail back to a default version that we can try with.
		value = fs.FstypeDefault
	}
	fstype = value

	mountpoint, err := d.MountVolume(fname, fstype, "", isReadOnly, false)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to mount ")
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: mountpoint}
}

// Unmount request from Docker. If mount refcount is drop to 0.
// Unmount and detach from VM
func (d *VolumeImplDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")

	// Nobody needs it, unmount and detach
	err := d.UnmountVolume(r.Name)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to unmount ")
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Err: ""}
}

// IsMounted figures if the named volume is mounted under the root
// used by this volume driver
func (d *VolumeImplDriver) IsMounted(name string) bool {
	return plugin_utils.IsMounted(name, mountRoot)
}
