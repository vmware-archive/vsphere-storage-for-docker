package main

//
// VMWare VMDK Docker Data Volume plugin.
//
// Provide suport for --driver=vmdk in Docker, when Docker VM is running under ESX.
//
// Serves requests from Docker Engine related to VMDK volume operations.
// Depends on vmdk-opsd service to be running on hosting ESX
// (see ./vmdkops-esxsrv)
///

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-vmdk-plugin/fs"
	"github.com/vmware/docker-vmdk-plugin/vmdkops"
	"path/filepath"
	"sync"
)

const (
	mountRoot = "/mnt/vmdk" // VMDK block devices are mounted here
)

type vmdkDriver struct {
	m          *sync.Mutex // create() serialization - for future use
	useMockEsx bool
	ops        vmdkops.VmdkOps
	refCounts  refCountsMap
}

// creates vmdkDriver which to real ESX (useMockEsx=False) or a mock
func newVmdkDriver(useMockEsx bool) *vmdkDriver {
	var d *vmdkDriver
	if useMockEsx {
		d = &vmdkDriver{
			m:          &sync.Mutex{},
			useMockEsx: true,
			ops:        vmdkops.VmdkOps{Cmd: vmdkops.MockVmdkCmd{}},
		}
	} else {
		d = &vmdkDriver{
			m:          &sync.Mutex{},
			useMockEsx: false,
			ops:        vmdkops.VmdkOps{Cmd: vmdkops.EsxVmdkCmd{}},
			refCounts:  make(refCountsMap),
		}
		d.refCounts.Init(d)
	}

	return d
}
func (d *vmdkDriver) getRefCount(vol string) uint  { return d.refCounts.getCount(vol) }
func (d *vmdkDriver) incrRefCount(vol string) uint { return d.refCounts.incr(vol) }
func (d *vmdkDriver) decrRefCount(vol string) (uint, error) { return d.refCounts.decr(vol) }

func getMountPoint(volName string) string {
	return filepath.Join(mountRoot, volName)

}

func (d *vmdkDriver) Get(r volume.Request) volume.Response {
	_, err := d.ops.Get(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	mountpoint := getMountPoint(r.Name)
	return volume.Response{Volume: &volume.Volume{Name: r.Name, Mountpoint: mountpoint}}
}

func (d *vmdkDriver) List(r volume.Request) volume.Response {
	volumes, err := d.ops.List()
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	responseVolumes := make([]*volume.Volume, 0, len(volumes))
	for _, vol := range volumes {
		mountpoint := getMountPoint(vol.Name)
		responseVol := volume.Volume{Name: vol.Name, Mountpoint: mountpoint}
		responseVolumes = append(responseVolumes, &responseVol)
	}
	return volume.Response{Volumes: responseVolumes}
}

// request attach and them mounts the volume
// actual mount - send attach to ESX and do the in-guest magix
// returns mount point and  error (or nil)
func (d *vmdkDriver) mountVolume(name string) (string, error) {
	mountpoint := getMountPoint(name)

	// First, make sure  that mountpoint exists.
	err := fs.Mkdir(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"name": name, "dir": mountpoint},
		).Error("Failed to make directory for volume mount ")
		return mountpoint, err
	}

	// Have ESX attach the disk
	dev, err := d.ops.Attach(name, nil)
	if err != nil {
		return mountpoint, err
	}

	if d.useMockEsx {
		return mountpoint, fs.Mount(mountpoint, nil, "ext4")
	}

	return mountpoint, fs.Mount(mountpoint, dev, "ext2")
}

// Unmounts the volume and then requests detach
func (d *vmdkDriver) unmountVolume(name string) error {
	mountpoint := getMountPoint(name)
	err := fs.Unmount(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"mountpoint": mountpoint, "error": err},
		).Error("Failed to unmount volume. Now trying to detach... ")
		// Do not return error. Continue with detach.
	}
	return d.ops.Detach(name, nil)
}

// The user wants to create a volume.
// No need to actually manifest the volume on the filesystem yet
// (until Mount is called).
// Name and driver specific options passed through to the ESX host
func (d *vmdkDriver) Create(r volume.Request) volume.Response {
	err := d.ops.Create(r.Name, r.Options)
	if err != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": err}).Error("Create volume failed ")
		return volume.Response{Err: err.Error()}
	}
	log.WithFields(log.Fields{"name": r.Name}).Info("Volume created ")
	return volume.Response{Err: ""}
}

func (d *vmdkDriver) Remove(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Removing volume ")

	// Docker is supposed to block 'remove' command if the volume is used. Verify.
	if d.getRefCount(r.Name) != 0 {
		msg := fmt.Sprintf("Remove faiure - volume is still mounted. "+
			" volume=%s, refcount=%d", r.Name, d.getRefCount(r.Name))
		log.Error(msg)
		return volume.Response{Err: msg}
	}

	err := d.ops.Remove(r.Name, r.Options)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err},
		).Error("Failed to remove volume ")
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Err: ""}
}

// give docker a reminder of the volume mount path
func (d *vmdkDriver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: getMountPoint(r.Name)}
}

// Provide a volume to docker container - called once per container start.
// We need to keep refcount and unmount on refcount drop to 0
func (d *vmdkDriver) Mount(r volume.Request) volume.Response {
	d.m.Lock()
	defer d.m.Unlock()
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")

	// If the volume is already mounted , just increase the refcount.
	//
	// Note: We are deliberately incrementing refcount first, before trying
	// to do anything else. If Mount fails, Docker will send Unmount request,
	// and we will happily decrement the refcount there, and will fail the unmount
	// since the volume will have been never mounted.
	// Note: for new keys, GO maps return zero value, so no need for if_exists.

	refcnt := d.incrRefCount(r.Name) // save map traversal
	if refcnt > 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: getMountPoint(r.Name)}
	}

	// This is the first time we are asked to mount the volume, so comply
	mountpoint, err := d.mountVolume(r.Name)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to mount ")
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: mountpoint}
}

// Unmount request from Docker. If mount refcount is drop to 0,
// unmount and detach from VM
func (d *vmdkDriver) Unmount(r volume.Request) volume.Response {
	d.m.Lock()
	defer d.m.Unlock()
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")

	// if the volume is still used by other containers, just return OK
	refcnt, err := d.decrRefCount(r.Name)
	if err != nil {
		// something went wrong - yell, but still try to unmount
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Error("Refcount error - still trying to unmount...")
	}
	if refcnt >= 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Still in use, skipping unmount request. ")
		return volume.Response{Err: ""}
	}

	// and if nobody needs it, unmount and detach
	err = d.unmountVolume(r.Name)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to unmount ")
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Err: ""}
}
