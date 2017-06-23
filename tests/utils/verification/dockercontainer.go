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

// This util provides various helper methods that can be used by different tests
// to do verification in on of docker containers and docker service itself.

package verification

import (
	"log"
	"strconv"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
)

// IsDockerServiceRunning returns true if the given service is running on
// the specified VM with specified replicas; false otherwise.
func IsDockerServiceRunning(hostName, serviceName string, replicas int) bool {
	log.Printf("Checking docker service [%s] status on VM [%s]\n", serviceName, hostName)
	out, err := dockercli.ListService(hostName, serviceName)
	if err != nil {
		log.Println(err)
		return false
	}

	for i := 1; i <= replicas; i++ {
		if !strings.Contains(out, serviceName+"."+strconv.Itoa(i)) {
			return false
		}
	}
	return true
}

// IsDockerContainerRunning returns true and the host IP if the containers are running on one of
// the given docker hosts with specified replicas; otherwise returns false and empty string.
func IsDockerContainerRunning(dockerHosts []string, serviceName string, replicas int) (bool, string) {
	log.Printf("Checking running containers for docker service [%s] on docker hosts: %v\n", serviceName, dockerHosts)

	//TODO: Need to implement generic polling logic for better reuse
	for attempt := 0; attempt < 30; attempt++ {
		status, host := isContainerRunning(dockerHosts, serviceName, replicas)
		if status {
			return status, host
		}
		misc.SleepForSec(5)
	}
	return false, ""
}

// isContainerRunning returns true if all replicated containers are up and running; false otherwise.
func isContainerRunning(dockerHosts []string, serviceName string, replicas int) (bool, string) {
	for _, host := range dockerHosts {
		out, err := dockercli.GetAllContainers(host, serviceName)
		if err != nil || out == "" {
			continue
		}

		log.Printf("Containers running on docker host [%s]: %s\n", host, out)
		for i := 1; i <= replicas; i++ {
			if !strings.Contains(out, serviceName+"."+strconv.Itoa(i)) {
				return false, ""
			}
		}
		return true, host
	}
	return false, ""
}
