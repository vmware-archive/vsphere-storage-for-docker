// Copyright 2016 VMware, Inc. All Rights Reserved.
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

package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	tmpLocation = "/tmp/docker-volume-vsphere"
)

// brute force code - just invokes external script and processes results
// see comments in the script for the actual test content
func TestRefCnt(t *testing.T) {
	err := executeScript(t, "refcnt_test.sh")
	if err != nil {
		t.Error(err)
	}
}

// Executes command 'name'.
//
// Command is looked for in fixed location in tmp, then . and bin
// no parameters are passed
//
// returns cmd.Wait()  (i.e. "The returned error is nil if the command runs,
// has no problems copying stdin, stdout, and stderr, and exits with a zero exit
// status. Otherwise returns error is of type *ExitError")
//
func executeScript(t *testing.T, name string) error {
	var fullPath string
	// find script
	for _, d := range []string{tmpLocation, "scripts", "."} {
		fullPath = filepath.Join(d, name)
		t.Logf("Looking for %s in '%s'\n", name, d)
		_, err := os.Stat(fullPath)
		if err == nil {
			break
		}
		fullPath = ""
	}

	if fullPath == "" {
		t.Fatalf("Script '%s' is not found", name)
	}

	// Execute script with piping output to t.Log
	cmd := exec.Command(fullPath)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("cmd.StdoutPipe failed", err)
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		t.Log("Time     Info\n")
		for scanner.Scan() {
			t.Logf("%s %s\n", time.Now().Format(time.RFC3339), scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		t.Fatal("cmd.Start failed", err)
	}

	return cmd.Wait()
}
