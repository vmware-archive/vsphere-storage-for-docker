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
	m       *sync.Mutex // create() serialization - for future use
	mockEsx bool
	ops     vmdkops.VmdkOps
}

var (
	// volume name -> refcount , for volumes with refcount > 0
	refcounts = make(map[string]int)
)

// creates vmdkDriver which may talk to real ESX (mockEsx=False) or
// real ESX.
func newVmdkDriver(mockEsx bool) vmdkDriver {
	var vmdkCmd vmdkops.VmdkCmdRunner
	if mockEsx {
		vmdkCmd = vmdkops.MockVmdkCmd{}
	} else {
		vmdkCmd = vmdkops.VmdkCmd{}
	}
	d := vmdkDriver{
		m:       &sync.Mutex{},
		mockEsx: mockEsx,
		ops:     vmdkops.VmdkOps{Cmd: vmdkCmd},
	}
	return d
}

func (d vmdkDriver) Get(r volume.Request) volume.Response {
	_, err := d.ops.Get(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	mountpoint := filepath.Join(mountRoot, r.Name)
	return volume.Response{Volume: &volume.Volume{Name: r.Name, Mountpoint: mountpoint}}
}

func (d vmdkDriver) List(r volume.Request) volume.Response {
	volumes, err := d.ops.List()
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	responseVolumes := make([]*volume.Volume, 0, len(volumes))
	for _, vol := range volumes {
		mountpoint := filepath.Join(mountRoot, vol.Name)
		responseVol := volume.Volume{Name: vol.Name, Mountpoint: mountpoint}
		responseVolumes = append(responseVolumes, &responseVol)
	}
	return volume.Response{Volumes: responseVolumes}
}

// request attach and them mounts the volume
// actual mount - send attach to ESX and do the in-guest magix
// TODO: this should actually be a goroutine , no need to block
//       SAME (and more) applies to unmount
func (d vmdkDriver) mountVolume(r volume.Request, path string) error {
	// First of all, have ESX attach the disk
	if err := d.ops.Attach(r.Name, r.Options); err != nil {
		return err
	}

	mountpoint := filepath.Join(mountRoot, r.Name)
	if d.mockEsx {
		return fs.Mount(mountpoint, r.Name, "ext4")
	}
	return fs.Mount(mountpoint, r.Name, "ext2")
}

// Unmounts the volume and then requests detach
func (d vmdkDriver) unmountVolume(r volume.Request) error {
	mountpoint := filepath.Join(mountRoot, r.Name)
	err := fs.Unmount(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"mountpoint": mountpoint, "error": err},
		).Error("Failed to unmount volume. Still trying to detach. ")
		// Do not return error. Continue with detach.
	}
	return d.ops.Detach(r.Name, r.Options)
}

// The user wants to create a volume.
// No need to actually manifest the volume on the filesystem yet
// (until Mount is called).
// Name and driver specific options passed through to the ESX host
func (d vmdkDriver) Create(r volume.Request) volume.Response {
	err := d.ops.Create(r.Name, r.Options)
	if err != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": err}).Error("Create volume failed ")
		return volume.Response{Err: err.Error()}
	}
	log.WithFields(log.Fields{"name": r.Name}).Info("Volume created ")
	return volume.Response{Err: ""}
}

func (d vmdkDriver) Remove(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Removing volume ")

	// Docker is supposed to block 'remove' command if the volume is used. Verify.
	if refcounts[r.Name] != 0 {
		msg := fmt.Sprintf("Remove faiure - volume is still mounted. " +
		    " volume=%s, refcount=%d", r.Name, refcounts[r.Name])
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
func (d vmdkDriver) Path(r volume.Request) volume.Response {
	m := filepath.Join(mountRoot, r.Name)
	return volume.Response{Mountpoint: m}
}

// Provide a volume to docker container - called once per container start.
// We need to keep refcount and unmount on refcount drop to 0
func (d vmdkDriver) Mount(r volume.Request) volume.Response {
	d.m.Lock()
	defer d.m.Unlock()
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")

	m := filepath.Join(mountRoot, r.Name)

	// If the volume is already mounted , just increase the refcount.
	//
	// Note: We are deliberately incrementing refcount first, before trying
	// to do anything else. If Mount fails, Docker will send Unmount request,
	// and we will happily decrement the refcount there, and will fail the unmount
	// since the volume will have been never mounted.
	// Note: for new keys, GO maps return zero value, so no need for if_exists.

	refcounts[r.Name]++
	refcnt := refcounts[r.Name] // save map traversal
	if refcnt > 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: m}
	}

	// Make sure  that mountpoint exists.
	err := fs.Mkdir(m)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "dir": m},
		).Error("Failed to make directory for volume mount")
		return volume.Response{Err: err.Error()}
	}

	// This is the first time we are asked to mount the volume, so comply
	if err := d.mountVolume(r, m); err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to mount ")
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: m}
}

// Unmount request from Docker. If mount refcount is drop to 0,
// unmount and detach from VM
func (d vmdkDriver) Unmount(r volume.Request) volume.Response {
	d.m.Lock()
	defer d.m.Unlock()
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")

	// if the volume is still used by other containers, just return OK
	refcounts[r.Name]--
	refcnt := refcounts[r.Name] // save map traversal
	if refcnt >= 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Still in use, skipping unmount request. ")
		return volume.Response{Err: ""}
	}
	// More "unmounts" than "mounts" receied. Yell, reset to 0 and keep going
	if refcnt != 0 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Error("WRONG REF COUNT in mount (should be 0). Resetting to 0. ")
	}
	delete(refcounts, r.Name)

	// and if nobody needs it, unmount and detach

	err := d.unmountVolume(r)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to unmount ")
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Err: ""}
}
