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

// This util is going to hold various helper methods to be consumed by testcase.
// Volume creation, deletion is supported as of now.

package dockercli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// sshIdentity an array variable to prepare ssh input parameter to pass identify value
var sshIdentity = []string{strings.Split(os.Getenv("SSH_KEY_OPT"), " ")[0], strings.Split(os.Getenv("SSH_KEY_OPT"), " ")[1], "-q", "-kTax", "-o StrictHostKeyChecking=no"}

// CreateDefaultVolume is going to create vsphere docker volume with
// defaults.
func CreateDefaultVolume(ip string, name string) ([]byte, error) {
	fmt.Printf("\ncreating volume [%s] on VM[%s]", name, ip)
	return InvokeCommand(ip, "docker volume create --driver=vsphere --name="+name)
}

// DeleteVolume helper deletes the created volume as per passed volume name.
func DeleteVolume(name string, ip string) ([]byte, error) {
	fmt.Printf("\ndestroying volume [%s]", name)
	return InvokeCommand(ip, "docker volume rm "+name)
}

// InvokeCommand helper method can be consumed by test directly to invoke
// any command on the remote host.
// remoteHostIP:
// 	remote machine address to execute on the machine
// cmd:
//	A command string to be executed on the remote host as per
//	remoteHostIP value
func InvokeCommand(remoteHostIP string, cmd string) ([]byte, error) {
	return exec.Command("/usr/bin/ssh", append(sshIdentity, "root@"+remoteHostIP, cmd)...).CombinedOutput()
}
