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

// Exposes util to invoke remote commands on non-windows hosts via ssh.

// +build !winutil

package ssh

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

// sshKeyOptPath is the option to specify the rsa key while connecting via ssh.
const sshKeyOptPath = "-i /root/.ssh/id_rsa"

// InvokeCommand - can be consumed by test directly to invoke
// any command on the remote host.
// ip: remote machine address to execute on the machine
// cmd: A command string to be executed on the remote host as per
func InvokeCommand(ip, cmd string) (string, error) {
	sshKeyOpt := strings.Split(os.Getenv("SSH_KEY_OPT"), " ")
	if sshKeyOpt == nil {
		sshKeyOpt = strings.Split(sshKeyOptPath, " ")
	}
	sshIdentity := []string{sshKeyOpt[0], sshKeyOpt[1], "-q", "-kTax", "-o StrictHostKeyChecking=no"}

	log.Printf("CMD is :%s", cmd)

	out, err := exec.Command("/usr/bin/ssh", append(sshIdentity, "root@"+ip, cmd)...).CombinedOutput()
	log.Printf("OUT is :%s", out)
	if err != nil {
		log.Printf("ERR is :%s", err.Error())
	}
	return strings.TrimSpace(string(out[:])), err
}
