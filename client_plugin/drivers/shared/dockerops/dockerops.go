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
	"io"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/engine-api/client"
	dockerTypes "github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/swarm"
)

const (
	// dockerAPIVersion: docker engine 1.24 and above support this api version
	dockerAPIVersion = "v1.24"
	// dockerUSocket: Unix socket on which Docker engine is listening
	dockerUSocket = "unix:///var/run/docker.sock"
	// Postfix added to names of Samba services for volumes
	serviceNamePrefix = "vSharedServer"
	// Path where the file server image resides in plugin
	fileServerPath = "/usr/lib/vmware/samba.tar"
	// Driver for the network which Samba services will use
	// for communicating to clients
	networkDriver = "overlay"
	// Name of the Samba server docker image
	sambaImageName = "dperson/samba"
	// Name of the Samba share used to expose a volume
	FileShareName = "share1"
	// Default username for all accessing Samba server mounts
	SambaUsername = "root"
	// Default password for all accessing Samba server mounts
	SambaPassword = "badpass"
	// Port number inside Samba container on which
	// Samba service listens
	defaultSambaPort = 445
	// Time between successive checks for Samba service
	// status to see if service container was launched
	checkDuration = 5 * time.Second
	// Time between successive checks for deleting a volume
	checkSleepDuration = time.Second
	// Timeout to mark Samba service launch as unsuccessful
	sambaRequestTimeout = 30 * time.Second
	// Prefix for internal volume names
	internalVolumePrefix = "InternalVol"
	// Error returned when no Samba service for that volume exists
	noSambaServiceError = "No file service exists"
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

// VolumeInspect - inspect volume from docker host, if failed, return error
func (d *DockerOps) VolumeInspect(volName string) error {
	_, err := d.Dockerd.VolumeInspect(context.Background(), volName)
	return err
}

// StartSMBServer - Start SMB server
// Input - Name of the volume for which SMB has to be started
// Output
//      int:     The overlay network port number on which the
//               newly created SMB server listens. This port
//               is opened on every host VM in the swarm.
//      string:  Name of the SMB service started
//      bool:    Indicated success/failure of the function. If
//               false, ignore other output values.
func (d *DockerOps) StartSMBServer(volName string) (int, string, bool) {
	var service swarm.ServiceSpec
	var options dockerTypes.ServiceCreateOptions

	// Name of the service
	service.Name = serviceNamePrefix + volName
	// The Docker image to run in this service
	service.TaskTemplate.ContainerSpec.Image = sambaImageName

	/* Args which will be passed to the service. These options are
	   * used by the Samba container, not Docker API.
	   * -s: Share related info: Name of the share,
	                             Path in the Samba container that will be shared,
	                             Browsable (yes),
	                             Read only (no),
	                             Guest access allowed by default (no),
	                             Which users can access (all),
	                             Which users are admins? (root)
	                             Writelist: If RO, who can write on the share (root)
	   * -u: Username and Password
	*/
	containerArgs := []string{"-s",
		FileShareName + ";/mount;yes;no;no;all;" +
			SambaUsername + ";" + SambaUsername,
		"-u",
		SambaUsername + ";" + SambaPassword}
	service.TaskTemplate.ContainerSpec.Args = containerArgs

	// Mount a volume on service containers at mount point "/mount"
	var mountInfo []swarm.Mount
	mountInfo = append(mountInfo, swarm.Mount{
		Type:   swarm.MountType("volume"),
		Source: internalVolumePrefix + volName,
		Target: "/mount"})
	service.TaskTemplate.ContainerSpec.Mounts = mountInfo

	// How many containers of this service should be running at a time?
	// Service mode can be Replicated or Global
	var uintContainerNum uint64
	uintContainerNum = 1
	numContainers := swarm.ReplicatedService{Replicas: &uintContainerNum}
	service.Mode = swarm.ServiceMode{Replicated: &numContainers}

	/* Ports that the service wants to expose
	   * Protocol: Samba operates on TCP
	   * TargetPort: The port within the container that we wish to expose.
	                 Port on host VM will get self assigned.
	*/
	var exposedPorts []swarm.PortConfig
	exposedPorts = append(exposedPorts, swarm.PortConfig{
		Protocol:   swarm.PortConfigProtocolTCP,
		TargetPort: defaultSambaPort,
	})

	// service.EndpointSpec is an input for service create.
	// It carries the previous data structure as well as Mode.

	// Mode here is the mode we want to use for service discovery.
	// Outside clients do not know on which node is the service
	// running or how many containers are running inside or their
	// IP addresses. Service discovery mechanisms like Virtual IPs
	// or DNS round robin are used to route packets from
	// 127.0.0.1:port to the service container.

	// swarm.ResolutionModeVIP implies that we want to use
	// virtual IPs for service resolution.
	service.EndpointSpec = &swarm.EndpointSpec{
		Mode:  swarm.ResolutionModeVIP,
		Ports: exposedPorts,
	}

	//Start the service
	resp, err := d.Dockerd.ServiceCreate(context.Background(),
		service, options)
	if err != nil {
		log.Warningf("Failed to create file server for volume %s. Reason: %v",
			volName, err)
		return 0, "", false
	}

	// Wait till service container starts
	ticker := time.NewTicker(checkDuration)
	defer ticker.Stop()
	timer := time.NewTimer(sambaRequestTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			log.Infof("Checking status of file server container...")
			port, isRunning := d.isFileServiceRunning(resp.ID, volName)
			if isRunning {
				return int(port), serviceNamePrefix + volName, isRunning
			}
		case <-timer.C:
			log.Warningf("Timeout reached while waiting for file server container for volume %s",
				volName)
			return 0, "", false
		}
	}
}

// isFileServiceRunning - Checks if a file service container is running
// It takes some time from service being brought up to a
// container for that service to be running.
// Input
//      servID:  ID of the service for which we want to check
//               number of running containers.
//      volName: Volume for which the service was run.
// Output
//      uint32:  Port number of overlay networking which is open on
//               every host VM and on which the service container
//               listens.
//      bool:    Indicates if the service container is actually
//               running or not. If false, ignore the port number.
func (d *DockerOps) isFileServiceRunning(servID string, volName string) (uint32, bool) {
	var port uint32
	// Grep the samba service running for this volume using service ID
	serviceFilters := filters.NewArgs()
	serviceFilters.Add("id", servID)
	services, err := d.Dockerd.ServiceList(context.Background(),
		dockerTypes.ServiceListOptions{Filter: serviceFilters})
	if err != nil {
		log.Warningf("Failed to check if file server for volume %s was started. %v", volName, err)
		return port, false
	}
	if len(services) < 1 {
		log.Warningf("No service returned for volume %s. Service not started properly.", volName)
		return port, false
	}

	port = services[0].Endpoint.Ports[0].PublishedPort
	if port == 0 {
		log.Warning("Bad port number assigned to file service for volume %s", volName)
		return port, false
	}

	// Grep all tasks for the service returned and verify that their states are running
	taskFilter := filters.NewArgs()
	for _, service := range services {
		taskFilter.Add("service", service.ID)
	}
	tasks, err := d.Dockerd.TaskList(context.Background(),
		dockerTypes.TaskListOptions{Filter: taskFilter})
	if err != nil {
		log.Warningf("Failed to get task list for file service for volume %s. %v", volName, err)
		return port, false
	}
	for _, task := range tasks {
		if task.Status.State != swarm.TaskStateRunning {
			log.Infof("File server not running for volume %s", volName)
			return port, false
		}
	}
	return port, true
}

// getServiceIDAndPort - return the file service ID and port for given volume
// Input
//      volName: Volume for which the service was run.
// Output
//		string:  service ID
//      uint32:  Port number of overlay networking which is open on
//               every host VM and on which the service container
//               listens.
//      error:   error returned when it can not can service ID and port number
func (d *DockerOps) getServiceIDAndPort(volName string) (string, uint32, error) {
	// Grep the samba service running using service name
	serviceName := serviceNamePrefix + volName
	serviceFilters := filters.NewArgs()
	serviceFilters.Add("name", serviceName)
	services, err := d.Dockerd.ServiceList(context.Background(),
		dockerTypes.ServiceListOptions{Filter: serviceFilters})
	if err != nil {
		msg := fmt.Sprintf("Failed to find service %v. %v", volName, err)
		log.Warningf(msg)
		return "", 0, errors.New(msg)
	}
	if len(services) < 1 {
		msg := fmt.Sprintf("No service returned with name %s.", volName)
		log.Warningf(msg)
		return "", 0, errors.New(noSambaServiceError)
	}

	port := services[0].Endpoint.Ports[0].PublishedPort
	if port == 0 {
		msg := fmt.Sprintf("Bad port number assigned to file service for volume %s", volName)
		log.Warning(msg)
		return "", 0, errors.New(msg)
	}

	return services[0].ID, port, nil
}

// ListVolumesFromServices - List shared volumes according to current docker services
func (d *DockerOps) ListVolumesFromServices() ([]string, error) {
	var volumes []string
	// Get all the samba service for vShared plugin
	filter := filters.NewArgs()
	filter.Add("name", serviceNamePrefix)
	services, err := d.Dockerd.ServiceList(context.Background(),
		dockerTypes.ServiceListOptions{Filter: filter})
	if err != nil {
		log.Errorf("Failed to get a list of docker services. Error: %v", err)
		return volumes, err
	}

	for _, service := range services {
		volumes = append(volumes,
			strings.TrimPrefix(service.Spec.Name, serviceNamePrefix))
	}

	return volumes, nil
}

// ListVolumesFromInternalVol - List shared volumes according to current internal volumes
func (d *DockerOps) ListVolumesFromInternalVol() ([]string, error) {
	var volumes []string
	filter := filters.NewArgs()
	filter.Add("name", internalVolumePrefix)
	volumeResponse, err := d.Dockerd.VolumeList(context.Background(),
		filter)
	if err != nil {
		log.Errorf("Failed to get a list of internal volumes. Error: %v", err)
		return volumes, err
	}

	for _, volume := range volumeResponse.Volumes {
		volumes = append(volumes,
			strings.TrimPrefix(volume.Name, internalVolumePrefix))
	}

	return volumes, nil
}

// DeleteVolume - delete the internal volume
func (d *DockerOps) DeleteInternalVolume(volName string) {
	internalVolname := internalVolumePrefix + volName
	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	// timeout set to sambaRequestTimeout because the internal volume maybe
	// still in use due to stop of SMB server in progress
	timer := time.NewTimer(sambaRequestTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			err := d.VolumeRemove(internalVolname)
			if err != nil {
				msg := fmt.Sprintf("Failed to remove internal volume for volume %s. Reason: %v.",
					volName, err)

				err = d.VolumeInspect(internalVolname)
				if err != nil {
					msg += fmt.Sprintf(" Failed to inspect internal volume. Error: %v.", err)
					log.Warningf(msg)
					return
				}
				// volume exists, continue waiting and retry removing
				msg += fmt.Sprintf(" Internal volume still in use. Wait and retry before timeout.")
				log.Warningf(msg)
			}
		case <-timer.C:
			// The deletion of internal volume will be handled by garbage collector
			log.Warningf("Timeout to remove internal volume for volume %s.",
				volName)
			return
		}
	}
}

// StopSMBServer - Stop SMB server
// The return values are just to maintain parity with StartSMBServer()
// as both these functions are passed to a nested function as args.
// Input
//      volName: Name of the volume for which the SMB service has to
//               be stopped.
// Output
//      int:     Port number on which the SMB server is listening.
//               Set this to 0 as cleanup.
//      string:  Name of the SMB service. Set to empty.
//      bool:    The result of the operation. True if the service was
//               successfully stopped.
func (d *DockerOps) StopSMBServer(volName string) (int, string, bool) {
	serviceID, _, err := d.getServiceIDAndPort(volName)
	if err != nil {
		return 0, "", false
	}

	//Stop the service
	err = d.Dockerd.ServiceRemove(context.Background(), serviceID)
	if err != nil {
		log.Warningf("Failed to remove file server for volume %s. Reason: %v",
			volName, err)
		return 0, "", false
	}

	// Wait till service container stops
	ticker := time.NewTicker(checkDuration)
	defer ticker.Stop()
	timer := time.NewTimer(sambaRequestTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			log.Infof("Checking status of file server container...")
			serviceID, _, err := d.getServiceIDAndPort(volName)
			if err != nil && err.Error() != noSambaServiceError {
				return 0, "", false
			}
			// service is removed successfully
			if serviceID == "" {
				return 0, "", true
			}
		case <-timer.C:
			log.Warningf("Timeout reached while waiting for file server container for volume %s to stop",
				volName)
			return 0, "", false
		}
	}
}

// loadFileServerImage - Load the file server image present
// in the plugin to Docker images
func (d *DockerOps) LoadFileServerImage() {
	file, err := os.Open(fileServerPath)
	if err != nil {
		log.Errorf("Failed to open file server tarball")
		return
	}
	// ImageLoad takes the tarball as an open file, and a bool
	// value for silently loading the image
	resp, err := d.Dockerd.ImageLoad(context.Background(),
		io.Reader(file),
		true)
	if err != nil {
		log.Errorf("Failed to load file server image: %v", err)
		return
	}
	err = resp.Body.Close()
	if err != nil {
		log.Errorf("Failed to close the file server tarball: %v", err)
		return
	}
	return
}
