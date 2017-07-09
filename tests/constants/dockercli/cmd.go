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

// A home to hold test constants related with docker cli.

package dockercli

const (
	docker        = "docker "
	dockerVol     = docker + "volume "
	dockerNode    = docker + "node "
	dockerService = docker + "service "

	// ListVolumes to list down docker volumes
	ListVolumes = dockerVol + "ls "

	// InspectVolume to grab volume properties
	InspectVolume = dockerVol + "inspect "

	// CreateVolume create a volume with vsphere driver
	CreateVolume = dockerVol + " create --driver=vsphere "

	// RemoveVolume delete volume command
	RemoveVolume = dockerVol + "rm "

	// KillDocker kill docker
	KillDocker = "pkill -9 dockerd "

	// RestartDockerWithSystemd - restart docker with systemctl
	RestartDockerWithSystemd = "systemctl restart docker"

	// RestartDockerService - restart docker service
	RestartDockerService = "service docker restart"

	// VDVSPluginName name of vDVS plugin
	VDVSPluginName = "vsphere "

	// VDVSName name of the vDVS service
	VDVSName = "docker-volume-vsphere"

	// GetVDVSPlugin gets vDVS plugin info
	GetVDVSPlugin = docker + "plugin list --no-trunc | grep " + VDVSPluginName

	// GetVDVSPID get the process id of vDVS plugin
	GetVDVSPID = "pidof " + VDVSName

	// GetDockerPID get docker pid
	GetDockerPID = "pidof dockerd"

	// KillVDVSPlugin kills vDVS plugin
	KillVDVSPlugin = "docker-runc kill "

	// StartVDVSPlugin starts the vDVS plugin
	StartVDVSPlugin = docker + " plugin enable " + VDVSPluginName

	// RunContainer create and run a container
	RunContainer = docker + "run "

	// StartContainer starts a container
	StartContainer = docker + "start "

	// StopContainer stops a container
	StopContainer = docker + "stop "

	// RemoveContainer remove the container
	RemoveContainer = docker + "rm -f "

	// RunCmdInContainer run a command in a running container
	RunCmdInContainer = docker + "exec -t "

	// ContainerImage busybox container image
	ContainerImage = " busybox "

	// TestContainer test busybox container that keeps running
	TestContainer = ContainerImage + " tail -f /dev/null "

	// QueryContainer checks whether container exists or not
	QueryContainer = docker + "ps -aq --filter name="

	// ContainerMountPoint mount point where a volume is mounted inside a container
	ContainerMountPoint = "/vol"

	// ListContainers list all running docker containers
	ListContainers = docker + "ps "

	// ListNodes list all docker swarm nodes
	ListNodes = dockerNode + "ls "

	// CreateService create a docker service
	CreateService = dockerService + "create "

	// ScaleService scale a docker service
	ScaleService = dockerService + "scale "

	// UpdateService updates a docker service
	UpdateService = dockerService + "update "

	// ListService list running docker services
	ListService = dockerService + "ps "

	// RemoveService remove docker services
	RemoveService = dockerService + "rm "

	// StopAllContainers stopping all the containers
	StopAllContainers = docker + "kill $(docker ps -aq)"

	// RemoveAllContainers removing all the containers forcefully
	RemoveAllContainers = docker + "rm $(docker ps -aq) -f"

	// ErrorVolumeCreate Error string prefix when volume creation fails
	ErrorVolumeCreate = "Error response from daemon: create"
)

// Volume properties returned for a volume by inspecting it
var VolumeStatusFields = []string{"access", "attach_as", "allocated", "used", "cloned_from", "created_by_VM", "datastore", "diskformat", "fstype", "disk_status", "attached_vm"}
