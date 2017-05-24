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
		dockercli.TestContainer)
}

// AttachVolumeWithRestart - attach volume to container on given host
// this util starts the container with restart=always flag so that container
// automatically restarts if killed
func AttachVolumeWithRestart(ip, volName, containerName string) ([]byte, error) {
	log.Printf("Attaching volume [%s] on VM[%s]\n", volName, ip)
	return ssh.InvokeCommand(ip, dockercli.RunContainer+" --restart=always -d -v "+volName+
		":/vol1 --name "+containerName+
		dockercli.TestContainer)
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
	misc.SleepForSec(2)
	return out, err
}

// GetVDVSPlugin - get vDVS plugin id
func GetVDVSPlugin(ip string) (string, error) {
	out, err := ssh.InvokeCommand(ip, dockercli.GetVDVSPlugin)
	if misc.FormatOutput(out) == "" {
		return "", fmt.Errorf("vDVS plugin unavailable")
	}
	return strings.Fields(misc.FormatOutput(out))[0], err
}

// GetVDVSPID - gets vDVS process id
func GetVDVSPID(ip string) (string, error) {
	out, err := ssh.InvokeCommand(ip, dockercli.GetVDVSPID)
	if err != nil {
		log.Printf("Unable to get docker-volume-vsphere pid")
		return "", err
	}
	return strings.TrimSpace(misc.FormatOutput(out)), nil
}

// KillVDVSPlugin - kill vDVS plugin. It is restarted automatically
func KillVDVSPlugin(ip string) ([]byte, error) {
	log.Printf("Killing vDVS plugin on VM [%s]\n", ip)

	pluginID, err := GetVDVSPlugin(ip)
	if err != nil {
		return nil, err
	}

	oldPID, err := GetVDVSPID(ip)
	if err != nil {
		return nil, err
	}

	out, err := ssh.InvokeCommand(ip, dockercli.KillVDVSPlugin+pluginID)
	if err != nil {
		log.Printf("Killing vDVS plugin failed")
		return nil, err
	}
	misc.SleepForSec(2)

	newPID, err := GetVDVSPID(ip)
	if err != nil {
		return nil, err
	}

	// unsuccessful restart
	if oldPID == newPID {
		return nil, fmt.Errorf("vDVS plugin autorestart failed")
	}

	return out, nil
}

// RemoveContainer - remove the container forcefully (stops and removes it)
func RemoveContainer(ip, containerName string) ([]byte, error) {
	log.Printf("Removing container [%s] on VM [%s]\n", containerName, ip)
	return ssh.InvokeCommand(ip, dockercli.RemoveContainer+containerName)
}

// StartContainer - starts an already created the container
func StartContainer(ip, containerName string) ([]byte, error) {
	log.Printf("Starting container [%s] on VM [%s]", containerName, ip)
	return ssh.InvokeCommand(ip, dockercli.StartContainer+containerName)
}

// StopContainer - stops the container
func StopContainer(ip, containerName string) ([]byte, error) {
	log.Printf("Stopping container [%s] on VM [%s]", containerName, ip)
	return ssh.InvokeCommand(ip, dockercli.StopContainer+containerName)
}
