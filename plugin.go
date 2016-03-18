package main

//
// A VMDK Docker Data Volume plugin
// (for now just an outline)
//
// It checks that it can connect to ESX via vSocket , and then
// fakes the successful ops by returning success and printing a messafe,
// without doing anything useful (useful stuff is work in progress)
//
// TBD: add check for the guest communicates from ROOT - check the doc wrt security
// TBD: check we are running inside ESX and tools installed - in CODE , not make

// TODO:
//  convert all to return error() and only response to docker to return msg
//
// make sure msg and err is properly initialized everywhere (and check in go if it's needed)
///
//TODO :
// Potentially: add unit test , fully contained on 1 Linux machine:
//- make server code actually create a loop block device and do a bind mount for volume test
//- hardcode location for testing (/var/vmware/dvolplug/volumes/vol-name
//	fallocate -d -l <length> volume.img
//	losetup -v /dev/loop$(id) -f volume.img
//	mkfs.ext4 /dev/loop$(id)
//	mount /dev/lopp$(id) /mnt/loop$(id) # mount all here. Or skip
//	mount -o bind # bind mount where Docker asks
//	Good refs: https://www.suse.com/communities/blog/accessing-file-systems-disk-block-image-files/
//)
// for actual mounts:
//- add actual create/mount code instead of prints
//
// TODO: add volumes tracking per the following docker spec:
//
// multiple containers on the same docker engine may use the same vmdk volume
// thus we need to track the volumes already attached, and just do bind mount for them
// Also it means we need to serialize all ops with mutex

// We also need to track volumes attached and mounted to save on this ops if requested
// On start, we need to list vmdks attached to the VM and polulate list of volumes from it

import (
	//	"encoding/json"
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
	m       *sync.Mutex // create() serialization
	mockEsx bool
	ops     vmdkops.VmdkOps
}

func newVmdkDriver(mockEsx bool) vmdkDriver {
	var vmdkCmd vmdkops.VmdkCmdRunner
	if mockEsx {
		vmdkCmd = vmdkops.MockVmdkCmd{}
	} else {
		vmdkCmd = vmdkops.VmdkCmd{}
	}
	d := vmdkDriver{
		// TODO: volumes map(string)volinfo, (name->uuid/refcount/creationtime)
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

	refCount := 0
	if refCount != 0 { // TODO: actual refcounting
		return nil
	}

	// TODO: refcount  if the volume is already mounted (for other container) and
	// just return volume.Response{Mountpoint: m} in this case
	// TODO: save info abouf voliume mount , in memory
	// d.volumes[m] = &volumeName{name: r.Name, connections: 1}
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
		log.WithFields(log.Fields{"error": err}).Info("Unmount failed")
		return fmt.Errorf("Unmount failed: %T", err)
	}
	log.WithFields(log.Fields{"name": r.Name, "options": r.Options}).Info("Detach Volume")
	return d.ops.Detach(r.Name, r.Options)
}

// The user wants to create a volume.
// No need to actually manifest the volume on the filesystem yet
// (until Mount is called).
// Name and driver specific options passed through to the ESX host
func (d vmdkDriver) Create(r volume.Request) volume.Response {
	err := d.ops.Create(r.Name, r.Options)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Err: ""}
}

func (d vmdkDriver) Remove(r volume.Request) volume.Response {
	err := d.ops.Remove(r.Name, r.Options)
	if err != nil {
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

	refCount := 0 // TBD: get actual from d.volumes(r.name).refCount
	if refCount != 0 {
		return volume.Response{Err: ""}
	}

	// Get the mount point path and make sure it exists.
	m := filepath.Join(mountRoot, r.Name)
	log.WithFields(log.Fields{"name": r.Name, "mountpoint": m}).Info("Mounting Volume ")

	err := fs.Mkdir(m)
	if err != nil {
		log.WithFields(log.Fields{"dir": m}).Error("Failed to make directory ")
		return volume.Response{Err: err.Error()}
	}

	if err := d.mountVolume(r, m); err != nil {
		return volume.Response{Err: err.Error()}
	}
	log.WithFields(log.Fields{"name": r.Name}).Info("Mount Succeeded ")

	return volume.Response{Mountpoint: m}
}

//
func (d vmdkDriver) Unmount(r volume.Request) volume.Response {
	// make sure it's unmounted on guest side, then detach
	refCount := 0 // TBD: get actual from d.volumes(r.name).refCount
	if refCount > 0 {
		return volume.Response{Err: ""}
	}

	err := d.unmountVolume(r)
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")
	if err != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": err.Error()}).Error("Unmount Failed ")
		return volume.Response{Err: err.Error()}
	}
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmount Succeeded ")
	return volume.Response{Err: ""}
}
