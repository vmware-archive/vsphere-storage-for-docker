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

package shared

//
// VMWare vSphere Shared Docker Data Volume plugin version.
//
// Provide support for --driver=shared in Docker, when Docker VM is running under ESX.
//
// Serves requests from Docker Engine related to vsphere shared volume operations.
///

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/dockerops"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/kvstore"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/kvstore/etcdops"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/utils"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/fs"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/plugin_utils"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/refcount"
)

/* Constants
   version:                 Version of the shared plugin driver
   internalVolumePrefix:    Prefix for names of internal volumes
                            which serve as backend stores for shared
                            volumes
   fsType:                  Type of file system that will be presented
                            in the shared volume
*/
const (
	version              = "vSphere Shared Volume Driver v0.2"
	internalVolumePrefix = "InternalVol"
	fsType               = "cifs"
)

/* VolumeDriver - vsphere shared plugin volume driver struct
   dockerOps:               Docker related methods and information
   internalVolumeDriver:    Name of the plugin used by shared volume
                            plugin to create internal volumes
   kvStore:                 Key-value store related methods and information
*/

// VolumeDriver - Contains vars specific to this driver
type VolumeDriver struct {
	utils.PluginDriver
	dockerOps            *dockerops.DockerOps
	internalVolumeDriver string
	kvStore              kvstore.KvStore
}

/* VolumeMetadata structure contains all the
   metadata about a volume that will be put in etcd

   status:          What state is the shared volume currently in?
   globalRefcount:  How many host VMs are accessing this volume?
   port:            On which port is the Samba service listening?
   serviceName:     What is the name of the Samba service for this volume?
   username:
   password:        Local Samba username and password
	            Only default values for now, later can be used
                    for multi tenancy.
   clientList:      List of all host VMs using this shared volume
*/

// VolumeMetadata - Contains metadata of shared volumes
type VolumeMetadata struct {
	Status         kvstore.VolStatus `json:"-"` // Field won't be marshalled
	GlobalRefcount int               `json:"-"` // Field won't be marshalled
	Port           int               `json:"port,omitempty"`
	ServiceName    string            `json:"serviceName,omitempty"`
	Username       string            `json:"username,omitempty"`
	Password       string            `json:"password,omitempty"`
	ClientList     []string          `json:"clientList,omitempty"`
}

// NewVolumeDriver creates driver instance
func NewVolumeDriver(cfg config.Config, mountDir string) *VolumeDriver {
	var d VolumeDriver

	d.RefCounts = refcount.NewRefCountsMap()
	d.RefCounts.Init(&d, mountDir, cfg.Driver)
	d.MountIDtoName = make(map[string]string)
	d.MountRoot = mountDir

	// Read flag from CLI. If not provided, use cfg value
	internalVolumeParam := flag.String("InternalDriver", "",
		"Driver which creates internal volumes")
	flag.Parse()
	if *internalVolumeParam == "" {
		d.internalVolumeDriver = cfg.InternalDriver
	} else {
		d.internalVolumeDriver = *internalVolumeParam
	}

	// create new docker operation client
	d.dockerOps = dockerops.NewDockerOps()
	if d.dockerOps == nil {
		log.Errorf("Failed to create new DockerOps")
		return nil
	}

	// Load the file server image
	go d.dockerOps.LoadFileServerImage()
	log.Infof("Started loading file server image")

	// initialize built-in etcd cluster
	etcdKVS := etcdops.NewKvStore(d.dockerOps)
	if etcdKVS == nil {
		log.Errorf("Failed to create new KV store")
		return nil
	}
	d.kvStore = etcdKVS

	log.WithFields(log.Fields{
		"version": version,
	}).Info("vSphere shared plugin started ")

	return &d
}

// Get info about a single volume
func (d *VolumeDriver) Get(r volume.Request) volume.Response {
	log.Infof("VolumeDriver Get: %s", r.Name)
	status, err := d.GetVolume(r.Name)
	if err != nil {
		log.WithFields(log.Fields{"name": r.Name, "error": err}).Error("Failed to get volume meta-data ")
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Volume: &volume.Volume{Name: r.Name,
		Mountpoint: d.GetMountPoint(r.Name),
		Status:     status}}
}

// List volumes known to the driver
func (d *VolumeDriver) List(r volume.Request) volume.Response {
	volumes, err := d.kvStore.List(kvstore.VolPrefixState)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to get volume list ")
		return volume.Response{Err: err.Error()}
	}

	responseVolumes := make([]*volume.Volume, 0, len(volumes))

	for _, vol := range volumes {
		responseVol := volume.Volume{Name: vol,
			Mountpoint: d.GetMountPoint(vol)}
		responseVolumes = append(responseVolumes, &responseVol)
	}

	return volume.Response{Volumes: responseVolumes}
}

// GetVolume - return volume meta-data.
func (d *VolumeDriver) GetVolume(name string) (map[string]interface{}, error) {
	var statusMap map[string]interface{}
	var volRecord VolumeMetadata
	statusMap = make(map[string]interface{})

	// The kv pairs we want from the KV store
	keys := []string{
		kvstore.VolPrefixState + name,
		kvstore.VolPrefixGRef + name,
		kvstore.VolPrefixInfo + name,
	}

	// KV pairs will be returned in same order in which they were requested
	entries, err := d.kvStore.ReadMetaData(keys)
	if err != nil {
		if err.Error() == kvstore.VolumeDoesNotExistError {
			log.Infof("Volume not found: %s", name)
			return statusMap, err
		}
		msg := fmt.Sprintf("Failed to read metadata for volume %s from KV store. %v",
			name, err)
		log.Warningf(msg)
		return statusMap, errors.New(msg)
	}

	statusMap["Volume Status"] = entries[0].Value
	statusMap["Global Refcount"], _ = strconv.Atoi(entries[1].Value)
	// Unmarshal Info key
	err = json.Unmarshal([]byte(entries[2].Value), &volRecord)
	if err != nil {
		msg := fmt.Sprintf("Failed to unmarshal data. %v", err)
		log.Warningf(msg)
		return statusMap, errors.New(msg)
	}
	statusMap["File server Port"] = volRecord.Port
	statusMap["Service name"] = volRecord.ServiceName
	statusMap["Clients"] = volRecord.ClientList

	return statusMap, nil
}

// Create - create a volume.
func (d *VolumeDriver) Create(r volume.Request) volume.Response {
	log.Infof("VolumeDriver Create: %s", r.Name)
	var msg string
	var entries []kvstore.KvPair

	// Initialize volume metadata in KV store
	volRecord := VolumeMetadata{
		Status:         kvstore.VolStateCreating,
		GlobalRefcount: 0,
		Port:           0,
		Username:       dockerops.SambaUsername,
		Password:       dockerops.SambaPassword,
	}

	// Append global refcount and status to kv pairs that will be written
	entries = append(entries, kvstore.KvPair{Key: kvstore.VolPrefixGRef + r.Name, Value: strconv.Itoa(volRecord.GlobalRefcount)})
	entries = append(entries, kvstore.KvPair{Key: kvstore.VolPrefixState + r.Name, Value: string(volRecord.Status)})
	// Append the rest of the metadata as one KV pair where the data is jsonified
	byteRecord, err := json.Marshal(volRecord)
	if err != nil {
		msg = fmt.Sprintf("Cannot create volume. Failed to marshal metadata to json. Reason: %v", err)
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}
	entries = append(entries, kvstore.KvPair{Key: kvstore.VolPrefixInfo + r.Name, Value: string(byteRecord)})

	log.Infof("Attempting to write initial metadata entry for %s", r.Name)
	err = d.kvStore.WriteMetaData(entries)
	if err != nil {
		msg = fmt.Sprintf("Failed to create volume %s. Reason: %v",
			r.Name, err)
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}

	// Create traditional volume as backend to shared volume
	log.Infof("Attempting to create internal volume for %s", r.Name)
	internalVolname := internalVolumePrefix + r.Name
	err = d.dockerOps.VolumeCreate(d.internalVolumeDriver, internalVolname, r.Options)
	if err != nil {
		msg = fmt.Sprintf("Failed to create internal volume %s. Reason: %v", r.Name, err)
		msg += fmt.Sprintf(". Check the status of the volumes belonging to driver \"%s\".", d.internalVolumeDriver)
		log.Warningf(msg)

		// If failed, attempt to delete the metadata for this volume
		err = d.kvStore.DeleteMetaData(r.Name)
		if err != nil {
			log.Warningf("Failed to remove metadata entry for volume: %s. Reason: %v", r.Name, err)
		}
		return volume.Response{Err: msg}
	}

	// Update metadata to indicate successful volume creation
	log.Infof("Attempting to update volume state to ready for volume: %s", r.Name)
	entries = nil
	entries = append(entries, kvstore.KvPair{Key: kvstore.VolPrefixState + r.Name, Value: string(kvstore.VolStateReady)})
	err = d.kvStore.WriteMetaData(entries)
	if err != nil {
		outerMessage := fmt.Sprintf("Failed to set status of volume %s to ready. Reason: %v", r.Name, err)
		log.Warningf(outerMessage)

		// If failed, attempt to remove the backing trad volume
		log.Infof("Attempting to delete internal volume")
		err = d.dockerOps.VolumeRemove(internalVolname)
		if err != nil {
			msg = fmt.Sprintf(" Failed to remove internal volume. Reason %v.", err)
			msg += fmt.Sprintf(" Please remove the volume manually. Volume: %s", internalVolname)
			log.Warningf(msg)
			outerMessage = outerMessage + msg
		}

		// Attempt to delete the metadata for this volume
		err = d.kvStore.DeleteMetaData(r.Name)
		if err != nil {
			log.Warningf("Failed to remove metadata entry for volume: %s. Reason: %v", r.Name, err)
		}

		return volume.Response{Err: outerMessage}
	}

	log.Infof("Successfully created volume: %s", r.Name)
	return volume.Response{Err: ""}
}

// Remove - removes individual volume. Docker would call it only if is not using it anymore
func (d *VolumeDriver) Remove(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Removing volume ")
	var msg string
	var volRecord VolumeMetadata

	// Cannot remove volumes till plugin completely initializes
	// because we don't know if it is being used or not
	if d.RefCounts.IsInitialized() != true {
		msg = fmt.Sprintf(plugin_utils.PluginInitError+" Cannot remove volume %s",
			r.Name)
		log.Error(msg)
		return volume.Response{Err: msg}
	}

	// Docker is supposed to block 'remove' command if the volume is used.
	// Check on the local refcount first
	if d.GetRefCount(r.Name) != 0 {
		msg = fmt.Sprintf("Remove failed: Containers on this host VM are still using volume %s.",
			r.Name)
		log.Error(msg)
		return volume.Response{Err: msg}
	}

	// Test and set status to Deleting
	if !d.kvStore.CompareAndPutStateOrBusywait(kvstore.VolPrefixState+r.Name,
		string(kvstore.VolStateReady),
		string(kvstore.VolStateDeleting)) {
		// Failed to change state from Ready to Deleting
		// 1. Volume is in Mounted state -> get clients and return error
		// 2. Volume is already in Deleting/Error/Creating/Unmounting and timeout -> continue delete
		// 3. Volume is in other states -> return error

		// Get a list of host VMs using this volume, if any
		keys := []string{
			kvstore.VolPrefixState + r.Name,
			kvstore.VolPrefixInfo + r.Name,
		}
		entries, err := d.kvStore.ReadMetaData(keys)
		if err != nil {
			msg = fmt.Sprintf("Remove failed: cannot read metadata of volume %s", r.Name)
			log.Errorf(msg)
			return volume.Response{Err: msg}
		}

		state := entries[0].Value
		switch state {
		case string(kvstore.VolStateDeleting):
			log.Warningf("Remove: volume in Deleting state after timeout. Continue deleting.")
		case string(kvstore.VolStateError):
		case string(kvstore.VolStateCreating):
		case string(kvstore.VolStateUnmounting):
			log.Warningf("Remove: volume in %s state after timeout. Continue deleting", state)
			if !d.kvStore.CompareAndPut(kvstore.VolPrefixState+r.Name,
				state, string(kvstore.VolStateDeleting)) {
				msg = fmt.Sprintf("Remove: Volume state changed unexpected. Please retry later")
				log.Errorf(msg)
				return volume.Response{Err: msg}

			}
		case string(kvstore.VolStateMounted):
			// Unmarshal Info key
			err = json.Unmarshal([]byte(entries[1].Value), &volRecord)
			if err != nil {
				msg = fmt.Sprintf("Remove failed: cannot unmarshal info data. %v", err)
				log.Errorf(msg)
				return volume.Response{Err: msg}
			}

			msg = fmt.Sprintf("Remove failed: volume state is Mounted.")
			msg += fmt.Sprintf(" Host VMs using this volume: %s",
				strings.Join(volRecord.ClientList, ","))
			log.Errorf(msg)
			return volume.Response{Err: msg}
		default:
			msg = fmt.Sprintf("Remove failed: cannot delete from current state %s.", state)
			log.Errorf(msg)
			return volume.Response{Err: msg}
		}
	}

	// Delete internal volume
	log.Infof("Attempting to delete internal volume for %s", r.Name)
	d.dockerOps.DeleteInternalVolume(r.Name)

	// Delete metadata associated with this volume
	log.Infof("Attempting to delete volume metadata for %s", r.Name)
	err := d.kvStore.DeleteMetaData(r.Name)
	if err != nil {
		msg = fmt.Sprintf("Failed to delete volume metadata for %s. Reason: %v", r.Name, err)
		return volume.Response{Err: msg}
	}

	return volume.Response{Err: ""}
}

// Path - give docker a reminder of the volume mount path
func (d *VolumeDriver) Path(r volume.Request) volume.Response {
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
func (d *VolumeDriver) Mount(r volume.MountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")

	// lock the state
	d.RefCounts.StateMtx.Lock()
	defer d.RefCounts.StateMtx.Unlock()

	// checked by refcounting thread until refmap initialized
	// useless after that
	d.RefCounts.MarkDirty()

	return d.processMount(r)
}

// processMount -  process a mount request
func (d *VolumeDriver) processMount(r volume.MountRequest) volume.Response {
	d.MountIDtoName[r.ID] = r.Name

	// If the volume is already mounted , just increase the refcount.
	// Note: for new keys, GO maps return zero value, so no need for if_exists.
	refcnt := d.IncrRefCount(r.Name) // save map traversal
	log.Debugf("volume name=%s refcnt=%d", r.Name, refcnt)
	if refcnt > 1 {
		log.WithFields(
			log.Fields{"name": r.Name, "refcount": refcnt},
		).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: d.GetMountPoint(r.Name)}
	}

	if plugin_utils.AlreadyMounted(r.Name, d.MountRoot) {
		log.WithFields(log.Fields{"name": r.Name}).Info("Already mounted, skipping mount. ")
		return volume.Response{Mountpoint: d.GetMountPoint(r.Name)}
	}

	mountpoint, err := d.MountVolume(r.Name, "", "", false, true)
	if err != nil {
		log.WithFields(
			log.Fields{"name": r.Name,
				"error": err},
		).Error("Failed to mount ")

		refcnt, _ := d.DecrRefCount(r.Name)
		if refcnt == 0 {
			log.Infof("Detaching %s - it is not used anymore", r.Name)
			// TODO: umount here
		}
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: mountpoint}
}

// MountVolume - Request attach and then mounts the volume.
func (d *VolumeDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	mountpoint := d.GetMountPoint(name)
	// First, make sure  that mountpoint exists.
	err := fs.Mkdir(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"name": name,
				"dir": mountpoint},
		).Error("Failed to make directory for volume mount ")
		return mountpoint, err
	}

	// Increase GRef
	log.Infof("Before AtomicIncr")
	err = d.kvStore.AtomicIncr(kvstore.VolPrefixGRef + name)
	if err != nil {
		log.WithFields(
			log.Fields{"name": name,
				"error": err},
		).Error("Failed to increase global refcount when processMount ")
		return "", err
	}

	// Blocking wait until the state of volume becomes Mounted
	// the change of global refcount will trigger one watcher on manager nodes
	// watchers should start event handler to transit the state of volumes
	info, err := d.kvStore.BlockingWaitAndGet(kvstore.VolPrefixState+name,
		string(kvstore.VolStateMounted), kvstore.VolPrefixInfo+name)
	if err != nil {
		msg := fmt.Sprintf("Failed to blocking wait for Mounted state. Error: %v.", err)
		err = d.kvStore.AtomicDecr(kvstore.VolPrefixGRef + name)
		if err != nil {
			msg += fmt.Sprintf(" Also failed to decrease global refcount. Error: %v.", err)
		}
		log.WithFields(
			log.Fields{"name": name,
				"error": msg}).Error("")
		return "", errors.New(msg)
	}

	// Start mounting
	log.Infof("Volume state mounted, prepare to mounting locally")
	var volRecord VolumeMetadata
	// Unmarshal Info key
	err = json.Unmarshal([]byte(info), &volRecord)
	if err != nil {
		log.WithFields(
			log.Fields{"name": name,
				"error": err},
		).Error("Failed to unmarshal info data ")
		return "", err
	}

	log.WithFields(
		log.Fields{"name": name,
			"Port":        volRecord.Port,
			"ServiceName": volRecord.ServiceName,
		}).Info("Get info for mounting ")
	err = d.mountSharedVolume(name, mountpoint, &volRecord)
	if err != nil {
		msg := fmt.Sprintf("Failed to mount shared volume. Error: %v.", err)
		// AtomicDecr decreases global refcount by one
		// if global refcount reduces from 1 to 0, a watcher event is triggered on manager nodes
		err = d.kvStore.AtomicDecr(kvstore.VolPrefixGRef + name)
		if err != nil {
			msg += fmt.Sprintf(" Also failed to decrease global refcount. Error: %v.", err)
		}
		log.WithFields(
			log.Fields{"name": name,
				"error": msg}).Error("")
		return "", errors.New(msg)
	}

	return mountpoint, nil
}

// mountSharedVolume - mount the shared volume according to volume metadata
func (d *VolumeDriver) mountSharedVolume(volName string, mountpoint string, volRecord *VolumeMetadata) error {
	// Build mount command as follows:
	//   mount [-t $fstype] [-o $options] [$source] $target
	mountArgs := []string{}
	mountArgs = append(mountArgs, "-t", fsType)

	options := []string{
		"username=" + volRecord.Username,
		"password=" + volRecord.Password,
		"port=" + strconv.Itoa(volRecord.Port),
		"vers=3.0",
	}
	mountArgs = append(mountArgs, "-o", strings.Join(options, ","))

	_, addr, _, err := d.dockerOps.GetSwarmInfo()
	if err != nil {
		log.WithFields(
			log.Fields{"volume name": volName,
				"error": err,
			}).Error("Failed to get IP address from docker swarm ")
		return err
	}
	source := "//" + addr + "/" + dockerops.FileShareName
	mountArgs = append(mountArgs, source)
	mountArgs = append(mountArgs, mountpoint)

	log.WithFields(
		log.Fields{"volume name": volName,
			"arguments": mountArgs,
		}).Info("Mounting volume with options ")
	command := exec.Command("mount", mountArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		log.WithFields(
			log.Fields{"volume name": volName,
				"output": string(output),
				"error":  err,
			}).Error("Mount failed: ")
		return err
	}

	return nil
}

// Unmount request from Docker. If mount refcount is drop to 0.
func (d *VolumeDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")

	// lock the state
	d.RefCounts.StateMtx.Lock()
	defer d.RefCounts.StateMtx.Unlock()

	if d.RefCounts.IsInitialized() != true {
		// if refcounting hasn't been succesful,
		// no refcounting, no unmount. All unmounts are delayed
		// until we succesfully populate the refcount map
		d.RefCounts.MarkDirty()
		return volume.Response{Err: ""}
	}

	return d.processUnmount(r)
}

// processUnMount -  process a unmount request
func (d *VolumeDriver) processUnmount(r volume.UnmountRequest) volume.Response {
	if fullVolName, exist := d.MountIDtoName[r.ID]; exist {
		r.Name = fullVolName
		delete(d.MountIDtoName, r.ID) //cleanup the map
	} else {
		msg := fmt.Sprintf("Unable to find volume %v.", r.Name)
		return volume.Response{Err: msg}
	}

	// if refcount has been succcessful, Normal flow
	// if the volume is still used by other containers, just return OK
	refcnt, err := d.DecrRefCount(r.Name)
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

// UnmountVolume - Request detach and then unmount the volume.
func (d *VolumeDriver) UnmountVolume(name string) error {
	mountpoint := d.GetMountPoint(name)
	err := fs.Unmount(mountpoint)
	if err != nil {
		log.WithFields(
			log.Fields{"mountpoint": mountpoint, "error": err},
		).Error("Failed to unmount volume. Now trying to detach... ")
		// Do not return error. Continue with detach.
	}

	// Decrease GRef
	log.Infof("Before AtomicDecr")
	err = d.kvStore.AtomicDecr(kvstore.VolPrefixGRef + name)
	if err != nil {
		log.WithFields(
			log.Fields{"name": name,
				"error": err},
		).Error("Failed to derease global refcount when processUnmount ")
		return err
	}

	return nil
}

// Capabilities - Report plugin scope to Docker
func (d *VolumeDriver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}

// DetachVolume - detach a volume from the VM
// do nothing for the shared driver.
func (d *VolumeDriver) DetachVolume(name string) error {
	return nil
}
