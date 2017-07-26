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

// A home to hold esxcli helpers

/*
For now, leaving the printf() calls commented out.
Once we have debug logs for tests then we can add these.
If some test is misbehaving, then the developer can enable that log and test.
*/

package esx

import (
	"log"

	"github.com/vmware/docker-volume-vsphere/tests/constants/esx"
	"github.com/vmware/docker-volume-vsphere/tests/constants/properties"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

type getVMPowerStatus func(string) string

// GetVMProcessID - returns VM process id for the passed VM
// esxcli vm process list | grep -e "photon1" -C 1 | grep "World ID:" | awk '{print $3}'
func GetVMProcessID(esxHostIP, vmName string) string {
	// log.Printf("Retrieving World ID for VM [%s] from ESX [%s]\n", vmName, esxHostIP)
	cmd := esx.ListVMProcess + "| grep -e  " + vmName + " -C 1 | grep 'World ID:' | awk '{print $3}'"
	out, err := ssh.InvokeCommand(esxHostIP, cmd)
	if err != nil {
		log.Fatalf("Failed to invoke command [%s]: %v", cmd, err)
	}
	return out
}

// KillVM - kills VM using esxcli command
// e.g. esxcli vm process kill --type=force --world-id=35713
func KillVM(esxHostIP, vmName string) bool {
	//log.Printf("Killing VM [%s] from ESX [%s]\n", vmName, esxHostIP)

	// Grab worldID/vmProcessID
	processID := GetVMProcessID(esxHostIP, vmName)
	// log.Printf("VM's process ID is: %s", processID)

	cmd := esx.KillVMProcess + processID
	_, err := ssh.InvokeCommand(esxHostIP, cmd)
	if err != nil {
		log.Printf("Failed to invoke command [%s]: %v", cmd, err)
		return false
	}
	return true
}

// WaitForExpectedState checks if power state is same as expected
func WaitForExpectedState(fn getVMPowerStatus, vmName, expectedState string) bool {
	// log.Printf("Confirming [%s] power status for vm [%s]\n", expectedState, vmName)
	maxAttempt := 20
	waitTime := 3
	for attempt := 0; attempt < maxAttempt; attempt++ {
		status := fn(vmName)
		if status == expectedState {
			return true
		}
		misc.SleepForSec(waitTime)
	}
	log.Printf("Timed out to poll status\n")
	return false
}

// IsVDVSRunningAfterVMRestart checks if vDVS is up and running after VM being restarted.
func IsVDVSRunningAfterVMRestart(vmIP, vmName string) bool {
	isStatusOnline := WaitForExpectedState(GetVMPowerState, vmName, properties.PowerOnState)
	if !isStatusOnline {
		log.Printf("VM %s is not in PoweredOn state", vmName)
		return false
	}

	isVDVSRunning := IsVDVSRunning(vmIP)
	if !isVDVSRunning {
		log.Printf("vDVS is not running on VM [%s]. Please check if the previous VM IP [%s] has changed.", vmName, vmIP)
		return false
	}

	return true
}
