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
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

const (
	maxRemoveVolAttempt = 15
	waitTime            = 2
	pluginInitError     = "Plugin initialization in progress."
)

// VOLUME API
// ----------

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
	return ssh.InvokeCommand(ip, dockercli.RunContainer+" --volume-driver vsphere -d -v "+volName+
		":"+dockercli.ContainerMountPoint+" --name "+containerName+dockercli.TestContainer)
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
	return ssh.InvokeCommand(ip, dockercli.RunContainer+" --restart=always --volume-driver vsphere -d -v "+volName+
		":"+dockercli.ContainerMountPoint+" --name "+containerName+
		dockercli.TestContainer)
}

// WriteToVolumeWithRestart this util does following:
// 1. Attach the volume with restart flag
// 2. Create a file with given name on volume
// 3. Keep container running
// Need restart flag so that container comes up automatically if it is stopped.
func WriteToVolumeWithRestart(ip, volName, containerName, fileName string) (string, error) {
	log.Printf("Attaching volume [%s] on VM[%s]\n", volName, ip)
	writeCmd := " /bin/sh -c 'touch " + dockercli.ContainerMountPoint + "/" + fileName + "; sync ; tail -f /dev/null'"

	return ssh.InvokeCommand(ip, dockercli.RunContainer+" --restart=always -d -v "+volName+
		":"+dockercli.ContainerMountPoint+" --name "+containerName+dockercli.ContainerImage+writeCmd)
}

// ListFilesOnVolume return output of list of files on volume
// Called when we need to list files on a volume
// This output is actual output (in form of string) of ls command on volume
func ListFilesOnVolume(ip, volName string) (string, error) {
	log.Printf("Listing files on volume [%s] on VM[%s]\n", volName, ip)
	listCmd := " /bin/sh -c 'ls -1 " + dockercli.ContainerMountPoint + "/'"

	return ssh.InvokeCommand(ip, dockercli.RunContainer+" --rm -v "+volName+
		":"+dockercli.ContainerMountPoint+" "+dockercli.ContainerImage+listCmd)
}

// WriteToVolume write data to a given file on given volume by starting a new container
func WriteToVolume(ip, volName, containerName, fileName, data string) (string, error) {
	log.Printf("Writing %s to file %s on volume [%s] from VM[%s]\n", data, fileName, volName, ip)

	writeCmd := " /bin/sh -c 'echo \"" + data + "\" > " + dockercli.ContainerMountPoint + "/test.txt'"
	return ssh.InvokeCommand(ip, dockercli.RunContainer+"--rm -v "+volName+
		":"+dockercli.ContainerMountPoint+" --name "+containerName+dockercli.ContainerImage+
		writeCmd)
}

// ReadFromVolume read content of given file on a given volume by starting a new container
func ReadFromVolume(ip, volName, containerName, fileName string) (string, error) {
	log.Printf("Reading from file %s on volume [%s] from VM[%s]\n", fileName, volName, ip)

	readCmd := " /bin/sh -c 'cat " + dockercli.ContainerMountPoint + "/" + fileName + "'"
	return ssh.InvokeCommand(ip, dockercli.RunContainer+"--rm -v "+volName+
		":"+dockercli.ContainerMountPoint+" --name "+containerName+dockercli.ContainerImage+
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
