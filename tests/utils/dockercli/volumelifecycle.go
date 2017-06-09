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

// This util is going to hold various helper methods to be consumed by testcase.
// Volume creation, deletion is supported as of now.

package dockercli

import (
	"fmt"
	"log"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/constants/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

const (
	maxRemoveVolAttempt = 15
	waitTime            = 2
	pluginInitError     = "Plugin initialization in progress."
)

// CreateVolume is going to create vsphere docker volume with given name.
func CreateVolume(ip, name string) (string, error) {
	log.Printf("Creating volume [%s] on VM [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, dockercli.CreateVolume+" --name= "+name)
}

// CreateVolumeWithOptions is going to create vsphere docker volume with given name.
func CreateVolumeWithOptions(ip, name, options string) (string, error) {
	log.Printf("Creating volume [%s] with options [%s] on VM [%s]\n", name, options, ip)
	return ssh.InvokeCommand(ip, dockercli.CreateVolume+"--name="+name+" "+options)
}

// AttachVolume - attach volume to container on given host
func AttachVolume(ip, volName, containerName string) (string, error) {
	log.Printf("Attaching volume [%s] on VM [%s]\n", volName, ip)
	return ssh.InvokeCommand(ip, dockercli.RunContainer+" -d -v "+volName+
		":/vol1 --name "+containerName+dockercli.TestContainer)
}

// InspectVolume - fetch the named volume's properties
func InspectVolume(ip, volName string) (string, error) {
	log.Printf("Inspecting volume [%s] on VM [%s]\n", volName, ip)
	return ssh.InvokeCommand(ip, dockercli.InspectVolume+volName)
}

// AttachVolumeWithRestart - attach volume to container on given host
// this util starts the container with restart=always flag so that container
// automatically restarts if killed
func AttachVolumeWithRestart(ip, volName, containerName string) (string, error) {
	log.Printf("Attaching volume [%s] on VM[%s]\n", volName, ip)
	return ssh.InvokeCommand(ip, dockercli.RunContainer+" --restart=always -d -v "+volName+
		":/vol1 --name "+containerName+
		dockercli.TestContainer)
}

// WriteToVolume write data to a given file on given volume
func WriteToVolume(ip, volName, containerName, fileName, data string) (string, error) {
	log.Printf("Writing %s to file %s on volume [%s] from VM[%s]\n", data, fileName, volName, ip)

	writeCmd := " /bin/sh -c 'echo \"" + data + "\" > /vol1/test.txt'"
	return ssh.InvokeCommand(ip, dockercli.RunContainer+" -v "+volName+
		":/vol1 --name "+containerName+dockercli.ContainerImage+
		writeCmd)
}

// ReadFromVolume read content of given file on a given volume
func ReadFromVolume(ip, volName, containerName, fileName string) (string, error) {
	log.Printf("Reading from file %s on volume [%s] from VM[%s]\n", fileName, volName, ip)

	readCmd := " /bin/sh -c 'cat /vol1/" + fileName + "'"
	return ssh.InvokeCommand(ip, dockercli.RunContainer+" -v "+volName+
		":/vol1 --name "+containerName+dockercli.ContainerImage+
		readCmd)
}

// DeleteVolume helper deletes the created volume as per passed volume name.
func DeleteVolume(ip, name string) (string, error) {
	log.Printf("Destroying volume [%s]\n", name)
	var out string
	var err error

	for attempt := 0; attempt < maxRemoveVolAttempt; attempt++ {
		out, err = ssh.InvokeCommand(ip, dockercli.RemoveVolume+name)
		if err != nil && strings.Contains(out, pluginInitError) {
			misc.SleepForSec(waitTime)
			log.Printf("Volume cannot be deleted yet as plugin initialization still in progress. Retrying...")
			continue
		} else {
			break
		}
	}
	return out, err
}

// ListVolumes - runs the docker list volumes command and returns the
// list
func ListVolumes(ip string) (string, error) {
	log.Printf("Listing volumes.")
	return ssh.InvokeCommand(ip, dockercli.ListVolumes)
}

// KillDocker - kill docker daemon. It is restarted automatically
func KillDocker(ip string) (string, error) {
	log.Printf("Killing docker on VM [%s]\n", ip)
	out, err := ssh.InvokeCommand(ip, dockercli.KillDocker)
	misc.SleepForSec(2)

	dockerPID, err := ssh.InvokeCommand(ip, dockercli.GetDockerPID)
	if dockerPID != "" {
		return out, err
	}

	// docker needs manual start using systemctl
	out, err = ssh.InvokeCommand(ip, dockercli.StartDocker)
	misc.SleepForSec(2)
	return out, err
}

// GetVDVSPlugin - get vDVS plugin id
func GetVDVSPlugin(ip string) (string, error) {
	out, err := ssh.InvokeCommand(ip, dockercli.GetVDVSPlugin)
	if out == "" {
		return "", fmt.Errorf("vDVS plugin unavailable")
	}
	return strings.Fields(out)[0], err
}

// GetVDVSPID - gets vDVS process id
func GetVDVSPID(ip string) (string, error) {
	out, err := ssh.InvokeCommand(ip, dockercli.GetVDVSPID)
	if err != nil {
		log.Printf("Unable to get docker-volume-vsphere pid")
		return "", err
	}
	return out, nil
}

// KillVDVSPlugin - kill vDVS plugin. It is restarted automatically
func KillVDVSPlugin(ip string) (string, error) {
	log.Printf("Killing vDVS plugin on VM [%s]\n", ip)

	pluginID, err := GetVDVSPlugin(ip)
	if err != nil {
		return "", err
	}

	oldPID, err := GetVDVSPID(ip)
	if err != nil {
		return "", err
	}

	out, err := ssh.InvokeCommand(ip, dockercli.KillVDVSPlugin+pluginID)
	if err != nil {
		log.Printf("Killing vDVS plugin failed")
		return "", err
	}
	misc.SleepForSec(2)

	newPID, err := GetVDVSPID(ip)
	if err != nil {
		return "", err
	}

	// unsuccessful restart
	if oldPID == newPID {
		return "", fmt.Errorf("vDVS plugin autorestart failed")
	}

	return out, nil
}

// RemoveContainer - remove the container forcefully (stops and removes it)
func RemoveContainer(ip, containerName string) (string, error) {
	log.Printf("Removing container [%s] on VM [%s]\n", containerName, ip)
	return ssh.InvokeCommand(ip, dockercli.RemoveContainer+containerName)
}

// StartContainer - starts an already created the container
func StartContainer(ip, containerName string) (string, error) {
	log.Printf("Starting container [%s] on VM [%s]", containerName, ip)
	return ssh.InvokeCommand(ip, dockercli.StartContainer+containerName)
}

// StopContainer - stops the container
func StopContainer(ip, containerName string) (string, error) {
	log.Printf("Stopping container [%s] on VM [%s]", containerName, ip)
	return ssh.InvokeCommand(ip, dockercli.StopContainer+containerName)
}

// ExecContainer - run a container and then remove it
func ExecContainer(ip, volName, containerName string) (string, error) {
	log.Printf("Attaching volume [%s] on VM [%s]\n", volName, ip)
	out, err := ssh.InvokeCommand(ip, dockercli.RunContainer+" -d --rm -v "+volName+
		":/vol1 --name "+containerName+dockercli.TestContainer)
	if err != nil {
		return out, err
	}
	return ssh.InvokeCommand(ip, dockercli.RemoveContainer+containerName)
}

// IsContainerExist - return true if container exists otherwise false
func IsContainerExist(ip, containerName string) bool {
	log.Printf("Checking container [%s] presence on VM [%s]", containerName, ip)
	out, _ := ssh.InvokeCommand(ip, dockercli.QueryContainer+containerName)
	if out != "" {
		log.Printf("container [%s] is present", containerName)
		return true
	}
	return false
}

// StopAllContainers - stops all the containers on a particular vm
func StopAllContainers(ip string) (string, error) {
	log.Printf("Stopping all containers on VM [%s]\n", ip)
	return ssh.InvokeCommand(ip, dockercli.StopAllContainers)
}

// RemoveAllContainers - removes all the containers on a particular vm
func RemoveAllContainers(ip string) (string, error) {
	log.Printf("Removing all containers on VM [%s]\n", ip)
	return ssh.InvokeCommand(ip, dockercli.RemoveAllContainers)
}

// GetVolumeProperties returns capacity,  attached-to-vm and disk-format field for volume.
func GetVolumeProperties(volumeName, hostName string) (string, error) {
	log.Printf("Getting size, disk-format and attached-to-vm for volume [%s] from vm [%s] using docker cli \n", volumeName, hostName)
	cmd := dockercli.InspectVolume + volumeName + " --format ' {{index .Status.capacity.size}} {{index .Status.diskformat}} {{index .Status \"attached to VM\"}}' | sed -e 's/<no value>/detached/' "
	return ssh.InvokeCommand(hostName, cmd)
}
