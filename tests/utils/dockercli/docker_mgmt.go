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
	dockerRestartCmd = "pidof systemd && " + dockercli.RestartDockerWithSystemd + " || " + dockercli.RestartDockerService
	dockerInfoCmd = "docker info"
	restartWait = 6
	maxRestartRetries = 10

)

// DOCKER AND PLUGIN MGMT API
// --------------------------

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

// RestartDocker - restarts the docker service (graceful)
func RestartDocker(ip string) (string, error) {
	log.Printf("Restarting docker ....")
	out, err := ssh.InvokeCommand(ip, dockerRestartCmd)
	if err != nil {
		return "", err
	}
	// Verify Docker daemon is available
	retries := 0
	out, err = ssh.InvokeCommand(ip, dockerInfoCmd)
	for retries < maxRestartRetries && err != nil {
		misc.SleepForSec(restartWait)
		out, err = ssh.InvokeCommand(ip, dockerInfoCmd)
	}
	return out, err
}

// KillDocker - kill docker daemon. It is restarted automatically (ungraceful)
func KillDocker(ip string) (string, error) {
	log.Printf("Killing docker on VM [%s]\n", ip)
	out, err := ssh.InvokeCommand(ip, dockercli.KillDocker)
	misc.SleepForSec(2)

	dockerPID, err := ssh.InvokeCommand(ip, dockercli.GetDockerPID)
	if dockerPID != "" {
		return out, err
	}

	// docker needs manual start using systemctl/service
	out, err = ssh.InvokeCommand(ip, dockerRestartCmd)
	misc.SleepForSec(2)
	return out, err
}
