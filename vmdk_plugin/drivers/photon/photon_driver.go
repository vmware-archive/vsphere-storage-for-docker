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

package photon

//
// VMWare VMDK Docker Data Volume plugin.
//
// Provide support for --driver=vmdk in Docker, when Docker VM is running under ESX.
//
// Serves requests from Docker Engine related to VMDK volume operations.
// Depends on vmdk-opsd service to be running on hosting ESX
// (see ./esx_service)
///

//"fmt"
//"path/filepath"
//"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/fs"
//"golang.org/x/exp/inotify"
import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/fs"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/refcount"
	"github.com/vmware/photon-controller-go-sdk/photon"
)

const (
	devWaitTimeout       = 1 * time.Second
	sleepBeforeMount     = 1 * time.Second
	watchPath            = "/dev/disk/by-path"
	version              = "Photon volume driver x.x"
	driverName           = "photon"
	photonPersistentDisk = "persistent-disk"
	capacityKB           = 1024
	capacityMB           = 1048576
	capacityGB           = 1
)

// Driver - Photon volume driver struct
type Driver struct {
	m         *sync.Mutex // create() serialization - for future use
	refCounts refcount.RefCountsMap
	client    *photon.Client
	target    string
	project   string
	hostID    string
	mountRoot string
}

// NewVolumeDriver - creates Driver, creates client for given target
func NewVolumeDriver(targetURL string, projectID string, hostID string, mountDir string) *Driver {

	d := &Driver{
		m:         &sync.Mutex{},
		target:    targetURL,
		project:   projectID,
		hostID:    hostID,
		refCounts: make(refcount.RefCountsMap),
	}
	// Use default timeout of thirty seconds and retry of three
	d.client = photon.NewClient(targetURL, nil, nil)
	d.mountRoot = mountDir

	d.refCounts.Init(d, mountDir, driverName)

	// Set target
	log.WithFields(log.Fields{
		"version": version,
		"target":  targetURL,
		"project": projectID,
		"hostID":  hostID,
	}).Info("Docker Photon plugin started ")

	return d
}
func (d *Driver) getRefCount(vol string) uint           { return d.refCounts.GetCount(vol) }
func (d *Driver) incrRefCount(vol string) uint          { return d.refCounts.Incr(vol) }
func (d *Driver) decrRefCount(vol string) (uint, error) { return d.refCounts.Decr(vol) }

func (d *Driver) getDiskID(vol string) string {
	return d.refCounts.GetID(vol)
}

func (d *Driver) addDiskID(vol string, id string) {
	d.refCounts.AddID(vol, id)
}

func (d *Driver) deleteRefCnt(vol string) {
	d.refCounts.DeleteRefCnt(vol)
}

func (d *Driver) getMountPoint(volName string) string {
	return filepath.Join(d.mountRoot, volName)
}

func validateCreateOptions(r volume.Request, fsMap map[string]string) error {
	// Clone isn't supported
	if _, result := r.Options["clone-from"]; result == true {
		return fmt.Errorf("Unrecognized option - clone-from")
	}

	// Flavor must be specified
	if _, result := r.Options["flavor"]; result == false {
		return fmt.Errorf("Missing option - flavor")
	}

	// Use default fstype if not specified
	if _, result := r.Options["fstype"]; result == false {
		r.Options["fstype"] = fs.FstypeDefault
	}

	// Verify the existence of fstype mkfs
	_, result := fsMap[r.Options["fstype"]]
	if result == false {
		msg := "Not found mkfs for " + r.Options["fstype"]
		msg += "\nSupported filesystems found: "
		validfs := ""
		for fs := range fsMap {
			if validfs != "" {
				validfs += ", " + fs
			} else {
				validfs += fs
			}
		}
		log.WithFields(log.Fields{"name": r.Name,
			"fstype": r.Options["fstype"]}).Error("Not found ")
		return fmt.Errorf(msg + validfs)
	}
	return nil
}

func (d *Driver) taskWait(id string) error {
	_, err := d.client.Tasks.Wait(id)
	if err != nil {
		log.WithFields(
			log.Fields{"error": err, "taskID": id},
		).Error("Task error - ")
		return err
	}
	return nil
}

func getDiskSize(r volume.Request) (int, error) {
	// Convert given size to GB and default to min
	// 1GB.
	capacity, err := r.Options["size"]
	if err == false {
		return capacityGB, nil
	}
	capacity = strings.ToLower(capacity)

	if strings.HasSuffix(capacity, "kb") {
		val := strings.Split(capacity, "kb")
		bytes, err := strconv.Atoi(val[0])
		if err != nil {
			return 0, err
		}
		log.Debugf("Got bytes=%d", bytes)
		if bytes < capacityKB || (bytes/capacityMB) < capacityGB {
			return 0, fmt.Errorf("Invalid size %s specified for volume %s",
				r.Options["size"], r.Name)
		}
		return bytes / capacityMB, nil
	} else if strings.HasSuffix(capacity, "mb") {
		val := strings.Split(capacity, "mb")
		bytes, err := strconv.Atoi(val[0])
		if err != nil {
			return 0, err
		}
		log.Debugf("Got bytes=%d", bytes)
		if bytes < capacityKB || (bytes/capacityKB) < capacityGB {
			return 0, fmt.Errorf("Invalid size %s specified for volume %s",
				r.Options["size"], r.Name)
		}
		return bytes / capacityKB, nil
	} else if strings.HasSuffix(capacity, "gb") {
		val := strings.Split(capacity, "gb")
		bytes, err := strconv.Atoi(val[0])
		if err != nil {
			return 0, err
		}
		log.Debugf("Got bytes=%d", bytes)
		return bytes, nil
	} else if strings.HasSuffix(capacity, "tb") {
		val := strings.Split(capacity, "tb")
		bytes, err := strconv.Atoi(val[0])
		if err != nil {
			return 0, err
		}
		log.Debugf("Got bytes=%d", bytes)
		return bytes * capacityKB, nil
	}
	return 0, fmt.Errorf("Invalid size %s specified for volume %s",
		r.Options["size"], r.Name)
}

func convertDiskTags2Map(tags []string) map[string]interface{} {
	tagMap := make(map[string]interface{})
	if len(tags) > 0 {
		// Convert each tag into a key value pair
		for _, tag := range tags {
			s := strings.Split(tag, ":")
			if len(s) == 2 {
				tagMap[s[0]] = s[1]
			}
		}
	}
	return tagMap
}

// Get info about a single volume
func (d *Driver) Get(r volume.Request) volume.Response {
	opt := photon.DiskGetOptions{Name: r.Name}
	dlist, err := d.client.Projects.GetDisks(d.project, &opt)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to get data for volume ")
		return volume.Response{Err: err.Error()}
	} else if len(dlist.Items) == 0 {
		// No disk of that name was found, its not an error
		// for Photon but return one to the caller.
		return volume.Response{Err: "Unknown volume " + r.Name}
	}

	mountpoint := d.getMountPoint(r.Name)

	// Create status map
	status := make(map[string]interface{})
	if len(dlist.Items) > 0 {
		pDisk := dlist.Items[0]
		status["Flavor"] = pDisk.Flavor
		status["Kind"] = pDisk.Kind
		status["Datastore"] = pDisk.Datastore
		status["CapacityGB"] = pDisk.CapacityGB
		status["State"] = pDisk.State
		status["ID"] = pDisk.ID
		if len(pDisk.VMs) > 0 {
			status["Attached-to-VM"] = pDisk.VMs[0]
		}
		status["Tags"] = convertDiskTags2Map(pDisk.Tags)
		//Ensure the refcount map has this disk ID
		d.addDiskID(r.Name, pDisk.ID)
	}
	log.WithFields(log.Fields{"name": r.Name, "status": status}).Info("Volume meta-data ")
	return volume.Response{Volume: &volume.Volume{
		Name:       r.Name,
		Mountpoint: mountpoint,
		Status:     status}}
}

// List volumes known to the driver
func (d *Driver) List(r volume.Request) volume.Response {
	dlist, err := d.client.Projects.GetDisks(d.project, nil)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	responseVolumes := make([]*volume.Volume, 0, len(dlist.Items))
	for _, vol := range dlist.Items {
		mountpoint := d.getMountPoint(vol.Name)
		responseVol := volume.Volume{Name: vol.Name, Mountpoint: mountpoint}
		responseVolumes = append(responseVolumes, &responseVol)
	}
	return volume.Response{Volumes: responseVolumes}
}

func (d *Driver) attachVolume(name string) error {
	diskOp := photon.VmDiskOperation{DiskID: d.getDiskID(name)}
	attachTask, errAttach := d.client.VMs.AttachDisk(d.hostID, &diskOp)
	if errAttach != nil {
		log.WithFields(log.Fields{"name": name, "error": errAttach}).Error("Failed to attach volume ")
		return errAttach
	}

	// Uses default timeout and retry count
	errTask := d.taskWait(attachTask.ID)
	if errTask != nil {
		log.WithFields(
			log.Fields{"name": name, "error": errTask},
		).Error("Failed to attach volume ")
		return errTask
	}
	return nil
}

func (d *Driver) detachVolume(name string, id string) error {
	diskOp := photon.VmDiskOperation{DiskID: id}
	detachTask, errDetach := d.client.VMs.DetachDisk(d.hostID, &diskOp)
	if errDetach != nil {
		log.WithFields(log.Fields{"name": name, "error": errDetach}).Error("Failed to detach volume ")
		return errDetach
	}

	// Uses default timeout and retry count
	errTask := d.taskWait(detachTask.ID)
	if errTask != nil {
		log.WithFields(log.Fields{"name": name, "error": errTask}).Error("Failed to detach volume ")
		return errTask
	}
	err := fs.DeleteDevicePathWithID(id)
	if err != nil {
		log.WithFields(log.Fields{"name": name, "id": id, "err": err.Error()}).Error("Failed to delete device path for ")
	}

	log.WithFields(log.Fields{"name": name, "id": id}).Info("Detached volume ")
	return nil
}

// Request attach and them mounts the volume.
// Returns mount point and  error (or nil)
func (d *Driver) mountVolume(name string, fstype string, skipAttach int) (string, error) {
	mountpoint := d.getMountPoint(name)

	// First, make sure  that mountpoint exists.
	err := fs.Mkdir(mountpoint)
	if err != nil {
		log.WithFields(log.Fields{"name": name, "dir": mountpoint}).Error("Failed to make directory for volume mount ")
		return "", err
	}
	if skipAttach == 0 {
		log.WithFields(log.Fields{"name": name, "fstype": fstype}).Info("Attaching volume ")
		err = d.attachVolume(name)
		if err != nil {
			return "", err
		}
	}
	return mountpoint, fs.MountWithID(mountpoint, fstype, d.getDiskID(name))
}

// Unmounts the volume and then requests detach
func (d *Driver) unmountVolume(name string, id string) error {
	mountpoint := d.getMountPoint(name)
	err := fs.Unmount(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"mountpoint": mountpoint, "error": err},
		).Error("Failed to unmount volume. Now trying to detach... ")
		// Do not return error. Continue with detach.
	}
	log.WithFields(log.Fields{"name": name, "id": id}).Info("Unmounted volume ")

	err = d.detachVolume(name, id)
	if err != nil {
		return err
	}
	return nil

}

// Create - create a volume.
func (d *Driver) Create(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name, "option": r.Options}).Info("Creating volume ")

	// Get existent filesystem tools, for now only ext4
	supportedFs := fs.MkfsLookup()

	err := validateCreateOptions(r, supportedFs)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	size, errSize := getDiskSize(r)
	if errSize != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errSize}).Error("Create volume failed, invalid size ")
		return volume.Response{Err: errSize.Error()}
	}

	// For now only the fstype is added as a tag to the disk
	tags := []string{"fstype:" + r.Options["fstype"]}

	// Create disk
	dSpec := photon.DiskCreateSpec{Flavor: r.Options["flavor"],
		Kind:       photonPersistentDisk,
		CapacityGB: size,
		//TODO: later
		//Affinities: []photon.LocalitySpec{{Kind: "vm", ID: d.hostID}},
		Name: r.Name,
		Tags: tags}

	createTask, errCreate := d.client.Projects.CreateDisk(d.project, &dSpec)
	if errCreate != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errCreate}).Error("Create volume failed ")
		return volume.Response{Err: errCreate.Error()}
	}

	// Uses default timeout and retry count
	errTask := d.taskWait(createTask.ID)
	if errTask != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": errTask},
		).Error("Failed to create volume ")
		return volume.Response{Err: errTask.Error()}
	}

	// Handle filesystem creation
	log.WithFields(log.Fields{"name": r.Name, "fstype": r.Options["fstype"]}).Info("Attaching volume and creating filesystem ")

	//Ensure the refcount map has this disk ID, needed by the attach below.
	d.addDiskID(r.Name, createTask.Entity.ID)

	errAttach := d.attachVolume(r.Name)
	if errAttach != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errAttach}).Error("Attach volume failed, removing the volume ")
		resp := d.Remove(volume.Request{Name: r.Name})
		if resp.Err != "" {
			log.WithFields(log.Fields{"name": r.Name, "error": resp.Err}).Warning("Remove volume failed ")
		}
		return volume.Response{Err: errAttach.Error()}
	}

	device, errGetDevicePath := fs.GetDevicePathByID(createTask.Entity.ID)
	if errGetDevicePath != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errGetDevicePath}).Error("Could not find attached device, removing the volume ")
		err = d.detachVolume(r.Name, createTask.Entity.ID)
		if err != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": err}).Warning("Detach volume failed ")
		}
		resp := d.Remove(volume.Request{Name: r.Name})
		if resp.Err != "" {
			log.WithFields(log.Fields{"name": r.Name, "error": resp.Err}).Warning("Remove volume failed ")
		}
		return volume.Response{Err: errGetDevicePath.Error()}
	}

	errMkfs := fs.Mkfs(supportedFs[r.Options["fstype"]], r.Name, device)
	if errMkfs != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errMkfs}).Error("Create filesystem failed, removing the volume ")
		err = d.detachVolume(r.Name, createTask.Entity.ID)
		if err != nil {
			log.WithFields(log.Fields{"name": r.Name, "error": err}).Warning("Detach volume failed ")
		}
		resp := d.Remove(volume.Request{Name: r.Name})
		if resp.Err != "" {
			log.WithFields(log.Fields{"name": r.Name, "error": resp.Err}).Warning("Remove volume failed ")
		}
		return volume.Response{Err: errMkfs.Error()}
	}

	err = d.detachVolume(r.Name, createTask.Entity.ID)
	if err != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": err}).Error("Detach volume failed ")
		return volume.Response{Err: err.Error()}
	}

	log.WithFields(log.Fields{"name": r.Name, "fstype": r.Options["fstype"]}).Info("Volume and filesystem created ")
	return volume.Response{Err: ""}
}

// Remove - removes individual volume. Docker would call it only if is not using it anymore
func (d *Driver) Remove(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Removing volume ")

	// Docker is supposed to block 'remove' command if the volume is used. Verify.
	if d.getRefCount(r.Name) != 0 {
		msg := fmt.Sprintf("Remove failure - volume is still mounted. "+
			" volume=%s, refcount=%d", r.Name, d.getRefCount(r.Name))
		log.Error(msg)
		return volume.Response{Err: msg}
	}

	// Always get the disk meta-data from Photon, when ref count is zero
	// there is no refcount map entry either and hence no ID.
	resp := d.Get(volume.Request{Name: r.Name})

	if resp.Err != "" {
		return resp
	}
	log.WithFields(log.Fields{"using volume ID": d.getDiskID(r.Name)}).Info("Removing volume ")
	rmTask, err := d.client.Disks.Delete(d.getDiskID(r.Name))
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err},
		).Error("Failed to remove volume ")
		d.deleteRefCnt(r.Name)
		return volume.Response{Err: err.Error()}
	}

	// Uses default timeout and retry count
	err = d.taskWait(rmTask.ID)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err},
		).Error("Failed to remove volume ")
		d.deleteRefCnt(r.Name)
		return volume.Response{Err: err.Error()}
	}

	// Clear the ID saved via Get() for this disk.
	d.deleteRefCnt(r.Name)

	return volume.Response{Err: ""}
}

// Capabilities - Report plugin scope to Docker
func (d *Driver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}

// Path - give docker a reminder of the volume mount path
func (d *Driver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: d.getMountPoint(r.Name)}
}

// Provide a volume to docker container - called once per container start.
// We need to keep refcount and unmount on refcount drop to 0

// Mount - mount a volume
func (d *Driver) Mount(r volume.MountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")

	d.m.Lock()
	defer d.m.Unlock()

	// If the volume is already mounted , just increase the refcount.
	//
	// Note: We are deliberately incrementing refcount first, before trying
	// to do anything else. If Mount fails, Docker will send Unmount request,
	// and we will happily decrement the refcount there, and will fail the unmount
	// since the volume will have been never mounted.
	// Note: for new keys, GO maps return zero value, so no need for if_exists.

	refcnt := d.incrRefCount(r.Name) // save map traversal
	log.Debugf("volume name=%s refcnt=%d", r.Name, refcnt)
	if refcnt > 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: d.getMountPoint(r.Name)}
	}

	// This is the first time we are asked to mount the volume. The disk
	// ID isn't always cached by the plugin on a host, the disk could have
	// been created on a different host and getting mounted on another host.
	// So on the first mount fetch the disk from Photon and add it's ID to
	// the ref count map.
	resp := d.Get(volume.Request{Name: r.Name})

	if resp.Err != "" {
		d.decrRefCount(r.Name)
		return resp
	}

	tmap := resp.Volume.Status["Tags"].(map[string]interface{})
	fstype, exists := tmap["fstype"].(string)
	if !exists {
		fstype = fs.FstypeDefault
	}

	skipAttach := 0
	if state, stateExists := resp.Volume.Status["State"]; stateExists {
		skipAttach = strings.Compare(state.(string), "DETACHED")
		log.WithFields(
			log.Fields{"name": r.Name, "skipAttach": skipAttach},
		).Info("Attached state ")
	}

	mountpoint, err := d.mountVolume(r.Name, fstype, skipAttach)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to mount ")
		d.decrRefCount(r.Name)
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: mountpoint}
}

// Unmount request from Docker. If mount refcount is drop to 0,
// Unmount and detach from VM
func (d *Driver) Unmount(r volume.UnmountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")
	d.m.Lock()
	defer d.m.Unlock()

	diskID := d.getDiskID(r.Name)
	// if the volume is still used by other containers, just return OK
	refcnt, err := d.decrRefCount(r.Name)
	if err != nil {
		// something went wrong - yell, but still try to unmount
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Error("Refcount error - still trying to unmount...")
	}

	log.Debugf("volume name=%s refcnt=%d", r.Name, refcnt)
	if refcnt >= 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Still in use, skipping unmount request. ")
		return volume.Response{Err: ""}
	}

	// and if nobody needs it, unmount and detach
	err = d.unmountVolume(r.Name, diskID)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to unmount ")
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Err: ""}
}
