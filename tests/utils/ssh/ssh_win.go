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

// Exposes util to invoke remote commands on windows hosts via ssh.

// +build winutil

package ssh

import (
	"fmt"
	"os/exec"
	"strings"
)

// sshTemplate is the ssh command template for Windows hosts.
const sshTemplate = "/usr/bin/ssh -q -o StrictHostKeyChecking=no root@%s '%s'; exit"

// InvokeCommand invokes the given command on the given host via ssh.
func InvokeCommand(ip, cmdStr string) (string, error) {
	// OpenSSH sessions terminate sporadically when a pty isn't allocated.
	// The -t flag doesn't work with OpenSSH on Windows, so we wrap the ssh call
	// within a bash session as a workaround, so that a pty is created.
	cmd := exec.Command("/bin/bash")
	cmd.Stdin = strings.NewReader(fmt.Sprintf(sshTemplate, ip, cmdStr))
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out[:])), err
}
