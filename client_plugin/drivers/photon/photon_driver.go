// Copyright 2016-2017 VMware, Inc. All Rights Reserved.
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
// Provide support for --driver=photon in Docker, when Docker VM is running under ESX.
//
// Serves requests from Docker Engine related to VMDK volume operations.
// Depends on vmdk-opsd service to be running on hosting ESX
// (see ./esx_service)
///

//"fmt"
//"path/filepath"
//"github.com/vmware/docker-volume-vsphere/client_plugin/utils/fs"
//"golang.org/x/exp/inotify"
import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/fs"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/plugin_utils"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/refcount"
	"github.com/vmware/photon-controller-go-sdk/photon"
)

const (
	devWaitTimeout       = 1 * time.Second
	sleepBeforeMount     = 1 * time.Second
	watchPath            = "/dev/disk/by-path"
	version              = "Photon volume driver 0.1"
	driverName           = "photon"
	photonPersistentDisk = "persistent-disk"
	capacityKB           = 1024
	capacityGB           = 1
	fsTypeTag            = "Fs_Type"
)

// VolumeDriver - Photon volume driver struct
type VolumeDriver struct {
	client        *photon.Client
	hostID        string
	mountRoot     string
	project       string
	refCounts     *refcount.RefCountsMap
	target        string
	mountIDtoName map[string]string // map of mountID -> full volume name
}

func (d *VolumeDriver) verifyTarget() error {
	// Try fetching the project for the given project ID,
	// verifies the target (client) and the project.
	_, err := d.client.Projects.Get(d.project)

	if err == nil {
		// Fetch the VM using given host ID.
		_, err = d.client.VMs.Get(d.hostID)
	}
	return err
}

// NewVolumeDriver - creates Driver, creates client for given target
func NewVolumeDriver(targetURL string, projectID string, hostID string, mountDir string) *VolumeDriver {

	d := &VolumeDriver{
		target:  targetURL,
		project: projectID,
		hostID:  hostID,
	}
	// Use default timeout of thirty seconds and retry of three
	d.client = photon.NewClient(targetURL, nil, nil)

	err := d.verifyTarget()
	if err != nil {
		log.WithFields(log.Fields{"target": targetURL, "project-id": projectID}).Warning("Invalid target and or project ID, exiting.")
		return nil
	}
	d.mountRoot = mountDir
	d.refCounts = refcount.NewRefCountsMap()
	d.refCounts.Init(d, mountDir, driverName)
	d.mountIDtoName = make(map[string]string)

	log.WithFields(log.Fields{
		"version": version,
		"target":  targetURL,
		"project": projectID,
		"hostID":  hostID,
	}).Info("Docker Photon plugin started ")

	return d
}

// In following three operations on refcount, if refcount
// map hasn't been initialized, return 1 prevent detach and remove.

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

func (d *VolumeDriver) getMountPoint(volName string) string {
	return filepath.Join(d.mountRoot, volName)
}

// validateCreateOptions validates the volume create request.
func validateCreateOptions(r volume.Request) error {
	// Clone isn't supported
	if _, result := r.Options["clone-from"]; result == true {
		return fmt.Errorf("Unrecognized option - clone-from")
	}

	// Flavor must be specified
	if _, result := r.Options["flavor"]; result == false {
		return fmt.Errorf("Missing option - flavor")
	}

	// Use default fstype if not specified
	if _, result := r.Options[fsTypeTag]; result == false {
		r.Options[fsTypeTag] = fs.FstypeDefault
	}

	// Check whether the fstype filesystem is supported.
	errFstype := fs.VerifyFSSupport(r.Options[fsTypeTag])
	if errFstype != nil {
		log.WithFields(log.Fields{"name": r.Name,
			"fstype": r.Options[fsTypeTag]}).Error("Not supported ")
		return errFstype
	}
	return nil
}

func (d *VolumeDriver) taskWait(id string) error {
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

	if strings.HasSuffix(capacity, "mb") {
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
	}
	return 0, fmt.Errorf("Invalid size %s specified for volume %s, size is specified as <size>mb/gb",
		r.Options["size"], r.Name)
}

func convertDiskTags2Map(tags []string, status map[string]interface{}) {
	if len(tags) > 0 {
		// Convert each tag into a key value pair
		for _, tag := range tags {
			s := strings.Split(tag, ":")
			if len(s) == 2 {
				status[s[0]] = s[1]
			}
		}
	}
}

// Get info about a single volume
func (d *VolumeDriver) Get(r volume.Request) volume.Response {
	mountpoint := d.getMountPoint(r.Name)
	status, err := d.GetVolume(r.Name)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to get data for volume ")
		return volume.Response{Err: err.Error()}
	}
	log.WithFields(log.Fields{"name": r.Name, "status": status}).Info("Volume meta-data ")
	return volume.Response{Volume: &volume.Volume{
		Name:       r.Name,
		Mountpoint: mountpoint,
		Status:     status}}
}

// List volumes known to the driver
func (d *VolumeDriver) List(r volume.Request) volume.Response {
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

func (d *VolumeDriver) attachVolume(name string, id string) error {
	diskOp := photon.VmDiskOperation{DiskID: id}
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

func (d *VolumeDriver) detachVolume(name string, id string) error {
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

// GetVolume - returns Photon specific data for a volume
func (d *VolumeDriver) GetVolume(name string) (map[string]interface{}, error) {
	// Create status map
	status := make(map[string]interface{})

	opt := photon.DiskGetOptions{Name: name}
	dlist, err := d.client.Projects.GetDisks(d.project, &opt)
	if err != nil {
		log.WithFields(
			log.Fields{"name": name, "error": err.Error()},
		).Error("Failed to get data for volume ")
		return status, err
	} else if len(dlist.Items) == 0 {
		// No disk of that name was found, its not an error
		// for Photon but return one to the caller.
		log.WithFields(log.Fields{"name": name}).Error("Unknown volume - ")
		return status, fmt.Errorf("Unknown volume - " + name)
	}

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
		convertDiskTags2Map(pDisk.Tags, status)
	}
	return status, nil
}

// MountVolume - Request attach and them mounts the volume.
// Returns mount point and  error (or nil)
func (d *VolumeDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	mountpoint := d.getMountPoint(name)

	// First, make sure  that mountpoint exists.
	err := fs.Mkdir(mountpoint)
	if err != nil {
		log.WithFields(log.Fields{"name": name, "dir": mountpoint}).Error("Failed to make directory for volume mount ")
		return "", err
	}
	if !skipAttach {
		log.WithFields(log.Fields{"name": name, "fstype": fstype}).Info("Attaching volume ")
		err = d.attachVolume(name, id)
		if err != nil {
			return "", err
		}
	}
	return mountpoint, fs.MountWithID(mountpoint, fstype, id, isReadOnly)
}

// private function that does the job of mounting volume in conjunction with refcounting
func (d *VolumeDriver) processMount(r volume.MountRequest) volume.Response {
	volumeInfo, err := plugin_utils.GetVolumeInfo(r.Name, "", d)
	if err != nil {
		log.Errorf("Unable to get volume info for volume %s. err:%v", r.Name, err)
		return volume.Response{Err: err.Error()}
	}
	r.Name = volumeInfo.VolumeName
	d.mountIDtoName[r.ID] = r.Name

	// If the volume is already mounted , just increase the refcount.
	// Note: for new keys, GO maps return zero value, so no need for if_exists.
	refcnt := d.incrRefCount(r.Name) // save map traversal
	log.Debugf("volume name=%s refcnt=%d", r.Name, refcnt)
	if refcnt > 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: d.getMountPoint(r.Name)}
	}

	if plugin_utils.AlreadyMounted(r.Name, d.mountRoot) {
		log.WithFields(log.Fields{"name": r.Name}).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: d.getMountPoint(r.Name)}
	}

	// get volume metadata if required
	volumeMeta := volumeInfo.VolumeMeta
	if volumeMeta == nil {
		if volumeMeta, err = d.GetVolume(r.Name); err != nil {
			d.decrRefCount(r.Name)
			return volume.Response{Err: err.Error()}
		}
	}

	fstype, exists := volumeMeta[fsTypeTag]
	if !exists {
		fstype = fs.FstypeDefault
	}

	skipAttach := false
	// If the volume is already attached to the VM, skip the attach.
	if state, stateExists := volumeMeta["State"]; stateExists {
		if strings.Compare(state.(string), "DETACHED") != 0 {
			skipAttach = true
		}
		log.WithFields(
			log.Fields{"name": r.Name, "skipAttach": skipAttach},
		).Info("Attached state ")
	}

	// Mount the volume and for now its always read-write.
	mountpoint, err := d.MountVolume(r.Name, fstype.(string), volumeMeta["ID"].(string), false, skipAttach)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to mount ")

		d.decrRefCount(r.Name)
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: mountpoint}
}

// UnmountVolume - Unmounts the volume and then requests detach
func (d *VolumeDriver) UnmountVolume(name string) error {
	mountpoint := d.getMountPoint(name)
	status, err := d.GetVolume(name)
	if err != nil {
		return err
	}
	id := status["ID"].(string)

	err = fs.Unmount(mountpoint)
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
func (d *VolumeDriver) Create(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name, "option": r.Options}).Info("Creating volume ")

	err := validateCreateOptions(r)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	size, errSize := getDiskSize(r)
	if errSize != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": errSize}).Error("Create volume failed, invalid size ")
		return volume.Response{Err: errSize.Error()}
	}

	// For now only the fstype is added as a tag to the disk
	tags := []string{"fstype:" + r.Options[fsTypeTag]}

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
	log.WithFields(log.Fields{"name": r.Name, "fstype": r.Options[fsTypeTag]}).Info("Attaching volume and creating filesystem ")

	errAttach := d.attachVolume(r.Name, createTask.Entity.ID)
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

	errMkfs := fs.MkfsByDevicePath(r.Options[fsTypeTag], r.Name, device)
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

	log.WithFields(log.Fields{"name": r.Name, "fstype": r.Options[fsTypeTag]}).Info("Volume and filesystem created ")
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

	// Always get the disk meta-data from Photon, when ref count is zero
	// there is no refcount map entry either and hence no ID.
	status, err := d.GetVolume(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	log.WithFields(log.Fields{"using volume ID": status["ID"].(string)}).Info("Removing volume ")
	rmTask, err := d.client.Disks.Delete(status["ID"].(string))
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err},
		).Error("Failed to remove volume ")
		return volume.Response{Err: err.Error()}
	}

	// Uses default timeout and retry count
	err = d.taskWait(rmTask.ID)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err},
		).Error("Failed to remove volume ")
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Err: ""}
}

// Capabilities - Report plugin scope to Docker
func (d *VolumeDriver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}

// Path - give docker a reminder of the volume mount path
func (d *VolumeDriver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: d.getMountPoint(r.Name)}
}

// Provide a volume to docker container - called once per container start.
// We need to keep refcount and unmount on refcount drop to 0

// Mount - mount a volume
func (d *VolumeDriver) Mount(r volume.MountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")

	// lock the state
	d.refCounts.StateMtx.Lock()
	defer d.refCounts.StateMtx.Unlock()

	// checked by refcounting thread until refmap initialized
	// useless after that
	d.refCounts.MarkDirty()

	return d.processMount(r)
}

// Unmount request from Docker. If mount refcount is drop to 0,
// Unmount and detach from VM
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

	if fullVolName, exist := d.mountIDtoName[r.ID]; exist {
		r.Name = fullVolName
		delete(d.mountIDtoName, r.ID) //cleanup the map
	} else {
		volumeInfo, err := plugin_utils.GetVolumeInfo(r.Name, "", d)
		if err != nil {
			log.Errorf("Unable to get volume info for volume %s. err:%v", r.Name, err)
			return volume.Response{Err: err.Error()}
		}
		r.Name = volumeInfo.VolumeName
	}

	// if refcount has been succcessful, Normal flow.
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
	err = d.UnmountVolume(r.Name)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name, "error": err.Error()},
		).Error("Failed to unmount ")
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Err: ""}
}
