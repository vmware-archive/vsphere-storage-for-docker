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

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// This util method is going to create vsphere docker volume with
// defaults.
func CreateDefaultVolume(ip string, name string) ([]byte, error) {

	fmt.Printf("\ncreating volume [%s] on VM[%s]", name, ip)

	return exec.Command("/usr/bin/ssh", strings.Split(os.Getenv("SSH_KEY_OPT"), " ")[0], strings.Split(os.Getenv("SSH_KEY_OPT"), " ")[1], "-q", "-kTax", "-o StrictHostKeyChecking=no", "root@"+ip, "docker volume create --driver=vmdk --name="+name).CombinedOutput()

}

// This helper deletes the created volume as per passed volume name.
func DeleteVolume(name string, ip string) ([]byte, error) {
	fmt.Printf("\ndestroying volume [%s]", name)

	return exec.Command("/usr/bin/ssh", strings.Split(os.Getenv("SSH_KEY_OPT"), " ")[0], strings.Split(os.Getenv("SSH_KEY_OPT"), " ")[1], "-q", "-kTax", "-o StrictHostKeyChecking=no", "root@"+ip, "docker volume rm "+name).CombinedOutput()
}
