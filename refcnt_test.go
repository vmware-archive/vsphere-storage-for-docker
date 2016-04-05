package main

import (
	"bufio"
	"os/exec"
	"testing"
	"time"
)

const (
	cmdName = "/tmp/docker-vmdk-plugin/refcnt_test.sh"
)

// brute force code - just invokes external script and processes results
// see comments in the script for the actual test content and TBDs
func TestRefCnt(t *testing.T) {

	cmd := exec.Command(cmdName)
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

	err = cmd.Wait()
	if err != nil {
		t.Error("Refcounter test failed", err)
	}
}
