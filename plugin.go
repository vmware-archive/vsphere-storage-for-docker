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
	"log"
	"strings"

	"os"
	"os/exec"
	"path/filepath"
	"sync"
	//	"syscall"

	"github.com/docker/go-plugins-helpers/volume"
	
	"github.com/vmware/docker-vmdk-plugin/vmdkops" 
)

const (
	mountRoot = "/mnt/vmdk" // VMDK block devices are mounted here
)

type vmdkDriver struct {
	m *sync.Mutex // create() serialization
}

func newVmdkDriver() vmdkDriver {
	d := vmdkDriver{
		// TODO: volumes map(string)volinfo, (name->uuid/refcount/creationtime)
		m: &sync.Mutex{},
	}
	return d
}

func (d vmdkDriver) Get(r volume.Request) volume.Response {
	log.Printf("'Get' called on %s -TBD return volume info \n", r.Name)
	return volume.Response{Err: ""}
}

func (d vmdkDriver) List(r volume.Request) volume.Response {
		log.Printf("'List' called on %s - TBD return volumes list \n", r.Name)
		return volume.Response{Err: ""}
	
}

// request attach and them mounts the volume
// actual mount - send attach to ESX and do the in-guest magix
// TODO: this should actually be a goroutine , no need to block
//       SAME (and more) applies to unmount
func (d vmdkDriver) mountVolume(r volume.Request, path string) string {

	// First of all, have ESX attach the disk

	refCount := 0
	if refCount != 0 { // TODO: actual refcounting
		return ""
	}

	// TODO: refcount  if the volume is already mounted (for other container) and
	// just return volume.Response{Mountpoint: m} in this case
	// TODO: save info abouf voliume mount , in memory
	// d.volumes[m] = &volumeName{name: r.Name, connections: 1}
	if msg := vmdkops.VmdkAttach(r.Name, r.Options); msg != "" {
		return msg
	}

	// TODO: evenntually drop the exec.command(mount) and replace with
	// docker.mount.mount (?)
	// OR
	//	if err := syscall.Mount(device, target, mType, flag, data); err != nil {
	//		return err
	//	}
	out, err := exec.Command("blkid", []string{"-L", r.Name}...).Output()
	if err != nil {
		log.Printf("blkid err: %T (%s)", err, string(out))
	} else {
		device := strings.TrimRight(string(out), " \n")

		mountPoint := filepath.Join(mountRoot, r.Name)
		log.Printf("Mounting %s for label %s", device, mountPoint)

		_, err = exec.Command("mount", []string{device, mountPoint}...).Output()
		if err != nil {
			log.Printf("mount failed: %T", err)
		}
	}
	return ""

}

// Unmounts the volume and then requests detach
// TODO : use docker.mount or syscall.mount, and goroutine
// syscall.Unmount(d.volumes(r.Name).device, flag)

func (d vmdkDriver) unmountVolume(r volume.Request) string {

	mountPoint := filepath.Join(mountRoot, r.Name)
	_, err := exec.Command("umount", mountPoint).Output()
	if err != nil {
		log.Printf("umount failed: %T", err)
		return err.Error()
	}
	log.Printf("detach request for %s : sending...", r.Name)
	return vmdkops.VmdkDetach(r.Name, r.Options)
}

// All plugin callbbacks are getting "Name" as a part of volume.Request.
// Create() also getting "Opts" (flags entered via '-o')

// The user wants to create a volume.
// No need to actually manifest the volume on the filesystem yet
// (until Mount is called).
// Name and driver specific options passed through to the ESX host
func (d vmdkDriver) Create(r volume.Request) volume.Response {
	msg := vmdkops.VmdkCreate(r.Name, r.Options)
	return volume.Response{Err: msg}
}

func (d vmdkDriver) Remove(r volume.Request) volume.Response {
	msg := vmdkops.VmdkRemove(r.Name, r.Options)
	return volume.Response{Err: msg}
}

// give docker a reminder of the volume mount path
func (d vmdkDriver) Path(r volume.Request) volume.Response {
	m := filepath.Join(mountRoot, r.Name)

	log.Print("path = ", m)
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
	log.Printf("Mounting volume %s on %s\n", r.Name, m)

	stat, err := os.Lstat(m)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(m, 0755); err != nil {
			return volume.Response{Err: err.Error()}
		}
	} else if err != nil {
		return volume.Response{Err: err.Error()}
	}

	if stat != nil && !stat.IsDir() {
		return volume.Response{Err: fmt.Sprintf("%v already exist and it's not a directory", m)}
	}

	if msg := d.mountVolume(r, m); msg != "" {
		return volume.Response{Err: msg}
	}

	return volume.Response{Mountpoint: m}
}

//
func (d vmdkDriver) Unmount(r volume.Request) volume.Response {
	// make sure it's unmounted on guest side, then detach
	refCount := 0 // TBD: get actual from d.volumes(r.name).refCount
	if refCount > 0 {
		return volume.Response{Err: ""}
	}

	log.Printf("unmount %s", r.Name)
	return volume.Response{Err: d.unmountVolume(r)}
}
