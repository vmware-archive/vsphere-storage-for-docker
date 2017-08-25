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

// A thread-safe PowerShell utility that reuses a single session.

// +build windows

package powershell

import (
	"fmt"
	"os"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	ps "github.com/gorillalabs/go-powershell"
	"github.com/gorillalabs/go-powershell/backend"
)

var (
	// shell is the handle to a single open Powershell session.
	shell ps.Shell

	// mutex synchronizes access to the single PowerShell session.
	mutex = &sync.Mutex{}
)

// init creates a PowerShell session.
func init() {
	var err error
	shell, err = ps.New(&backend.Local{})
	if err != nil {
		log.WithField("err", err).Fatal("Failed to create a PowerShell session")
		fmt.Println("Failed to create a PowerShell session")
		os.Exit(1)
	}
}

// Exec executes the given command in a PowerShell session.
func Exec(command string) (string, string, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// A \n character marks the end of command in PowerShell. Therefore, we
	// escape such characters to prevent partial script execution.
	escapedCmd := strings.Replace(command, "\n", " ", -1) + "\n"
	stdout, stderr, err := shell.Execute(escapedCmd)
	if err != nil {
		log.WithFields(log.Fields{"cmd": command, "escapedCmd": escapedCmd, "err": err,
			"stdout": stdout, "stderr": stderr}).Error("Failed to execute PowerShell command")
	}
	return stdout, stderr, err
}

// Exit terminates the PowerShell session.
func Exit() {
	mutex.Lock()
	defer mutex.Unlock()
	shell.Exit()
}
