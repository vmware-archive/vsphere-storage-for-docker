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

// This util is holding misc small functions for operations to be done using admincli on esx

package admincli

import (
	"log"

	"github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// CreatePolicy creates a policy
func CreatePolicy(ip, name, content string) (string, error) {
	log.Printf("Creating policy [%s] with content [%s] on ESX [%s]\n", name, content, ip)
	return ssh.InvokeCommand(ip, admincli.CreatePolicy+" --name "+name+" --content "+content)
}

// UpdateVolumeAccess update the volume access as per params
func UpdateVolumeAccess(ip, volName, vmgroup, access string) (string, error) {
	log.Printf("Updating access to [%s] for volume [%s] ", access, volName)
	return ssh.InvokeCommand(ip, admincli.SetVolumeAccess+" --vmgroup="+vmgroup+
		" --volume="+volName+" --options=\"access="+access+"\"")
}
