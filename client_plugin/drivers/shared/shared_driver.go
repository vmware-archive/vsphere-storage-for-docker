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
	log "github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/engine-api/client"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/utils"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/refcount"
)

const (
	version       = "vSphere Shared Volume Driver v0.2"
	apiVersion    = "v1.24" // docker engine 1.24 and above support this api version
	dockerUSocket = "unix:///var/run/docker.sock"
)

// VolumeDriver - vsphere shared plugin volume driver struct
type VolumeDriver struct {
	utils.PluginDriver
	dockerd *dockerClient.Client
}

// NewVolumeDriver creates driver instance
func NewVolumeDriver(cfg config.Config, mountDir string) *VolumeDriver {
	var d VolumeDriver

	d.RefCounts = refcount.NewRefCountsMap()
	d.RefCounts.Init(&d, mountDir, cfg.Driver)
	d.MountIDtoName = make(map[string]string)
	d.MountRoot = mountDir

	// create new docker client
	cli, err := dockerClient.NewClient(dockerUSocket, apiVersion, nil, nil)
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to create client for Docker ")
		return nil
	}
	d.dockerd = cli

	// initialize built-in etcd cluster
	err = d.initEtcd()
	if err != nil {
		log.Errorf("Failed to InitEtcd")
		return nil
	}

	log.WithFields(log.Fields{
		"version": version,
	}).Info("vSphere shared plugin started ")

	return &d
}

// Get info about a single volume
func (d *VolumeDriver) Get(r volume.Request) volume.Response {
	log.Errorf("VolumeDriver Get to be implemented")
	return volume.Response{Err: ""}
}

// List volumes known to the driver
func (d *VolumeDriver) List(r volume.Request) volume.Response {
	volumes, err := d.etcdList()
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
	statusMap = make(map[string]interface{})
	log.Errorf("VolumeDriver GetVolume to be implemented")
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
	log.Errorf("VolumeDriver Create to be implemented")
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
