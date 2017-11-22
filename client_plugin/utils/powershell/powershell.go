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
	"strings"

	log "github.com/Sirupsen/logrus"
	ps "github.com/gorillalabs/go-powershell"
	"github.com/gorillalabs/go-powershell/backend"
)

var (
	// shell is the handle to a single open Powershell session.
	shell ps.Shell
)

// Exec executes the given command in a PowerShell session.
func Exec(command string) (string, string, error) {
	shell, _ = ps.New(&backend.Local{})
	defer shell.Exit()
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
