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
	"time"

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
	time.Sleep(2 * time.Second)
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

// StartVDVSPlugin - manually start vDVS plugin
func StartVDVSPlugin(ip string) ([]byte, error) {
	log.Printf("Manually starting vDVS plugin on VM [%s]\n", ip)
	out, err := ssh.InvokeCommand(ip, dockercli.StartVDVSPlugin)
	return out, err
}

// KillVDVSPlugin - kill vDVS plugin. It is restarted automatically
func KillVDVSPlugin(ip, pluginID string) ([]byte, error) {
	out, err := ssh.InvokeCommand(ip, dockercli.KillVDVSPlugin+pluginID)
	time.Sleep(2 * time.Second)
	return out, err
}

// RestartVDVSPlugin - kill vDVS plugin. It is restarted automatically
// if autorestart failed, start it manually
func RestartVDVSPlugin(ip string) ([]byte, error) {
	log.Printf("Restarting vDVS plugin on VM [%s]\n", ip)

	pluginID, err := GetVDVSPlugin(ip)
	if err != nil {
		log.Printf("vDVS plugin unavailable")
		return nil, err
	}

	out, err := KillVDVSPlugin(ip, pluginID)
	if err != nil {
		log.Printf("Killing vDVS plugin failed")
		return out, err
	}

	pluginID, err = GetVDVSPlugin(ip)
	// plugin autorestart succeeded i.e. there is an entry for vDVS plugin
	if pluginID != "" {
		return out, err
	}

	// manually restart plugin if autorestart wasn't successful
	return StartVDVSPlugin(ip)
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
