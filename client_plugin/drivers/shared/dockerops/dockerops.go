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

// Docker host related operations
//
// DockerOps struct holds the docker client based on a certain API version
// and docker socket. All the operations which require a docker client should
// be executed through this structure, including docker volume create/remove,
// docker service start/stop, and docker information retrieve.

package dockerops

import (
	"context"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/engine-api/client"
	dockerTypes "github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/swarm"
)

const (
	// dockerAPIVersion: docker engine 1.24 and above support this api version
	dockerAPIVersion = "v1.24"
	// dockerUSocket: Unix socket on which Docker engine is listening
	dockerUSocket = "unix:///var/run/docker.sock"
)

// DockerOps is the interface for docker host related operations
type DockerOps struct {
	Dockerd *dockerClient.Client
}

func NewDockerOps() *DockerOps {
	var d *DockerOps

	client, err := dockerClient.NewClient(dockerUSocket, dockerAPIVersion, nil, nil)
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to create client for Docker ")
		return nil
	}

	d = &DockerOps{
		Dockerd: client,
	}

	return d
}

// GetSwarmInfo - returns the node ID and node IP address in swarm cluster
// also returns if this node is a manager or not
func (d *DockerOps) GetSwarmInfo() (nodeID string, addr string, isManager bool, err error) {
	info, err := d.Dockerd.Info(context.Background())
	if err != nil {
		return
	}

	// if node is not in active swarm mode, return error
	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive {
		err = fmt.Errorf("Swarm node state is not active, local node state: %s",
			string(info.Swarm.LocalNodeState))
		return
	}

	// get the swarmID and IP address of current node
	nodeID = info.Swarm.NodeID
	addr = info.Swarm.NodeAddr
	isManager = info.Swarm.ControlAvailable

	return
}

// GetSwarmManagers - return all the managers according to local docker info
func (d *DockerOps) GetSwarmManagers() ([]swarm.Peer, error) {
	info, err := d.Dockerd.Info(context.Background())
	if err != nil {
		return nil, err
	}

	return info.Swarm.RemoteManagers, nil
}

// IsSwarmLeader - check if nodeID is a swarm leader or not
// this function can only be executed successfully on a swarm manager node
func (d *DockerOps) IsSwarmLeader(nodeID string) (bool, error) {
	node, _, err := d.Dockerd.NodeInspectWithRaw(context.Background(), nodeID)
	if err != nil {
		return false, err
	}

	return node.ManagerStatus.Leader, nil
}

// GetSwarmLeader - return the IP address of the swarm leader
// this function can only be executed successfully on a swarm manager node
func (d *DockerOps) GetSwarmLeader() (string, error) {
	nodes, err := d.Dockerd.NodeList(context.Background(), dockerTypes.NodeListOptions{})
	if err != nil {
		return "", err
	}

	for _, n := range nodes {
		if n.ManagerStatus != nil && n.ManagerStatus.Leader == true {
			return n.ManagerStatus.Addr, nil
		}
	}

	msg := fmt.Sprintf("Failed to get leader for swarm manager")
	return "", errors.New(msg)
}

// VolumeCreate - create volume from docker host with specific volume driver
func (d *DockerOps) VolumeCreate(volumeDriver string, volName string, options map[string]string) error {
	dockerVolOptions := dockerTypes.VolumeCreateRequest{
		Driver:     volumeDriver,
		Name:       volName,
		DriverOpts: options,
	}

	_, err := d.Dockerd.VolumeCreate(context.Background(), dockerVolOptions)

	return err
}

// VolumeCreate - remove volume from docker host with specific volume driver
func (d *DockerOps) VolumeRemove(volName string) error {
	return d.Dockerd.VolumeRemove(context.Background(), volName)
}

// StartSMBServer - Start SMB server
func (d *DockerOps) StartSMBServer(volName string) bool {
	log.Errorf("startSMBServer to be implemented")
	return true
}

// StopSMBServer - Stop SMB server
func (d *DockerOps) StopSMBServer(volName string) bool {
	log.Errorf("stopSMBServer to be implemented")
	return true
}
