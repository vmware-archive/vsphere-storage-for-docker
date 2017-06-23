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
// do verification on docker host (VM) specific to host environment (eg. logs, mounted devices etc)

package verification

import (
	"log"
	"strconv"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/constants/vm"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// CheckVolumeInProcMounts - check if volume entry is present in docker host mount file
// grep does the work. If entry isn't present, err is not nil (Exit status 1)
func CheckVolumeInProcMounts(hostName, volName string) bool {
	cmd := "grep -q " + volName + vm.MountFile
	_, err := ssh.InvokeCommand(hostName, cmd)

	if err != nil {
		log.Printf("No entry for volume %s in %s.", volName, vm.MountFile)
		return false
	}

	return true
}

// CheckRefCount - check the log record for expected refcount of the volume.
// grep does the work. If refcount isn't the expected one, err is not nil (Exit status 1)
// logLines are number of log lines to be tailed from the log file.
// eg: tail -1 /var/log/docker-volume-vsphere.log | grep testName |  grep -q refcount=5
func CheckRefCount(hostName, volName string, logLines, expectedRefCnt int) bool {
	cmd := "tail -" + strconv.Itoa(logLines) + vm.LogFile + " | grep " + volName + " | grep -q refcount=" +
		strconv.Itoa(expectedRefCnt)
	_, err := ssh.InvokeCommand(hostName, cmd)

	if err != nil {
		log.Printf("Unable to find refcount for %s in logs.", volName)
		return false
	}

	return true
}

// checkRecoveryRecord - check the recovery record printed by plugin after restart
// The log record must contain expected refcount
// logLines are number of log lines to be tailed from the log file.
// eg: tail -50 /var/log/docker-volume-vsphere.log | grep 'Volume name=testName' | grep 'mounted=true'
func checkRecoveryRecord(hostName, volName string, logLines, expectedRefCnt int) bool {
	cmd := "tail -" + strconv.Itoa(logLines) + vm.LogFile + " | grep 'Volume name=" + volName +
		"' | grep 'mounted=true'"
	op, err := ssh.InvokeCommand(hostName, cmd)
	expectedStr := "count=" + strconv.Itoa(expectedRefCnt)
	if err != nil {
		log.Printf("No recovery record for %s in logs.", volName)
		return false
	}

	log.Printf("Recovery record is %s ", op)

	if strings.Contains(op, expectedStr) != true {
		log.Printf("Recovery record %s doesn't match expected refcount %d", op, expectedRefCnt)
		return false
	}
	return true
}

// VerifyRecoveryRecord Verify the recovery record. There is a delay after which recovery record appears in the log
// due to refcounting being delayed in background. (Docker doesn't respond immediately)
func VerifyRecoveryRecord(hostName, volName string, logLines, expectedRefCnt int) bool {
	const retryAttempt = 60

	isRefcounted := false

	for i := 0; i < retryAttempt; i++ {
		isRefcounted = checkRecoveryRecord(hostName, volName, logLines, expectedRefCnt)
		if isRefcounted {
			break
		}
		misc.SleepForSec(2)
	}

	return isRefcounted
}
