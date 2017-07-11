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
	"errors"
        "strings"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"os"
	"path/filepath"
	"time"
	etcdClient "github.com/coreos/etcd/client"
	dockerTypes "github.com/docker/engine-api/types"
	dockerClient "github.com/docker/engine-api/client"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/plugin_utils"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/refcount"
)

const (
	version            = "vSphere Volume Shared Driver v0.1"
	tradVolumePrefix   = "InternalTrad"
	volStateCreate     = "Creating"
	volStateReady      = "Ready"
	volStateMounted    = "Mounted"
	volStateUnmounting = "Unmounting"
)

// VolumeDriver - vsphere shared volume driver struct
type VolumeDriver struct {
	refCounts     *refcount.RefCountsMap
	mountIDtoName map[string]string // map of mountID -> full volume name
}

// Shared volume metadata
type SharedVolumeMetadata struct {
	status         string
	globalRefcount int
	port           int
	username       string
	password       string
	clientList     []string
}

var mountRoot string
var sharedDriverSlave string

// KV store objects
// kvclient is safe for concurrent use by multiple goroutines
var kvcfg etcdClient.Config
var kvclient etcdClient.Client
var kvStoreRequestTimeout time.Duration

// Connection to dockerd
var dockerd *dockerClient.Client

// NewVolumeDriver creates driver instance
func NewVolumeDriver(mountDir string, driverName string,
	kvStorePort string, kvtimeout int, sdslave string) *VolumeDriver {
	var d *VolumeDriver
	var err error

	kvStoreRequestTimeout = time.Duration(kvtimeout)
	sharedDriverSlave = sdslave
        mountRoot = mountDir

	d = &VolumeDriver{
		refCounts: refcount.NewRefCountsMap(),
	}

	d.mountIDtoName = make(map[string]string)
	d.refCounts.Init(d, mountDir, driverName)

	// Initialize the connection to KV store
	kvendpoint := "http://127.0.0.1:"
	kvendpoint = kvendpoint + kvStorePort
	kvcfg = etcdClient.Config {
		Endpoints: []string{kvendpoint},
		Transport: etcdClient.DefaultTransport,
	}
	kvclient, err = etcdClient.New(kvcfg)
	if err != nil {
		log.Warning("Failed to create KV client")
		return nil // This will disable the plugin
	}

	// Initialize connection to dockerd
	dockerd, err = dockerClient.NewEnvClient()
	if err != nil {
		log.Warning("Failed to create connection to dockerd")
		return nil
	}

	log.WithFields(log.Fields{
		"version": version,
	}).Info("Docker vsphere shared plugin started ")

	return d
}

// In following three operations on refcount, if refcount
// map hasn't been initialized, return 1 to prevent detach and remove.

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

// Returns the given volume mountpoint
func getMountPoint(volName string) string {
	return filepath.Join(mountRoot, volName)
}

// Get info about a single volume
func (d *VolumeDriver) Get(r volume.Request) volume.Response {
	log.Errorf("VolumeDriver Get to be implemented")
	return volume.Response{Err: ""}
}

// List volumes known to the driver
func (d *VolumeDriver) List(r volume.Request) volume.Response {
	log.Errorf("VolumeDriver List to be implemented")
	return volume.Response{Err: ""}
}

// GetVolume - return volume meta-data.
func (d *VolumeDriver) GetVolume(name string) (map[string]interface{}, error) {
	var statusMap map[string]interface{}
	statusMap = make(map[string]interface{})
	log.Errorf("VolumeDriver GetVolume to be implemented")
	return statusMap, nil
}

// MountVolume - Request attach and them mounts the volume.
func (d *VolumeDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	log.Errorf("VolumeDriver MountVolume to be implemented")
	mountpoint := getMountPoint(name)
	return mountpoint, nil
}

// UnmountVolume - Unmounts the volume and then requests detach
func (d *VolumeDriver) UnmountVolume(name string) error {
	log.Errorf("VolumeDriver UnmountVolume to be implemented")
	return nil
}

// Update or Create volume metadata in KV store
func writeVolMetadata(name string,
	record SharedVolumeMetadata,
	isCreate bool) error {

	var volname string
	var msg string
	var value string
	var err error

	value = record.status + ":"
	value += fmt.Sprintf("%d:", record.globalRefcount)
	value += fmt.Sprintf("%d:", record.port)
	value += record.username + ":"
	value += record.password + ":"
	for _, element := range record.clientList {
		value += element
		value += "?"
	}

	volname = "/" + name
	kAPI := etcdClient.NewKeysAPI(kvclient)
	ctx, cancel := context.WithTimeout(context.Background(),
		kvStoreRequestTimeout*time.Second)
	defer cancel()
	if isCreate {
		_, err = kAPI.Create(ctx, volname, value)
	} else {
		_, err = kAPI.Update(ctx, volname, value)
	}
	if err != nil {
		msg = fmt.Sprintf("Failed to write metadata entry for volume %s. Reason: %v", name, err)
		log.Warningf(msg)
		return errors.New(msg)
	}

	return nil
}

// Read volume metadata from KV store
func readVolumeMetadata(name string) (error, SharedVolumeMetadata) {
        var err error
        var value string
        var resp etcdClient.Response
        var record SharedVolumeMetadata

	volname := "/" + name
	kAPI := etcdClient.NewKeysAPI(kvclient)
	ctx, cancel := context.WithTimeout(context.Background(),
		kvStoreRequestTimeout*time.Second)
	defer cancel()

        resp, err = kAPI.Get(ctx, volname, nil)
        if err != nil {
		msg = fmt.Sprintf("Failed to read metadata entry for volume %s. Reason: %v", name, err)
		log.Warningf(msg)
		return errors.New(msg)
        }

        value = resp.Node.Value
        words := strings.Split(value, ":")
        record.status = words[0]
        record.globalRefcount = int(words[1])
        record.port = int(words[2])
        record.username = words[3]
        record.password = words[4]
        hostnames := strings.Split(words[5], "?")
        for _, element := range hostnames {
                if element != "" {
                        record.clientList = append(record.clientList, element)
                }
        }

        return nil, record
}

// Delete volume metadata from KV store
func deleteVolMetadata(name string) error {
	var err error
	volname := "/" + name
	kAPI := etcdClient.NewKeysAPI(kvclient)
	ctx, cancel := context.WithTimeout(context.Background(),
		kvStoreRequestTimeout*time.Second)
	defer cancel()
	_, err = kAPI.Delete(ctx, volname, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to delete metadata entry for volume %s. Reason: %v", name, err)
		log.Warningf(msg)
		return errors.New(msg)
	}

        return nil
}

// Create - create a volume.
func (d *VolumeDriver) Create(r volume.Request) volume.Response {
	log.Infof("Entering shared VolumeDriver Create")
	var msg string
	var hnames []string

        // Initialize volume metadata in KV store
	hostname, err := os.Hostname()
	if err != nil {
		msg = fmt.Sprintf("Failed to fetch hostname during volume creation")
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}
	hnames = append(hnames, hostname)
	volRecord := SharedVolumeMetadata{
		status:         volStateCreate,
		globalRefcount: 0,
		port:           0,
		username:       "root",
		password:       "badpass",
		clientList:     hnames,
	}
	log.Infof("Attempting to write initial metadata entry for %s", r.Name)
	err = writeVolMetadata(r.Name, volRecord, true)
	if err != nil {
		msg = fmt.Sprintf("Failed to create volume %s. Reason: %v",
			r.Name, err)
		log.Warningf(msg)
		return volume.Response{Err: msg}
	}

	// Create traditional volume as backend to shared volume
	msg = fmt.Sprintf("Attempting to create internal volume for %s", r.Name)
	log.Infof(msg)
	internalVolname := tradVolumePrefix + r.Name
	dockerVolOptions := dockerTypes.VolumeCreateRequest {
		Driver:     "vsphere", //TODO read this from config file
		Name:       internalVolname,
		DriverOpts: r.Options,
	}
	_, err = dockerd.VolumeCreate(context.Background(), dockerVolOptions)
	if err != nil {
                omsg := fmt.Sprintf("Failed to create volume %s. Reason: %v",
			r.Name, err)
		log.Warningf(omsg)

                // Attempt to delete the metadata for this volume
		err = deleteVolMetadata(r.Name)
		if err != nil {
			msg = fmt.Sprintf("%v. Please remove metadata entry manually using etcd CLI.", err)
			log.Warningf(msg)
			omsg = omsg + msg
		}
		return volume.Response{Err: omsg}
	}

	// Update metadata to indicate successful volume creation
	volRecord = SharedVolumeMetadata {
		status:         volStateReady,
		globalRefcount: 0,
		port:           0,
		username:       "root",
		password:       "badpass",
		clientList:     hnames,
	}
	err = writeVolMetadata(r.Name, volRecord, false)
	if err != nil {
		omsg := fmt.Sprintf("Failed to create volume %s. Reason: %v", r.Name, err)
		log.Warningf(omsg)

		// Attempt to remove the backing trad volume
		log.Infof("Attempting to delete internal volume")
		err = dockerd.VolumeRemove(context.Background(), internalVolname)
		if err != nil {
			msg = fmt.Sprintf("Failed to remove internal volume. Please remove this volume manually using docker cli. Reason: %v ", err)
			log.Warningf(msg)
			omsg = omsg + msg
		}

		// Attempt to delete the metadata for this volume
		err = deleteVolMetadata(r.Name)
		if err != nil {
			msg = fmt.Sprintf("%v. Please remove metadata entry manually using etcd CLI.", err)
			log.Warningf(msg)
			omsg = omsg + msg
		}

		return volume.Response{Err: omsg}
	}

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

	log.Errorf("VolumeDriver Remove to be implemented")
	return volume.Response{Err: ""}
}

// Path - give docker a reminder of the volume mount path
func (d *VolumeDriver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: getMountPoint(r.Name)}
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
	d.refCounts.StateMtx.Lock()
	defer d.refCounts.StateMtx.Unlock()

	log.Errorf("VolumeDriver Mount to be implemented")
	return volume.Response{Err: ""}
}

// Unmount request from Docker. If mount refcount is drop to 0.
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

	log.Errorf("VolumeDriver Unmount to be implemented")
	return volume.Response{Err: ""}
}

// Capabilities - Report plugin scope to Docker
func (d *VolumeDriver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}
