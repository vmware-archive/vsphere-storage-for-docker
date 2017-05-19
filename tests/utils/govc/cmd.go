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
govc util is going to hold various helper methods to be consumed by testcase.
It levereges govc CLI, parses json response and serves testcase/verification
util need.
*/

package govc

import (
	"log"

	"github.com/vmware/docker-volume-vsphere/tests/constants/govc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// RetrieveVMnameFromIP util retrieves VM  name from passed VM IP
//govc vm.info -vm.ip=10.20.104.62 -json | jq -r .VirtualMachines[].Name
func RetrieveVMnameFromIP(ip string) string {
	cmd := govc.VMInfoByIP + ip + govc.JSONTypeOutput + "| " + govc.JSONParser + govc.VirtualMachineName
	log.Printf("Finding VM name from IP Address [%s]\n", ip)
	return ssh.InvokeCommandLocally(cmd)
}
