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
	"log"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/constants/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// CONTAINER MGMT API
// ------------------

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

// WriteToContainer write data to a file in the existing running container
func WriteToContainer(ip, containerName, volPath, fileName, data string) (string, error) {
	log.Printf("Writing data to file [%s] in container [%s] on VM [%s]\n", fileName, containerName, ip)

	writeCmd := " /bin/sh -c 'echo \"" + data + "\" > " + volPath + "/" + fileName + "'"
	fullCmd := dockercli.RunCmdInContainer + containerName + writeCmd

	log.Println(fullCmd)
	return ssh.InvokeCommand(ip, fullCmd)
}

// ReadFromContainer read content from the file in the existing running container
func ReadFromContainer(ip, containerName, volPath, fileName string) (string, error) {
	log.Printf("Reading from file [%s] in container [%s] on VM [%s]\n", fileName, containerName, ip)

	readCmd := " /bin/sh -c 'cat " + volPath + "/" + fileName + "'"
	fullCmd := dockercli.RunCmdInContainer + containerName + readCmd

	log.Println(fullCmd)
	return ssh.InvokeCommand(ip, fullCmd)
}

// GetVolumeStatus returns a property map for a volume 
func GetVolumeStatus(hostName, volumeName string) (map[string]string, error) {
 	formatStr1 := " --format '{{index .Status.access}} {{index .Status \"attach-as\"}} {{index .Status.capacity.allocated}} {{index .Status.capacity.size}} {{index .Status \"clone-from\"}}"
	formatStr2 := " {{index .Status \"created by VM\"}} {{index .Status.datastore}} {{index .Status.diskformat}} {{index .Status.fstype}} {{index .Status.status}} {{index .Status \"attached to VM\"}}'"

	cmd := dockercli.InspectVolume + volumeName + formatStr1 + formatStr2
	out, err := ssh.InvokeCommand(hostName, cmd)

	if err != nil {
		return nil, err
	}

	status := make(map[string]string)
	val := strings.Fields(out)

	for i := 0; i < len(dockercli.VolumeStatusFields); i += 1 {
		status[dockercli.VolumeStatusFields[i]] = val[i]
	}
	return status, nil
}
