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
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/dockerops"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/kvstore"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/kvstore/etcdops"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/utils"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/plugin_utils"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/refcount"
	"strconv"
	"strings"
)

/* Constants
   version:              Version of the shared plugin driver
   internalVolumePrefix: Prefix for the names of internal volumes. These
                         volumes are the actual vmdk backing the shared volumes.
   sambaImageName:       Name of the docker hub image we pull as samba server
   sambaUsername:        Default username for all accessing Samba servers
   sambaPassword:        Default password for accessing all Samba servers
*/
const (
	version              = "vSphere Shared Volume Driver v0.2"
	internalVolumePrefix = "InternalVol"
	// TODO Replace with our own samba later
	sambaImageName = "dperson/samba"
	sambaUsername  = "root"
	sambaPassword  = "badpass"
)

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

	// initialize built-in etcd cluster
	d.kvStore = etcdops.NewKvStore(d.dockerOps)
	if d.kvStore == nil {
		log.Errorf("Failed to create new KV store")
		return nil
	}

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

// MountVolume - Request attach and them mounts the volume.
func (d *VolumeDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	log.Errorf("VolumeDriver MountVolume to be implemented")
	mountpoint := d.GetMountPoint(name)
	return mountpoint, nil
}

// UnmountVolume - Unmounts the volume and then requests detach
func (d *VolumeDriver) UnmountVolume(name string) error {
	log.Errorf("VolumeDriver UnmountVolume to be implemented")
	return nil
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
		Username:       sambaUsername,
		Password:       sambaPassword,
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
		msg += fmt.Sprintf(" Check the status of the volumes belonging to driver %s", d.internalVolumeDriver)
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
		clientFetchSucceeded := true
		// Get a list of host VMs using this volume, if any
		keys := []string{
			kvstore.VolPrefixInfo + r.Name,
		}
		entries, err := d.kvStore.ReadMetaData(keys)
		if err != nil {
			clientFetchSucceeded = false
			log.Warningf("Failed to check which host VMs are using volume %s", r.Name)
		}
		// Unmarshal Info key
		err = json.Unmarshal([]byte(entries[0].Value), &volRecord)
		if err != nil {
			clientFetchSucceeded = false
			log.Warningf("Failed to unmarshal data. %v", err)
		}

		msg = fmt.Sprintf("Remove failed: Failed to set volume state to deleting.")
		if clientFetchSucceeded {
			msg += fmt.Sprintf(" Containers on other host VM are still using volume %s.",
				r.Name)
			msg += fmt.Sprintf(" Host VMs using this volume: %s",
				strings.Join(volRecord.ClientList, ","))
		}
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}

	// Delete internal volume
	log.Infof("Attempting to delete internal volume for %s", r.Name)
	internalVolname := internalVolumePrefix + r.Name
	err := d.dockerOps.VolumeRemove(internalVolname)
	if err != nil {
		msg = fmt.Sprintf("Failed to remove internal volume %s. Reason: %v", r.Name, err)
		msg += fmt.Sprintf(" Check the status of the volumes belonging to driver %s", d.internalVolumeDriver)
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}

	// Delete metadata associated with this volume
	log.Infof("Attempting to delete volume metadata for %s", r.Name)
	err = d.kvStore.DeleteMetaData(r.Name)
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

	log.Errorf("VolumeDriver Mount to be implemented")
	return volume.Response{Err: ""}
}

// Unmount request from Docker. If mount refcount is drop to 0.
func (d *VolumeDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")

	log.Errorf("VolumeDriver Unmount to be implemented")
	return volume.Response{Err: ""}
}

// Capabilities - Report plugin scope to Docker
func (d *VolumeDriver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}
