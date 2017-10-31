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

// This util provides various helper functions for managing docker swarm cluster.

package dockercli

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/constants/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// ListContainers returns all running docker containers
func ListContainers(ip string) (string, error) {
	log.Printf("Listing all docker containers on VM [%s]\n", ip)
	return ssh.InvokeCommand(ip, dockercli.ListContainers)
}

// ListNodes returns all swarm cluster nodes from the swarm master
func ListNodes(ip string) (string, error) {
	log.Printf("Listing swarm cluster nodes on VM [%s]\n", ip)
	return ssh.InvokeCommand(ip, dockercli.ListNodes)
}

// CreateService creates a docker service
func CreateService(ip, name, opts string) (string, error) {
	log.Printf("Creating a docker service [%s] on VM [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, dockercli.CreateService+"--name "+name+" "+opts)
}

// ListService lists the tasks of a docker service
func ListService(ip, name string) (string, error) {
	log.Printf("Listing docker service [%s] running on VM [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, dockercli.ListService+name)
}

// ScaleService scales one or multiple replicated services
func ScaleService(ip, name string, replicas int) (string, error) {
	log.Printf("Scaling %d replicated services for [%s] on VM [%s]\n", replicas, name, ip)
	return ssh.InvokeCommand(ip, dockercli.ScaleService+name+"="+strconv.Itoa(replicas))
}

// UpdateService updates the service with given command options
func UpdateService(ip, name, opts string) (string, error) {
	log.Printf("Updating service [%s] on VM [%s] with options: %s\n", name, ip, opts)
	return ssh.InvokeCommand(ip, dockercli.UpdateService+name+" "+opts)
}

// StopService stops a docker service
func StopService(ip, name string) (string, error) {
	log.Printf("Stopping docker service [%s] on VM [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, dockercli.StopContainer+name)
}

// RemoveService remove a docker service
func RemoveService(ip, name string) (string, error) {
	log.Printf("Removing docker service [%s] on VM [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, dockercli.RemoveService+name)
}

// GetContainerName returns full container name based on given short name
func GetContainerName(hostName, shortName string) (string, error) {
	cmd := dockercli.ListContainers + "--filter name='" + shortName + "' --format '{{.Names}}'"
	return ssh.InvokeCommand(hostName, cmd)
}

// GetAllContainers returns all running containers on the give host based on the filtering criteria
func GetAllContainers(hostName, filter string) (string, error) {
	cmd := dockercli.ListContainers + "--filter name='" + filter + "' --format '{{.Names}}'"
	return ssh.InvokeCommand(hostName, cmd)
}

// PromoteNode promotes a worker node to manager node
func PromoteNode(hostIP, targetIP string) error {
	log.Printf("Promoting worker node IP %s to manager", targetIP)
	out, err := ssh.InvokeCommand(hostIP, dockercli.ListNodes+"-q")
	if err != nil {
		return err
	}

	IDs := strings.Split(out, "\n")
	for _, nodeID := range IDs {
		nodeIP, err := ssh.InvokeCommand(hostIP, dockercli.InspectNode+nodeID+" --format \"{{.Status.Addr}}\"")
		if err != nil {
			return err
		}
		if nodeIP == targetIP {
			log.Printf("Found the ID of target IP is %s", nodeID)
			_, err = ssh.InvokeCommand(hostIP, dockercli.PromoteNode+nodeID)
			return err
		}
	}

	err = fmt.Errorf("No node from 'docker node ls' matches with target IP")
	return err
}

// DemoteNode demotes a manager node to worker node
func DemoteNode(hostIP, targetIP string) error {
	log.Printf("Demoting manager node ID %s to worker", targetIP)
	out, err := ssh.InvokeCommand(hostIP, dockercli.ListNodes+"-q")
	if err != nil {
		return err
	}

	IDs := strings.Split(out, "\n")
	for _, nodeID := range IDs {
		nodeIP, err := ssh.InvokeCommand(hostIP, dockercli.InspectNode+nodeID+" --format \"{{.ManagerStatus.Addr}}\"")
		if err != nil {
			continue
		}
		nodeIP, _, err = net.SplitHostPort(nodeIP)
		if err != nil {
			continue
		}
		if nodeIP == targetIP {
			log.Printf("Found the ID of target IP is %s", nodeID)
			_, err = ssh.InvokeCommand(hostIP, dockercli.DemoteNode+nodeID)
			return err
		}
	}

	err = fmt.Errorf("No node from 'docker node ls' matches with target IP")
	return err
}
