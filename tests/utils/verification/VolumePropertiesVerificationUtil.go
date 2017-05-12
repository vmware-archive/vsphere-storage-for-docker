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

package utils

import (
	"log"
	"strings"
	sshutil "github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
)

var ADMIN_CLI_LS = "/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls "
var DOCKER_CLI_INSPC = "docker volume inspect "

// returns attached to vm field of volume using docker cli
func GetVmAttachedToVolUsingDockerCli(volName string, hostname string) string {
	cmd := DOCKER_CLI_INSPC + " --format '{{index .Status \"attached to VM\"}}' " + volName
	op := ExecCmd(hostname, cmd)
	if op == "" {
		log.Fatal("Null value is returned by docker cli when looking for attached to vm field for volume. Output: ", op)
	}
	return strings.TrimSpace(op)
}

// returns attached to vm field of volume using admin cli
func GetVmAttachedToVolUsingAdminCli(volName string, hostname string) string {
	cmd := ADMIN_CLI_LS + "-c volume,attached-to 2>/dev/null | grep " + volName
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

// returns capacity, attached-to-vm and disk-format field
// for volume using Admin cli
func GetVolumePropertiesAdminCli(volName string, hostname string) string {
	cmd := ADMIN_CLI_LS + "-c volume,attached-to,capacity,disk-format 2>/dev/null | grep " + volName
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

// returns capacity,  attached-to-vm and disk-format field
// for volume using Docker cli
func GetVolumePropertiesDockerCli(volName string, hostname string) string {
	cmd := DOCKER_CLI_INSPC + " --format '{{index .Status.capacity.size}} {{index .Status.diskformat}} {{index .Status \"attached to VM\"}}' " + volName
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

// returns docker version
func GetDockerVersion(hostname string) string {
	cmd := "docker -v"
	out := ExecCmd(hostname, cmd)
	return out
}

//method takes command and host and calls InvokeCommand
//and then returns the output after converting to string
func ExecCmd(hostname string, cmd string) string {
	out, err := sshutil.InvokeCommand(hostname, cmd)
	if err != nil {
		log.Fatal(err)
	}
	return string(out[:])
}

// docker volume inspect on docker 1.11
// returns less fields as compared to 1.11.
// So this method can be useful if we
// do not want to run certain verifications
// on docker 1.11
func IsDockerCliCheckNeeded(ipAddr string) bool {
	dkrVrsn := GetDockerVersion(ipAddr)
	log.Println("Docker version:  ", dkrVrsn)
	if strings.Contains(dkrVrsn, "Docker version 1.11.") {
		return false
	}
	return true
}
