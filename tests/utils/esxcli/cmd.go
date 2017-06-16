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

package esxcli

import (
	"log"

	"github.com/vmware/docker-volume-vsphere/tests/constants/esxcli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// GetVMProcessID - returns VM process id for the passed VM
// esxcli vm process list | grep -e "photon1" -C 1 | grep "World ID:" | awk '{print $3}'
func GetVMProcessID(esxHostIP, vmName string) string {
	log.Printf("Retrieving process ID for VM [%s] from ESX [%s]\n", vmName, esxHostIP)
	cmd := esxcli.ListVMProcess + "| grep -e  " + vmName + " -C 1 | grep 'World ID:' | awk '{print $3}'"
	out, err := ssh.InvokeCommand(esxHostIP, cmd)
	if err != nil {
		log.Fatalf("Failed to invoke command [%s]: %v", cmd, err)
	}
	return out
}

// KillVM - kills VM using esxcli command
// e.g. esxcli vm process kill --type=force --world-id=35713
func KillVM(esxHostIP, vmName string) bool {
	log.Printf("Killing VM [%s] from ESX [%s]\n", vmName, esxHostIP)

	// Grab worldID/vmProcessID
	processID := GetVMProcessID(esxHostIP, vmName)
	log.Printf("VM's process ID is: %s", processID)

	cmd := esxcli.KillVMProcess + processID
	_, err := ssh.InvokeCommand(esxHostIP, cmd)
	if err != nil {
		log.Printf("Failed to invoke command [%s]: %v", cmd, err)
		return false
	}
	return true
}
