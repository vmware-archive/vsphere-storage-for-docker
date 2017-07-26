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

// A home to hold govmomi cli constants.

package esx

const (
	govcCmd = "govc "

	// VmInfo refers to govc query to retrieve vm information
	VMInfo = govcCmd + "vm.info "

	// VMInfoByIP refers to govc query to grab VM IP from VM info object
	VMInfoByIP = VMInfo + "-vm.ip="

	// DatastoreInfo get datastore info from govc
	DatastoreInfo = govcCmd + "datastore.info "

	// DatastoreList get the list of datastore names from govc output
	DatastoreList = ".Datastores[].Name "

	// JSONTypeOutput sets govc response type
	JSONTypeOutput = " -json "

	// JSONParser CLI util to parse json string
	JSONParser = "jq -r "

	govcResponseRoot = ".VirtualMachines[]"

	// VMName JSON object key refers VM name
	VMName = govcResponseRoot + ".Name"

	// VMPowerState JSON object key refers VM power state
	VMPowerState = govcResponseRoot + ".Runtime.PowerState"

	// PowerOnVM refers to govc cli (power on vm)
	PowerOnVM = govcCmd + "vm.power -on=true "

	// PowerOffVM refers to govc cli (power off vm)
	PowerOffVM = govcCmd + "vm.power -off=true "

	// ShutDownVM refers to govc cli (shut down vm)
	ShutDownVM = govcCmd + "vm.power -s=true "

	// VMCreate refers to govc create vm
	VMCreate = govcCmd + "vm.create -ds="

	// VMDestroy refers to govc destroy vm
	VMDestroy = govcCmd + "vm.destroy "

	// ListVMs refers to govc vm ls
	ListVMs = govcCmd + "ls "

	// TakeSnapshot takes a snapshot of a VM
	TakeSnapshot = govcCmd + "snapshot.create -vm %s %s"

	// RemoveSnapshot removes a snapshot of a VM
	RemoveSnapshot = govcCmd + "snapshot.remove -vm %s %s"
)
