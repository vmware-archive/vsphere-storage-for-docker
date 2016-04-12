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
	tmpLocation = "/tmp/docker-vmdk-plugin"
)

// brute force code - just invokes external script and processes results
// see comments in the script for the actual test content and TBDs
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
		var since time.Duration
		start := time.Now()
		t.Log("Duration    Info\n")
		for scanner.Scan() {
			since = time.Since(start)
			start = time.Now()
			t.Logf("%2.3fs  %s\n", since.Seconds(), scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		t.Fatal("cmd.Start failed", err)
	}

	return cmd.Wait()
}
