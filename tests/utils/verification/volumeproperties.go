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

// This util provides various helper methods that can be used by different tests to
// fetch information like capacity, disk-format and attched-to-vm fields
// for volume using docker cli or admin cli.

package verification

import (
	"log"
	"strings"
	"time"

	"github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/constants/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/constants/properties"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

const (
	pollTimeout = 64
)

// GetVMAttachedToVolUsingDockerCli returns attached to vm field of volume using docker cli
func GetVMAttachedToVolUsingDockerCli(volName string, hostname string) string {
	cmd := dockercli.InspectVolume + " --format '{{index .Status \"attached to VM\"}}' " + volName
	op := ExecCmd(hostname, cmd)
	if op == "" {
		log.Fatal("Null value is returned by docker cli when looking for attached to vm field for volume. Output: ", op)
	}
	return strings.TrimSpace(op)
}

// GetVMAttachedToVolUsingAdminCli returns attached to vm field of volume using admin cli
func GetVMAttachedToVolUsingAdminCli(volName string, hostname string) string {
	cmd := admincli.ListVolumes + "-c volume,attached-to 2>/dev/null | grep " + volName
	op := ExecCmd(hostname, cmd)
	volProps := strings.Fields(op)
	if op == "" {
		log.Fatal("Null value is returned by admin cli when looking for attached to vm field for volume. Output: ", op)
	}
	if len(volProps) != 2 {
		log.Fatalf("Admin cli output is expected to consist of two elements only - "+
			"volume name and attached-to-vm status. Actual output %s ", op)
	}
	return volProps[1]
}

// GetVolumePropertiesAdminCli returns capacity, attached-to-vm and disk-format field
// for volume using Admin cli
func GetVolumePropertiesAdminCli(volName string, hostname string) string {
	cmd := admincli.ListVolumes + "-c volume,attached-to,capacity,disk-format 2>/dev/null | grep " + volName
	op := ExecCmd(hostname, cmd)
	if op == "" {
		log.Fatal("Null value is returned by admin cli when looking for, size, disk-format and attached to vm. Output: ", op)
	}
	if len(strings.Fields(op)) != 4 {
		log.Fatalf("Output is expected to consist of four elements only - "+
			"volume name, attached-to-vm status, size and disk-format. Actual output: %s", op)
	}
	return op
}

// GetVolumePropertiesDockerCli returns capacity,  attached-to-vm and disk-format field
// for volume using Docker cli
func GetVolumePropertiesDockerCli(volName string, hostname string) string {
	cmd := dockercli.InspectVolume + " --format '{{index .Status.capacity.size}} {{index .Status.diskformat}} {{index .Status \"attached to VM\"}}' " + volName
	op := ExecCmd(hostname, cmd)
	expctedLen := 0
	if op == "" {
		log.Fatal("Null value is returned by docker cli when looking for, size, disk-format and attached to vm. Output: ", op)
	}
	// converting the output to an array and comparing the length of array is as expected so that we do not see any
	// random strings/values attached to the expected output
	if strings.Contains(op, "<no value>") {
		expctedLen = 4
	} else {
		expctedLen = 3
	}
	if len(strings.Fields(op)) != expctedLen {
		log.Fatalf("Docker cli inpect output is expected to consist of three elements only - "+
			"size, disk-format and attached-to-vm status. Actual output: %s", op)
	}
	return op
}

// CheckVolumeAvailability returns true if the given volume is available
// from the specified VM; false otherwise.
func CheckVolumeAvailability(hostName string, volumeName string) bool {
	log.Printf("Checking volume [%s] availability from VM [%s]\n", volumeName, hostName)

	volumes := GetDockerVolumes(hostName)
	//TODO: add more detailed verification here, e.g. checking volume driver name
	return strings.Contains(volumes, volumeName)
}

// GetDockerVolumes returns all docker volumes available from the given host
func GetDockerVolumes(hostName string) string {
	return ExecCmd(hostName, dockercli.ListVolumes)
}

// VerifyAttachedStatus - verify volume is attached and name of the VM attached
// is consistent on both docker host and ESX
func VerifyAttachedStatus(name, hostName, esxName string) bool {
	log.Printf("Confirming attached status for volume [%s]\n", name)

	vmAttachedHost := GetVMAttachedToVolUsingDockerCli(name, hostName)
	vmAttachedESX := GetVMAttachedToVolUsingAdminCli(name, esxName)
	//expectedVMName := govc.RetrieveVMNameFromIP(hostName)

	//isMatching := ((vmAttachedHost == expectedVMName) && (vmAttachedHost == vmAttachedESX))
	isMatching := (vmAttachedHost == vmAttachedESX)

	if !isMatching {
		//log.Printf("Expected Attached VM name is [%s]", expectedVMName)
		log.Printf("Attached VM name from Docker CLI is [%s]", vmAttachedHost)
		log.Printf("Attached VM name from Admin CLI is [%s]", vmAttachedESX)
	}

	return isMatching
}

// GetVolumeStatusHost - get the volume status on a given host
func GetVolumeStatusHost(name, hostName string) string {
	cmd := dockercli.InspectVolume + " --format '{{index .Status.status}}' " + name
	op := ExecCmd(hostName, cmd)
	return strings.TrimSpace(op)
}

// VerifyDetachedStatus - check if the status gets detached within the timeout
func VerifyDetachedStatus(name, hostName, esxName string) bool {
	log.Printf("Confirming detached status for volume [%s]\n", name)
	for attempt := 0; attempt < pollTimeout; attempt++ {
		time.Sleep(1 * time.Second)
		status := GetVolumeStatusHost(name, hostName)
		if status != properties.DetachedStatus {
			continue
		}
		// this api returnes "detached" in when volume is detached
		status = GetVMAttachedToVolUsingAdminCli(name, esxName)
		if status == properties.DetachedStatus {
			return true
		}
	}
	log.Fatalf("Timed out to poll status\n")
	return false
}

// GetDockerVersion returns docker version
func GetDockerVersion(hostName string) string {
	cmd := "docker -v"
	out := ExecCmd(hostName, cmd)
	return out
}

//ExecCmd method takes command and host and calls InvokeCommand
//and then returns the output after converting to string
func ExecCmd(hostName string, cmd string) string {
	out, err := ssh.InvokeCommand(hostName, cmd)
	if err != nil {
		log.Fatal(err)
	}
	return string(out[:])
}

// IsDockerCliCheckNeeded method can be useful if we
// do not want to run certain verifications
// on docker 1.11
func IsDockerCliCheckNeeded(ipAddr string) bool {
	dockerVer := GetDockerVersion(ipAddr)
	log.Println("Docker version:  ", dockerVer)
	if strings.Contains(dockerVer, "Docker version 1.11.") {
		return false
	}
	return true
}
