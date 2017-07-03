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
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/strslice"
	testutil "github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/log_formatter"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/refcount"
	"golang.org/x/net/context"
)

const (
	defaultMountLocation = "/mnt/testvol"
	// tests are often run under regular account and have no access to /var/log
	defaultTestLogPath = "/tmp/test-docker-volume-vsphere.log"
	// Number of volumes per client for parallel tests
)

var (
	// flag vars - see init() for help
	endPoint1        string
	endPoint2        string
	volumeName       string
	driverName       string
	defaultHeaders   map[string]string
	removeContainers bool
	parallelVolumes  int
	parallelClones   int
)

type testClient struct {
	endPoint string
	client   *client.Client
}

// prepares the environment. Kind of "main"-ish code for tests.
// Parses flags and inits logs and mount ref counters (the latter waits on Docker
// actually replying). As any other init(), it is called somewhere during init phase
// so do not expect ALL inits from other tests (if any) to compete by now.
func init() {
	logLevel := flag.String("log_level", "debug", "Logging Level")
	logFile := flag.String("log_file", config.DefaultLogPath, "Log file path")
	configFile := flag.String("config", config.DefaultConfigPath, "Configuration file path")

	flag.BoolVar(&removeContainers, "rm", true, "rm container after run")
	flag.StringVar(&driverName, "d", "vsphere:latest", "Driver name. We refcount volumes on this driver")
	flag.IntVar(&parallelVolumes, "parallel_volumes", 3, "Volumes per docker daemon for create/delete concurrent tests")
	flag.IntVar(&parallelClones, "parallel_clones", 2, "Volumes per docker daemon for clone concurrent tests")
	flag.Parse()
	usingConfigFileDefaults := logInit(logLevel, logFile, configFile)

	defaultHeaders = map[string]string{"User-Agent": "engine-api-client-1.0"}

	log.WithFields(log.Fields{
		"driver":                   driverName,
		"log_level":                *logLevel,
		"log_file":                 *logFile,
		"conf_file":                *configFile,
		"using_conf_file_defaults": usingConfigFileDefaults,
	}).Info("VMDK plugin tests started ")
	log.SetFormatter(new(log_formatter.VmwareFormatter))
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

	r, err := client.ContainerCreate(context.Background(),
		&container.Config{Image: image, Cmd: *cmd,
			Volumes: map[string]struct{}{mountPoint: {}}},
		&container.HostConfig{Binds: []string{bind}}, nil, "")
	if err != nil {
		t.Fatalf("\tContainer create failed: %v", err)
	}

	err = client.ContainerStart(context.Background(), r.ID,
		types.ContainerStartOptions{})
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

	err = client.ContainerRemove(context.Background(), r.ID,
		types.ContainerRemoveOptions{
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
	reply, err := c.VolumeList(context.Background(), filters.Args{})
	if err != nil {
		t.Fatalf("Failed to enumerate  volumes: %v", err)
	}

	for _, v := range reply.Volumes {
		//	t.Log(v.Name, v.Driver, v.Mountpoint)
		// "docker volume ls" return volume with format vol_name@datastore
		// variable "vol" is specified with short notation (just the volume name,
		// not including the @datastore part)
		if strings.Contains(v.Name, vol) {
			return v
		}
	}
	return nil
}

// Initialize the clients and connect with the docker daemons
func getClients(t *testing.T) []testClient {

	clients := []testClient{
		{testutil.GetEndPoint1(), new(client.Client)},
		{testutil.GetEndPoint2(), new(client.Client)},
	}

	for idx, elem := range clients {
		c, err := client.NewClient(elem.endPoint, refcount.ApiVersion, nil, defaultHeaders)
		if err != nil {
			t.Fatalf("Failed to connect to %s, err: %v", elem.endPoint, err)
		}
		t.Logf("Successfully connected to %s", elem.endPoint)
		clients[idx].client = c
	}
	return clients
}

// Sanity test for VMDK volumes
// - check we can attach/detach correct volume (we use 'touch' and 'stat' to validate
// - check volumes are correctly created and deleted.
// - check we see it properly from another docker VM (-H2 flag)
func TestSanity(t *testing.T) {

	fmt.Printf("%s START: Running TestSanity on  %s (may take a while)...\n",
		time.Now().Format(time.RFC3339), testutil.GetEndPoint1())

	clients := getClients(t)
	c := clients[0].client // this is the endpoint we use as master
	volumeName := testutil.GetVolumeName()
	t.Logf("Creating vol=%s on client %s.", volumeName, clients[0].endPoint)
	_, err := c.VolumeCreate(context.Background(),
		types.VolumeCreateRequest{
			Name:   volumeName,
			Driver: driverName,
			DriverOpts: map[string]string{
				"size": "200MB",
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

	err = c.VolumeRemove(context.Background(), volumeName)
	if err != nil {
		t.Fatalf("Failed to delete volume, err: %v", err)
	}

	for _, elem := range clients {
		if volumeVmdkExists(t, elem.client, volumeName) != nil {
			t.Errorf("Volume=%s is still present on %s after removal",
				volumeName, elem.endPoint)
		}
	}
	fmt.Printf("%s END: Running TestSanity on  %s (may take a while)...\n",
		time.Now().Format(time.RFC3339), testutil.GetEndPoint1())

}

// Test concurrent volume operations
// - concurrent create/delete between different docker hosts
// - concurrent create/delete on the same docker host
// - concurrent clone/delete between different docker hosts
func TestConcurrency(t *testing.T) {

	clients := getClients(t)
	volumeName := "volTestP"

	fmt.Printf("%s Running concurrent tests on %s and %s (may take a while)...\n",
		time.Now().Format(time.RFC3339), clients[0].endPoint, clients[1].endPoint)

	// Buffered channel to read results from
	results := make(chan error, parallelVolumes)

	createRequest := types.VolumeCreateRequest{
		Name:   volumeName,
		Driver: driverName,
		DriverOpts: map[string]string{
			"size": "200mb",
		},
	}

	// Only run this if testing against different clients
	if clients[0].endPoint != clients[1].endPoint {
		fmt.Printf("%s START: Running create/delete multi-host concurrent test ...\n",
			time.Now().Format(time.RFC3339))
		// Create/delete goroutine
		for idx, elem := range clients {
			go func(idx int, c *client.Client) {
				for i := 0; i < parallelVolumes; i++ {
					volName := fmt.Sprintf("%s%d%d", volumeName, idx, i)
					createRequest.Name = volName
					_, err := c.VolumeCreate(context.Background(), createRequest)
					results <- err
					err = c.VolumeRemove(context.Background(), volName)
					results <- err
				}
			}(idx, elem.client)
		}
		// Read the results from the channel
		for i := 0; i < len(clients)*parallelVolumes*2; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Create/delete concurrent test failed, err: %v", err)
			}
		}
	} else {
		fmt.Printf("Skipping create/delete concurrent test, same docker host. Will be tested next.\n")
	}
	fmt.Printf("%s END: Running create/delete multi-host concurrent test ...\n",
		time.Now().Format(time.RFC3339))

	fmt.Printf("Running same docker host concurrent create/delete test on %s...\n", clients[0].endPoint)
	parallelVolumes1 := parallelVolumes / 2
	for idx := 0; idx < 3; idx++ {
		go func(idx int, c *client.Client) {
			for i := 0; i < parallelVolumes1; i++ {
				volName := fmt.Sprintf("%s-same%d%d", volumeName, idx, i)
				createRequest.Name = volName
				_, err := c.VolumeCreate(context.Background(), createRequest)
				results <- err
				err = c.VolumeRemove(context.Background(), volName)
				results <- err
			}
		}(idx, clients[0].client)
	}
	// Read the results from the channel
	for i := 0; i < 3*parallelVolumes1*2; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Same docker host concurrent create/delete test failed, err: %v", err)
		}
	}

	fmt.Printf("%s START: Running clone concurrent test...\n",
		time.Now().Format(time.RFC3339))
	masterVolName := volumeName + "Clone"

	// Create master volume for cloning
	createRequest.Name = masterVolName
	createRequest.DriverOpts["size"] = "100mb"

	_, err := clients[0].client.VolumeCreate(context.Background(), createRequest)
	if err != nil {
		t.Fatal(err)
	}

	// Remove size option for clone
	delete(createRequest.DriverOpts, "size")

	// Change diskformat for the 1st clones to produce waits
	createRequest.DriverOpts["diskformat"] = "eagerzeroedthick"

	// Clone goroutine
	for idx, elem := range clients {
		go func(idx int, c *client.Client) {
			for i := 0; i < parallelClones; i++ {
				volName := fmt.Sprintf("%s-clone%d%d", masterVolName, idx, i)
				createRequest.Name = volName
				createRequest.DriverOpts["clone-from"] = masterVolName

				// After 1st clone use thin diskformat
				if i > 0 {
					createRequest.DriverOpts["diskformat"] = "thin"
				}
				_, err := c.VolumeCreate(context.Background(), createRequest)
				results <- err
				err = c.VolumeRemove(context.Background(), volName)
				results <- err
			}
		}(idx, elem.client)
	}

	// Read the results from the channel
	for i := 0; i < len(clients)*parallelClones*2; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Running clone concurrent test failed, err: %v", err)
		}
	}

	// Remove the master volume
	clients[0].client.VolumeRemove(context.Background(), masterVolName)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s END: Running clone concurrent test...\n",
		time.Now().Format(time.RFC3339))

}
