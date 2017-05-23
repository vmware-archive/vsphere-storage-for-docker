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
	"time"

	"github.com/vmware/docker-volume-vsphere/tests/constants/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// CreateVolume is going to create vsphere docker volume with given name.
func CreateVolume(ip, name string) ([]byte, error) {
	log.Printf("Creating volume [%s] on VM [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, dockercli.CreateVolume+"--name="+name)
}

// AttachVolume - attach volume to container on given host
func AttachVolume(ip, volName, containerName string) ([]byte, error) {
	log.Printf("Attaching volume [%s] on VM [%s]\n", volName, ip)
	return ssh.InvokeCommand(ip, dockercli.RunContainer+"-d -v "+volName+
		":/vol1 --name "+containerName+
		" busybox tail -f /dev/null")
}

// DeleteVolume helper deletes the created volume as per passed volume name.
func DeleteVolume(ip, name string) ([]byte, error) {
	log.Printf("Destroying volume [%s]\n", name)
	return ssh.InvokeCommand(ip, dockercli.RemoveVolume+name)
}

// KillDocker - kill docker daemon. It is restarted automatically
func KillDocker(ip string) ([]byte, error) {
	log.Printf("Killing docker on VM [%s]\n", ip)
	out, err := ssh.InvokeCommand(ip, dockercli.KillDocker)
	time.Sleep(2 * time.Second)
	return out, err
}

// RemoveContainer - remove the container forcefully (stops and removes it)
func RemoveContainer(ip, containerName string) ([]byte, error) {
	log.Printf("Removing container [%s] on VM [%s]\n", containerName, ip)
	return ssh.InvokeCommand(ip, dockercli.RemoveContainer+containerName)
}
