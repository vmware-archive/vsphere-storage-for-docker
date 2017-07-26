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

// This util provides various helper methods that can be used by different tests to
// fetch information related to vSphere docker-volume-service.

package esx

import (
	"log"

	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// IsVDVSRunning checks if vDVS is running on given VM. This util can be
// useful in scenarios where VM is powered-on and user wants to find out
// if VDVS is up and running to be able to run docker volume commands.
func IsVDVSRunning(ip string) bool {
	log.Printf("Verifying if vDVS is running on vm: %s", ip)
	maxAttempt := 30
	waitTime := 3
	for attempt := 0; attempt < maxAttempt; attempt++ {
		pid, _ := ssh.InvokeCommand(ip, "pidof docker-volume-vsphere")
		if pid != "" {
			log.Printf("Process ID of docker-volume-vsphere is: %s", pid)
			return true
		}
		misc.SleepForSec(waitTime)
	}
	log.Printf("vDVS is not running on VM: %s", ip)
	return false
}
