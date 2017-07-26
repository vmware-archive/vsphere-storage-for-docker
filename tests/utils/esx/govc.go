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

/*
govc util holds various helper methods to be consumed by testcase.
It levereges govc CLI, parses json response and serves testcase/verification
util need.

For now, leaving the printf() calls commented out.
Once we have debug logs for tests then we can add these.
If some test is misbehaving, then the developer can enable that log and test.
*/

package esx

import (
	"fmt"
	"log"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/constants/esx"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

const (
	esxcliJSON = esx.JSONTypeOutput + "| " + esx.JSONParser
)

// RetrieveVMNameFromIP util retrieves VM  name from passed VM IP
//govc vm.info -vm.ip=10.20.104.62 -json | jq -r .VirtualMachines[].Name
func RetrieveVMNameFromIP(ip string) string {
	// log.Printf("Finding VM name from IP Address [%s]\n", ip)
	cmd := esx.VMInfoByIP + ip + esxcliJSON + esx.VMName
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// GetVMPowerState util retrieves VM's current power state
// govc vm.info -json photon1 | jq -r .VirtualMachines[].Runtime.PowerState
func GetVMPowerState(vmName string) string {
	// log.Printf("Retrieving VM power state for [%s]\n", vmName)
	cmd := esx.VMInfo + esx.JSONTypeOutput + vmName + " | " + esx.JSONParser + esx.VMPowerState
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// PowerOnVM util powers on the VM
// govc vm.power -on=true photon
func PowerOnVM(vmName string) string {
	log.Printf("Powering on VM [%s]\n", vmName)
	cmd := esx.PowerOnVM + vmName
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// PowerOffVM util powers off the VM
// Caution: If a VM is writing to disk when it receives a
// Power Off command, data corruption may occur.
// govc vm.power -off=true photon
func PowerOffVM(vmName string) string {
	log.Printf("Powering off VM [%s]\n", vmName)
	cmd := esx.PowerOffVM + vmName
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// ShutDownVM util shuts down the VM
// govc vm.power -s=true photon
func ShutDownVM(vmName string) string {
	log.Printf("Shutting down VM [%s]\n", vmName)
	cmd := esx.ShutDownVM + vmName
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// GetDatastoreList returns a list of datastore names available
func GetDatastoreList() []string {
	// log.Printf("Finding Datastores available on ESX")
	cmd := esx.DatastoreInfo + esxcliJSON + esx.DatastoreList
	out, _ := ssh.InvokeCommandLocally(cmd)
	return strings.Fields(out)
}

// GetDatastoreByType returns the datastore name of type specified
func GetDatastoreByType(typeName string) string {
	cmd := esx.DatastoreInfo + esxcliJSON + " '.Datastores[].Summary | select(.Type==\"" + typeName + "\") | .Name'"
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// CreateVM creates a vm on the specified ds and esx
func CreateVM(vmName, datastore, networkAdapterType string) string {
	log.Printf("Creating a vm [%s] \n", vmName)
	cmd := esx.VMCreate + datastore + " -on=false -link=false -net.adapter=" + networkAdapterType + " " + vmName
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// DestroyVM deletes a vm
func DestroyVM(vmName string) string {
	log.Printf("Deleting a vm - %s \n", vmName)
	cmd := esx.VMDestroy + vmName
	out, _ := ssh.InvokeCommandLocally(cmd)
	return out
}

// IsVMExist returns true/false based on vm existence
func IsVMExist(vmName string) bool {
	log.Printf("Verifying if vm - %s exists \n", vmName)
	maxAttempt := 15
	waitTime := 2
	for attempt := 0; attempt < maxAttempt; attempt++ {
		vmList, _ := ssh.InvokeCommandLocally(esx.ListVMs + " /ha-datacenter/vm")
		if strings.Contains(vmList, vmName) {
			return true
		}
		misc.SleepForSec(waitTime)
	}
	return false
}

// TakeSnapshot - take a VM snapshot
func TakeSnapshot(vm, snap string) (string, error) {
	cmd := fmt.Sprintf(esx.TakeSnapshot, vm, snap)
	log.Printf("Creating snapshot - %s", cmd)
	out, err := ssh.InvokeCommandLocally(cmd)
	return out, err
}

// RemoveSnapshot - remove a VM snapshot
func RemoveSnapshot(vm, snap string) (string, error) {
	cmd := fmt.Sprintf(esx.RemoveSnapshot, vm, snap)
	log.Printf("Removing snapshot - %s", cmd)
	out, err := ssh.InvokeCommandLocally(cmd)
	return out, err
}
