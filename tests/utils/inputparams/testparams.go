// Copyright 2016 VMware, Inc. All Rights Reserved.
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

package inputparams

// This file holds basic utility/helper methods for object creation used
// at test methods

import (
	"flag"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"log"

	"github.com/vmware/docker-volume-vsphere/tests/utils/esx"
)

// TestConfig - struct for common test configuration params
type TestConfig struct {
	EsxHost         string
	DockerHosts     []string
	DockerHostNames []string
	DockerHostUuids []string
	Datastores      []string
}

var (
	endPoint1  string
	endPoint2  string
	volumeName string
	once       sync.Once
	config     *TestConfig
)

func init() {

	flag.StringVar(&endPoint1, "H1", "unix:///var/run/docker.sock", "Endpoint (Host1) to connect to")
	flag.StringVar(&endPoint2, "H2", "unix:///var/run/docker.sock", "Endpoint (Host2) to connect to")
	flag.StringVar(&volumeName, "v", "TestVol", "Volume name to use in tests")
	flag.Parse()
}

// GetVolumeName returns the default volume name (set as environment variable or supplied through cli)
func GetVolumeName() string {
	return volumeName
}

// GetUniqueContainerName prepares unique container name with a random generated number
func GetUniqueContainerName(containerName string) string {
	return containerName + "_container_" + GetRandomNumber()
}

// GetUniqueServiceName prepares unique service name with a random generated number
func GetUniqueServiceName(serviceName string) string {
	return serviceName + "_service_" + GetRandomNumber()
}

// GetUniqueVmgroupName prepares unique vmgroup name with a random generated number.
func GetUniqueVmgroupName(vmgroupName string) string {
	return vmgroupName + "_vmgroup_" + GetRandomNumber()
}

// GetUniqueVolumeName prepares unique volume name with a random generated number
func GetUniqueVolumeName(volName string) string {
	return volName + "_volume_" + GetRandomNumber()
}

// GetVolumeNameOfSize returns a random volume name of required length
func GetVolumeNameOfSize(size int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, size)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// GetEndPoint1 returns first VM endpoint supplied through CLI
func GetEndPoint1() string {
	return endPoint1
}

// GetEndPoint2 returns second VM endpoint supplied through CLI
func GetEndPoint2() string {
	return endPoint2
}

// GetSwarmManager1 returns swarm manager node IP from the configured swarm cluster
func GetSwarmManager1() string {
	return os.Getenv("MANAGER1")
}

// GetSwarmWorker1 returns 1st swarm worker node IP from the configured swarm cluster
func GetSwarmWorker1() string {
	return os.Getenv("WORKER1")
}

// GetSwarmWorker2 returns 2nd swarm worker node IP from the configured swarm cluster
func GetSwarmWorker2() string {
	return os.Getenv("WORKER2")

}

// GetSwarmNodes returns all nodes in the configured swarm cluster
func GetSwarmNodes() []string {
	return []string{GetSwarmManager1(), GetSwarmWorker1(), GetSwarmWorker2()}
}

// GetEsxIP returns the ip of the esx
func GetEsxIP() string {
	esxIP := os.Getenv("ESX")
	if esxIP == "" {
		log.Fatal("ESX host not found. Stopping the test run.")
	}
	return esxIP
}

// GetDatastores returns list of datastores
func GetDatastores() []string {
	return esx.GetDatastoreList()
}

// GetTestConfig - returns the configuration of IPs for the
// ESX host, docker hosts and the datastores on the host
func getInstance() *TestConfig {
	noVMName := "no such VM"
	config = new(TestConfig)
	config.EsxHost = GetEsxIP()
	config.DockerHosts = append(config.DockerHosts, os.Getenv("VM1"))
	config.DockerHosts = append(config.DockerHosts, os.Getenv("VM2"))
	if config.DockerHosts[0] == "" || config.DockerHosts[1] == "" {
		log.Fatal("Two docker hosts are needed to run tests.")
	}
	config.DockerHostNames = append(config.DockerHostNames, esx.RetrieveVMNameFromIP(config.DockerHosts[0]))
	config.DockerHostNames = append(config.DockerHostNames, esx.RetrieveVMNameFromIP(config.DockerHosts[1]))
	if config.DockerHostNames[0] == noVMName || config.DockerHostNames[1] == noVMName {
		log.Fatalf("Failed to find vm name for docker hosts - %s , %s ", config.DockerHosts[0], config.DockerHosts[1])
	}
	config.DockerHostUuids = append(config.DockerHostUuids, esx.RetrieveVMUuidFromIP(config.DockerHosts[0]))
	config.DockerHostUuids = append(config.DockerHostUuids, esx.RetrieveVMUuidFromIP(config.DockerHosts[1]))

	config.Datastores = esx.GetDatastoreList()
	if len(config.Datastores) < 1 {
		log.Fatalf("No datastores found. At least one datastore is needed to run tests.")
	}
	return config
}

// GetRandomNumber returns random number
func GetRandomNumber() string {
	min := 99999
	max := 999999
	rand.Seed(time.Now().UTC().UnixNano())
	bytes := min + rand.Intn(max)
	return strconv.Itoa(int(bytes))
}

// GetTestConfig - Creates one instance of TestConfig
func GetTestConfig() *TestConfig {
	once.Do(func() {
		config = getInstance()
	})
	return config
}
