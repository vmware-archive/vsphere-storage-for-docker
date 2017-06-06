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

package vsphere

//
// VMWare vSphere Docker Data Volume plugin.
//
// Provide support for --driver=vsphere in Docker, when Docker VM is running under ESX.
//
// Serves requests from Docker Engine related to VMDK volume operations.
// Depends on vmdk-opsd service to be running on hosting ESX
// (see ./esx_service)
///

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/drivers/network"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/drivers/vmdk"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/plugin_utils"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/refcount"
	"strings"
)

const (
	version  = "vSphere Volume Driver v0.4"
	volType  = "type"
	vmdkImpl = "vmdk"
	nfsImpl  = "nfs"
)

// VolumeDriver - vSphere driver struct
type VolumeDriver struct {
	refCounts      *refcount.RefCountsMap
	mountedVolumes map[string]string
	mountIDToName  map[string]string
	config         config.Config
}

// volumeBackingMap - Maps FS type to implementing driver object
var volumeBackingMap map[string]VolumeImpl

func (d *VolumeDriver) getVolImplWithType(name string) (VolumeImpl, string) {
	// If the volume is mounted then get the backing for
	// it from the mounted volumes map.
	if fstype, ok := d.mountedVolumes[name]; ok {
		return volumeBackingMap[fstype], fstype
	}

	// Else figure the FS type for the label and use the
	// volume impl for that.
	dslabel := plugin_utils.GetDSLabel(name)
	if dslabel != "" && len(d.config.RemoteDirs.RemoteDirTbl) > 0 {
		if rdir, ok := d.config.RemoteDirs.RemoteDirTbl[dslabel]; ok {
			return volumeBackingMap[rdir.FSType], rdir.FSType
		}
	}
	// Default to a VMDK type volume
	return volumeBackingMap[vmdkImpl], vmdkImpl
}

// NewVolumeDriver creates Driver which to real ESX (useMockEsx=False) or a mock
func NewVolumeDriver(port int, useMockEsx bool, mountDir string, driverName string, configFile string) *VolumeDriver {
	var d *VolumeDriver

	d = new(VolumeDriver)
	c, err := config.Load(configFile)
	if err != nil {
		log.Warning("Failed to load config file - ", configFile)
		return nil
	}
	d.config = c

	// Init all known backends - VMDK and network volume drivers
	d.mountedVolumes = make(map[string]string)
	d.mountIDToName = make(map[string]string)
	vimpl, err := vmdk.Init(port, useMockEsx, mountDir)
	if err != nil {
		return nil
	}
	volumeBackingMap[vmdkImpl] = vimpl

	nimpl, err := network.Init(d, mountDir, d.config)
	if err != nil {
		return nil
	}
	volumeBackingMap[nfsImpl] = nimpl

	d.refCounts = refcount.NewRefCountsMap()
	d.refCounts.Init(d, mountDir, driverName)

	return d
}

// Return the number of references for the given volume
func (d *VolumeDriver) GetRefCount(vol string) uint { return d.refCounts.GetCount(vol) }

// Increment the reference count for the given volume
func (d *VolumeDriver) incrRefCount(vol string) uint { return d.refCounts.Incr(vol) }

// Decrement the reference count for the given volume
func (d *VolumeDriver) decrRefCount(vol string) (uint, error) { return d.refCounts.Decr(vol) }

// Get info about a single volume
func (d *VolumeDriver) Get(r volume.Request) volume.Response {
	volImpl, _ := d.getVolImplWithType(r.Name)
	return volImpl.Get(r)
}

// List volumes known to the driver
func (d *VolumeDriver) List(r volume.Request) volume.Response {
	// Get and append volumes from the two backing types
	responseVolumes, err := volumeBackingMap[vmdkImpl].List(r)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	filVols, err := volumeBackingMap[nfsImpl].List(r)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	responseVolumes = append(responseVolumes, filVols...)
	return volume.Response{Volumes: responseVolumes}
}

// Create - create a volume.
func (d *VolumeDriver) Create(r volume.Request) volume.Response {
	// For file type volume the network driver handles any
	// addition opts that specify the exported fs to create
	// the volume. Later change this to match against the known
	// volume backing types to figure which one to use. For now
	// the only two types are "nfs" and "vmdk".
	if ftype, ok := r.Options[volType]; ok == true && strings.Contains(ftype, nfsImpl) {
		return volumeBackingMap[nfsImpl].Create(r)
	}

	// If a DS label was specified the use the volume impl
	// for the type associated with DS label.
	dslabel := plugin_utils.GetDSLabel(r.Name)
	if dslabel != "" && len(d.config.RemoteDirs.RemoteDirTbl) > 0 {
		// Default to the one remote FS type thats supported now
		rdir, ok := d.config.RemoteDirs.RemoteDirTbl[dslabel]
		if ok && strings.Contains(rdir.FSType, nfsImpl) {
			return volumeBackingMap[nfsImpl].Create(r)
		}
	}
	// If volume doesn't have a label or not a remote dir.
	return volumeBackingMap[vmdkImpl].Create(r)
}

// Remove - removes individual volume. Docker would call it only if is not using it anymore
func (d *VolumeDriver) Remove(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Removing volume ")
	// Docker is supposed to block 'remove' command if the volume is used. Verify.
	if d.GetRefCount(r.Name) != 0 {
		msg := fmt.Sprintf("Remove failure - volume is still in use, "+
			" volume=%s, refcount=%d", r.Name, d.GetRefCount(r.Name))
		log.Error(msg)
		return volume.Response{Err: msg}
	}
	volImpl, _ := d.getVolImplWithType(r.Name)
	return volImpl.Remove(r)
}

// Path - give docker a reminder of the volume mount path
func (d *VolumeDriver) Path(r volume.Request) volume.Response {
	volImpl, _ := d.getVolImplWithType(r.Name)
	return volImpl.Path(r)
}

// Mount - Provide a volume to docker container - called once per container start.
func (d *VolumeDriver) Mount(r volume.MountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")

	volImpl, fstype := d.getVolImplWithType(r.Name)

	// lock the state
	d.refCounts.LockStateLock()
	defer d.refCounts.UnlockStateLock()

	// Checked by refcounting thread until refmap initialized
	d.refCounts.SetDirty()

	// Get the full name for the named volume
	volInfo, err := plugin_utils.GetVolumeInfo(r.Name, "", d)
	if err != nil {
		log.Errorf("Unable to get volume info for volume %s. err:%v", r.Name, err)
		return volume.Response{Err: err.Error()}
	}

	fname := volInfo.VolumeName

	// If the volume is already mounted , increase the refcount.
	// Note: for new keys, GO maps return zero value, so no need for if_exists.
	refcnt := d.incrRefCount(fname) // save map traversal
	log.Debugf("volume name=%s refcnt=%d", fname, refcnt)

	if refcnt > 1 || volImpl.IsMounted(fname) {
		log.WithFields(
			log.Fields{"name": fname, "refcount": refcnt},
		).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: volImpl.GetMountPoint(fname)}
	}

	response := volImpl.Mount(r, volInfo)
	if response.Err != "" {
		d.decrRefCount(fname)
		d.refCounts.ClearDirty()
		delete(d.mountedVolumes, fname)
	} else {
		// Add to the mounted volumes map
		d.mountedVolumes[fname] = fstype
		d.mountIDToName[r.ID] = fname

	}

	return response
}

// Unmount request from Docker. If mount refcount is drop to 0.
// Unmount and detach from VM
func (d *VolumeDriver) Unmount(r volume.UnmountRequest) volume.Response {
	var fname string

	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")
	volImpl, _ := d.getVolImplWithType(r.Name)

	// lock the state
	d.refCounts.LockStateLock()
	defer d.refCounts.UnlockStateLock()

	// Get the fully qualified name of the volume, the mount
	// may have happened before the plugin restarted, so check its
	// ok to use, else fetch volume meta-data and the full name
	fname, ok := d.mountIDToName[r.ID]
	if !ok {
		// Get the full name for the named volume
		volInfo, err := plugin_utils.GetVolumeInfo(r.Name, "", d)
		if err != nil {
			log.Errorf("Unable to get volume info for volume %s. err:%v", r.Name, err)
			return volume.Response{Err: err.Error()}
		}

		fname = volInfo.VolumeName
	} else {
		delete(d.mountIDToName, r.ID)
	}

	if d.refCounts.GetInitSuccess() != true {
		// If refcounting hasn't been succesful,
		// no refcounting, no unmount. All unmounts are delayed
		// until we succesfully populate the refcount map
		d.refCounts.SetDirty()
		delete(d.mountedVolumes, fname)

		return volume.Response{Err: ""}
	}

	// if refcount has been succcessful, Normal flow
	// if the volume is still used by other containers, just return OK
	refcnt, err := d.decrRefCount(fname)
	if err != nil {
		// something went wrong - yell, but still try to unmount
		log.WithFields(
			log.Fields{"name": fname, "refcount": refcnt},
		).Error("Refcount error - still trying to unmount...")
	}

	log.Debugf("volume name=%s refcnt=%d", fname, refcnt)
	if refcnt >= 1 {
		log.WithFields(
			log.Fields{"name": fname, "refcount": refcnt},
		).Info("Still in use, skipping unmount request. ")
		return volume.Response{Err: ""}
	}

	// Remove from the mounted volumes map
	delete(d.mountedVolumes, fname)
	return volImpl.Unmount(volume.UnmountRequest{Name: fname})
}

// Capabilities - Report plugin scope to Docker
func (d *VolumeDriver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}

// MountVolume - mount a volume without reference counting, name
// is the fully qualified name of the volume (volume@ds).
func (d *VolumeDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	volImpl, fs := d.getVolImplWithType(name)

	// If mounting via the refcounter then create the entry in the
	// mountedVolumes map so the next mount to the same volume finds it
	d.mountedVolumes[name] = fs

	res, err := volImpl.MountVolume(name, fstype, id, isReadOnly, skipAttach)
	return res, err
}

// UnmountVolume - unmount a volume without reference counting, name
// is the fully qualified name of the volume (volume@ds).
func (d *VolumeDriver) UnmountVolume(name string) error {
	volImpl, _ := d.getVolImplWithType(name)

	// Remove from the map as no one is using the volume
	if name, exist := d.mountedVolumes[name]; exist {
		delete(d.mountedVolumes, name)
	}

	return volImpl.UnmountVolume(name)

}

// GetVolume - get volume data.
func (d *VolumeDriver) GetVolume(name string) (map[string]interface{}, error) {
	volImpl, _ := d.getVolImplWithType(name)
	return volImpl.GetVolume(name)
}
