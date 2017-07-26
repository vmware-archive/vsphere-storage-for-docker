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

// This util exposes util to invoke remove commands using ssh

package ssh

import (
	"log"
	"os/exec"
	"strings"
)

// InvokeCommandLocally - can be consumed by test directly to invoke
// any command locally.
// cmd: A command string to be executed on the remote host as per
func InvokeCommandLocally(cmd string) (string, error) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		log.Printf("Failed to invoke command [%s]: %v", cmd, err)
	}
	return strings.TrimSpace(string(out[:])), err
}
