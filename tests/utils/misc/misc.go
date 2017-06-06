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

// This util is holding misc small functions that can be reused

package misc

import (
	"log"
	"time"
)

type getVMPowerStatus func(string) string

const (
	maxAttempt = 20
	waitTime = 5
)

// SleepForSec sleep for a given number of seconds
func SleepForSec(sec int) {
	log.Printf("Sleep for %d seconds", sec)
	time.Sleep(time.Duration(sec) * time.Second)
}

// WaitForExpectedState test
func WaitForExpectedState(fn getVMPowerStatus, vmName, expectedState string) bool {

	log.Printf("Confirming [%s] power status for vm [%s]\n", expectedState, vmName)
	for attempt := 0; attempt < maxAttempt; attempt++ {
		SleepForSec(waitTime)
		status := fn(vmName)
		if status == expectedState {
			return true
		}
	}
	log.Printf("Timed out to poll status\n")
	return false
}
// LogTestStart - Print a start log with given test name and
// current time stamp
func LogTestStart(testGroup string, testName string) {
	log.Printf("START:%s %s %s", testGroup, testName, curTime())
}

// LogTestEnd - Print a stop log with given test name and
// current time stamp
func LogTestEnd(testGroup string, testName string) {
	log.Printf("END:%s %s %s", testGroup, testName, curTime())
}

func curTime() string {
	return time.Now().Format(time.UnixDate)
}
