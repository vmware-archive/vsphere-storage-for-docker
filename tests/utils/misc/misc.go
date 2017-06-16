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

// SleepForSec sleep for a given number of seconds
func SleepForSec(sec int) {
	log.Printf("Sleep for %d seconds", sec)
	time.Sleep(time.Duration(sec) * time.Second)
}

// LogTestStart - Print a start log with given test group and test case name
func LogTestStart(testName string) {
	log.Printf("START: %s", testName)
}

// LogTestEnd - Print a stop log with given test group and test case name
func LogTestEnd(testName string) {
	log.Printf("END: %s", testName)
}
