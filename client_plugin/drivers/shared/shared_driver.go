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
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/engine-api/client"
	dockerTypes "github.com/docker/engine-api/types"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/utils"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/refcount"
	"strconv"
)

// volStatus: Datatype for keeping status of a shared volume
type volStatus string

/* Constants
   version:              Version of the shared plugin driver
   dockerAPIVersion:     docker engine 1.24 and above support this api version
   dockerUSocket:        Unix socket on which Docker engine is listening
   internalVolumePrefix: Prefix for the names of internal volumes. These
                         volumes are the actual vmdk backing the shared volumes.
   sambaImageName:       Name of the docker hub image we pull as samba server
   sambaUsername:        Default username for all accessing Samba servers
   sambaPassword:        Default password for accessing all Samba servers

   volStateCreating:     Shared volume is being created. Not ready to be mounted.
   volStateReady:        Shared volume is ready to be mounted but
                         no Samba service running right now.
   volStateMounted:      Samba service already running. Volume mounted
                         on at least one host VM.
   volStateIntermediate: Metadata of shared volume is being changed.
                         Dont proceed with your op if this is the status.
*/
const (
	version              = "vSphere Shared Volume Driver v0.2"
	dockerAPIVersion     = "v1.24"
	dockerUSocket        = "unix:///var/run/docker.sock"
	internalVolumePrefix = "InternalVol"
	// TODO Replace with our own samba later
	sambaImageName                 = "dperson/samba"
	sambaUsername                  = "root"
	sambaPassword                  = "badpass"
	volStateCreating     volStatus = "Creating"
	volStateReady        volStatus = "Ready"
	volStateMounted      volStatus = "Mounted"
	volStateIntermediate volStatus = "MetadataUpdateInProgress"
	volStateError        volStatus = "Error"
	stateIdx                       = 0
	GRefIdx                        = 1
	InfoIdx                        = 2
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
	Status         volStatus `json:"-"` // Field won't be marshalled
	GlobalRefcount int       `json:"-"` // Field won't be marshalled
	Port           int       `json:"port,omitempty"`
	ServiceName    string    `json:"serviceName,omitempty"`
	Username       string    `json:"username,omitempty"`
	Password       string    `json:"password,omitempty"`
	ClientList     []string  `json:"clientList,omitempty"`
}

/* VolumeDriver - vsphere shared plugin volume driver struct
   dockerd:                 Client used for talking to Docker
   internalVolumeDriver:    Name of the plugin used by shared volume
                            plugin to create internal volumes
*/

// VolumeDriver - Contains vars specific to this driver
type VolumeDriver struct {
	utils.PluginDriver
	dockerd              *dockerClient.Client
	internalVolumeDriver string
	etcd                 *etcdKVS
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

	// create new docker client
	cli, err := dockerClient.NewClient(dockerUSocket, dockerAPIVersion, nil, nil)
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to create client for Docker ")
		return nil
	}
	d.dockerd = cli

	// initialize built-in etcd cluster
	d.etcd = NewKvStore(&d)
	if d.etcd == nil {
		log.Errorf("Failed to create new etcd KV store")
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
	volumes, err := d.etcd.ListVolumeName()
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
		etcdPrefixState + name,
		etcdPrefixGRef + name,
		etcdPrefixInfo + name,
	}

	// KV pairs will be returned in same order in which they were requested
	entries, err := d.etcd.ReadVolMetadata(keys)
	if err != nil {
		if err.Error() == VolumeDoesNotExistError {
			log.Infof("Volume not found: %s", name)
			return statusMap, err
		}
		msg := fmt.Sprintf("Failed to read metadata for volume %s from KV store. %v",
			name, err)
		log.Warningf(msg)
		return statusMap, errors.New(msg)
	}

	statusMap["Volume Status"] = entries[stateIdx].value
	statusMap["Global Refcount"], _ = strconv.Atoi(entries[GRefIdx].value)
	// Unmarshal Info key
	err = json.Unmarshal([]byte(entries[InfoIdx].value), &volRecord)
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
	var entries []kvPair

	// Ensure that the node running this command is a manager
	info, err := d.dockerd.Info(context.Background())
	if err != nil {
		msg = fmt.Sprintf("Failed to get Info from docker client. Reason: %v", err)
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}
	if info.Swarm.ControlAvailable == false {
		msg = fmt.Sprintf("This node is not a swarm manager.")
		msg += fmt.Sprintf(" Shared Volume creation is only possible from swarm manager nodes.")
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}

	// Initialize volume metadata in KV store
	volRecord := VolumeMetadata{
		Status:         volStateCreating,
		GlobalRefcount: 0,
		Username:       sambaUsername,
		Password:       sambaPassword,
	}

	// Append global refcount and status to kv pairs that will be written
	entries = append(entries, kvPair{key: etcdPrefixGRef + r.Name, value: strconv.Itoa(volRecord.GlobalRefcount)})
	entries = append(entries, kvPair{key: etcdPrefixState + r.Name, value: string(volRecord.Status)})
	// Append the rest of the metadata as one KV pair where the data is jsonified
	byteRecord, err := json.Marshal(volRecord)
	if err != nil {
		msg = fmt.Sprintf("Cannot create volume. Failed to marshal metadata to json. Reason: %v", err)
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}
	entries = append(entries, kvPair{key: etcdPrefixInfo + r.Name, value: string(byteRecord)})

	log.Infof("Attempting to write initial metadata entry for %s", r.Name)
	err = d.etcd.WriteVolMetadata(entries)
	if err != nil {
		msg = fmt.Sprintf("Failed to create volume %s. Reason: %v",
			r.Name, err)
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}

	// Create traditional volume as backend to shared volume
	log.Infof("Attempting to create internal volume for %s", r.Name)
	internalVolname := internalVolumePrefix + r.Name
	dockerVolOptions := dockerTypes.VolumeCreateRequest{
		Driver:     d.internalVolumeDriver,
		Name:       internalVolname,
		DriverOpts: r.Options,
	}
	_, err = d.dockerd.VolumeCreate(context.Background(), dockerVolOptions)
	if err != nil {
		msg = fmt.Sprintf("Failed to create internal volume %s. Reason: %v", r.Name, err)
		msg += fmt.Sprintf(" Check the status of the volumes belonging to driver %s", d.internalVolumeDriver)
		log.Warningf(msg)

		// If failed, attempt to delete the metadata for this volume
		err = d.etcd.DeleteVolMetadata(r.Name)
		if err != nil {
			log.Warningf("Failed to remove metadata entry for volume: %s. Reason: %v", r.Name, err)
		}
		return volume.Response{Err: msg}
	}

	// Update metadata to indicate successful volume creation
	log.Infof("Attempting to update volume state to ready for volume: %s", r.Name)
	entries = nil
	entries = append(entries, kvPair{key: etcdPrefixState + r.Name, value: string(volStateReady)})
	err = d.etcd.WriteVolMetadata(entries)
	if err != nil {
		outerMessage := fmt.Sprintf("Failed to set status of volume %s to ready. Reason: %v", r.Name, err)
		log.Warningf(outerMessage)

		// If failed, attempt to remove the backing trad volume
		log.Infof("Attempting to delete internal volume")
		err = d.dockerd.VolumeRemove(context.Background(), internalVolname)
		if err != nil {
			msg = fmt.Sprintf(" Failed to remove internal volume. Reason %v.", err)
			msg += fmt.Sprintf(" Please remove the volume manually. Volume: %s", internalVolname)
			log.Warningf(msg)
			outerMessage = outerMessage + msg
		}

		// Attempt to delete the metadata for this volume
		err = d.etcd.DeleteVolMetadata(r.Name)
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

	log.Errorf("VolumeDriver Remove to be implemented")
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

// startSMBServer - Start SMB server
func (d *VolumeDriver) startSMBServer(volName string) bool {
	log.Errorf("startSMBServer to be implemented")
	return true
}

// stopSMBServer - Stop SMB server
func (d *VolumeDriver) stopSMBServer(volName string) bool {
	log.Errorf("stopSMBServer to be implemented")
	return true
}
