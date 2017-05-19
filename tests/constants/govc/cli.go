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

package govc

const (
	govcCmd = "govc "

	vmInfo = govcCmd + "vm.info "

	// VMInfoByIP refers to govc query to grab VM IP from VM info object
	VMInfoByIP = vmInfo + "-vm.ip="

	// JSONTypeOutput sets govc response type
	JSONTypeOutput = " -json "

	// JSONParser CLI util to parse json string
	JSONParser = "jq -r "

	govcResponseRoot = ".VirtualMachines[]"

	// VirtualMachineName JSON object key refers VM name
	VirtualMachineName = govcResponseRoot + ".Name"
)
