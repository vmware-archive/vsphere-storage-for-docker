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

//
// VMDK Docker driver sanity tests.
//

package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/strslice"
	"github.com/vmware/docker-volume-vsphere/config"
	"golang.org/x/net/context"
	"testing"
	"time"
)

const (
	defaultMountLocation = "/mnt/testvol"
	// tests are often run under regular account and have no access to /var/log
	defaultTestLogPath = "/tmp/test-docker-volume-vsphere.log"
)

var (
	// flag vars - see init() for help
	endPoint1        string
	endPoint2        string
	volumeName       string
	removeContainers bool
)

// prepares the environment. Kind of "main"-ish code for tests.
// Parses flags and inits logs and mount ref counters (the latter waits on Docker
// actuallu replyimg). As any other init(), it is called somewhere during init phase
// so do not expect ALL inits from other tests (if any) to compete by now.
func init() {
	waitSec := flag.Int("t", defaultSleepIntervalSec, "Reconnect polling time, in seconds")
	logLevel := flag.String("log_level", "debug", "Logging Level")
	logFile := flag.String("log_file", config.DefaultLogPath, "Log file path")
	configFile := flag.String("config", config.DefaultConfigPath, "Configuration file path")

	flag.StringVar(&endPoint1, "H1", dockerUSocket, "Endpoint (Host1) to connect to")
	flag.StringVar(&endPoint2, "H2", dockerUSocket, "Endpoint (Host2) to connect to")
	flag.StringVar(&volumeName, "v", "TestVol", "Volume name to use in sanity tests")
	flag.BoolVar(&removeContainers, "rm", true, "rm container after run")
	flag.StringVar(&driverName, "d", "vmdk", "Driver name. We refcount volumes on this driver")
	flag.Parse()

	usingConfigFileDefaults := logInit(logLevel, logFile, configFile)

	reconnectSleepInterval = time.Duration(*waitSec) * time.Second
	defaultHeaders = map[string]string{"User-Agent": "engine-api-client-1.0"}

	log.WithFields(log.Fields{
		"driver":                   driverName,
		"waitSec":                  *waitSec,
		"log_level":                *logLevel,
		"log_file":                 *logFile,
		"conf_file":                *configFile,
		"using_conf_file_defaults": usingConfigFileDefaults,
	}).Info("VMDK plugin tests started ")
}

// returns in-container mount point for a volume
func getMountpoint(vol string) string {
	return defaultMountLocation + "/" + vol
}

// runs a command in a container , with volume mounted
// returns completion code.
// exits (t.Fatal() or create/start/wait errors
func runContainerCmd(t *testing.T, client *client.Client, volumeName string,
	image string, cmd *strslice.StrSlice, addr string) int {

	mountPoint := getMountpoint(volumeName)
	bind := volumeName + ":" + mountPoint
	t.Logf("Running cmd=%v with vol=%s on client %s", cmd, volumeName, addr)

	r, err := client.ContainerCreate(
		&container.Config{Image: image, Cmd: *cmd,
			Volumes: map[string]struct{}{mountPoint: {}}},
		&container.HostConfig{Binds: []string{bind}}, nil, "")
	if err != nil {
		t.Fatalf("\tContainer create failed: %v", err)
	}

	err = client.ContainerStart(r.ID)
	if err != nil {
		t.Fatalf("\tContainer start failed: id=%s, err %v", r.ID, err)
	}

	code, err := client.ContainerWait(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("\tContainer wait failed: id=%s, err %v", r.ID, err)
	}

	if removeContainers == false {
		t.Logf("\tSkipping container removal, id=%s (removeContainers == false)",
			r.ID)
		return code
	}

	err = client.ContainerRemove(types.ContainerRemoveOptions{
		ContainerID:   r.ID,
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		t.Fatalf("\nContainer removal failed: %v", err)
	}

	return code
}

// Checks that we can touch a file in one container and then stat it
// in another container, using the same (vmdk-based) volume
//
// goes over 'cases' and runs commands, then checks expected return code
func checkTouch(t *testing.T, c *client.Client, vol string,
	file string, addr string) {

	cases := []struct {
		image    string             // Container image to use
		cmd      *strslice.StrSlice // Command to run under busybox
		expected int                // expected results
	}{
		{"busybox", &strslice.StrSlice{"touch", getMountpoint(vol) + "/" + file}, 0},
		{"busybox", &strslice.StrSlice{"stat", getMountpoint(vol) + "/" + file}, 0},
	}

	for _, i := range cases {
		code := runContainerCmd(t, c, vol, i.image, i.cmd, addr)
		if code != i.expected {
			t.Errorf("Expected  %d, got %d (cmd: %v)", i.expected, code, i.cmd)
		}
	}
}

// returns nil for NOT_FOUND and  if volume exists
// still fails the test if driver for this volume is not vmdk
func volumeVmdkExists(t *testing.T, c *client.Client, vol string) *types.Volume {
	reply, err := c.VolumeList(filters.Args{})
	if err != nil {
		t.Fatalf("Failed to enumerate  volumes: %v", err)
	}

	for _, v := range reply.Volumes {
		//	t.Log(v.Name, v.Driver, v.Mountpoint)
		if v.Name == vol {
			return v
		}
	}
	return nil
}

// Sanity test for VMDK volumes
// - check we can attach/detach correct volume (we use 'touch' and 'stat' to validate
// - check volumes are correctly created and deleted.
// - check we see it properly from another docker VM (-H2 flag)
func TestSanity(t *testing.T) {

	fmt.Printf("Running tests on  %s (may take a while)...\n", endPoint1)
	clients := []struct {
		endPoint string
		client   *client.Client
	}{
		{endPoint1, new(client.Client)},
		{endPoint2, new(client.Client)},
	}

	for idx, elem := range clients {
		c, err := client.NewClient(elem.endPoint, apiVersion, nil, defaultHeaders)
		if err != nil {
			t.Fatalf("Failed to connect to %s, err: %v", elem.endPoint, err)
		}
		t.Logf("Successfully connected to %s", elem.endPoint)
		clients[idx].client = c
	}

	c := clients[0].client // this is the endpoint we use as master
	t.Logf("Creating vol=%s on client %s.", volumeName, clients[0].endPoint)
	_, err := c.VolumeCreate(
		types.VolumeCreateRequest{
			Name:   volumeName,
			Driver: driverName,
			DriverOpts: map[string]string{
				"size":   "1gb",
				"policy": "good",
			},
		})
	if err != nil {
		t.Fatal(err)
	}

	checkTouch(t, c, volumeName, "file_to_touch", clients[0].endPoint)

	for _, elem := range clients {
		v := volumeVmdkExists(t, elem.client, volumeName)
		if v == nil {
			t.Fatalf("Volume=%s is missing on %s after create",
				volumeName, elem.endPoint)
		}
		if v.Driver != driverName {
			t.Fatalf("wrong driver (%s) for volume %s", v.Driver, v.Name)
		}
	}

	err = c.VolumeRemove(volumeName)
	if err != nil {
		t.Fatalf("Failed to delete volume, err: %v", err)
	}

	for _, elem := range clients {
		if volumeVmdkExists(t, elem.client, volumeName) != nil {
			t.Errorf("Volume=%s is still present on %s after removal",
				volumeName, elem.endPoint)
		}
	}
}
