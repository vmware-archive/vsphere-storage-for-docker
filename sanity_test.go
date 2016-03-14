//
// VMDK Docker driver sanity tests.
//

package main

import (
	"flag"
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/strslice"
	"golang.org/x/net/context"
	"testing"
)

const (
	apiVersion    = "v1.22"
	driverName    = "vmdk"
	dockerUSocket = "unix:///var/run/docker.sock"
)

var (
	// flag vars - see init() for help
	endPoint1        string
	endPoint2        string
	volumeName       string
	removeContainers bool

	defaultHeaders map[string]string
)

func init() {
	flag.StringVar(&endPoint1, "H1", dockerUSocket, "Endpoint (Host1) to connect to")
	flag.StringVar(&endPoint1, "H", dockerUSocket, "Endpoint (Host) to connect to")
	flag.StringVar(&endPoint2, "H2", dockerUSocket, "Endpoint (Host2) to connect to")
	flag.StringVar(&volumeName, "v", "TestVol", "Volume name to use in sanity tests")
	flag.BoolVar(&removeContainers, "rm", true, "rm container after run")
	flag.Parse()

	defaultHeaders = map[string]string{"User-Agent": "engine-api-client-1.0"}
}

// returns in-container mount point for a volume
func getMountpoint(vol string) string {
	return "/mnt/vol/" + vol
}

// runs a command in a container , with volume mounted
// returns completion code.
// exits (t.Fatal() or create/start/wait errors
func runBusyboxCmd(t *testing.T, client *client.Client, volumeName string,
	cmd *strslice.StrSlice) int {

	mountPoint := getMountpoint(volumeName)
	bind := volumeName + ":" + mountPoint
	t.Logf("Running container vol=%s cmd=%v", volumeName, cmd)

	r, err := client.ContainerCreate(
		&container.Config{Image: "busybox", Cmd: *cmd,
			Volumes: map[string]struct{}{mountPoint: {}}},
		&container.HostConfig{Binds: []string{bind}}, nil, "")
	if err != nil {
		t.Fatalf("\tCreate failed: %v", err)
	}

	err = client.ContainerStart(r.ID)
	if err != nil {
		t.Fatalf("\tStart failed: %v", err)
	}

	code, err := client.ContainerWait(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("\tWait failed: %v", err)
	}

	if removeContainers == false {
		t.Logf("\tSkipping container rm (removeContainers == false)")
		return code
	}

	err = client.ContainerRemove(types.ContainerRemoveOptions{
		ContainerID:   r.ID,
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		t.Fatalf("\nRm failed: %v", err)
	}

	return code
}

// checks that we can touch and then stat a file
// goes over 'cases' and runs commands, then checks expected return code
func checkTouch(t *testing.T, c *client.Client, vol string,
	file string) {

	cases := []struct {
		cmd      *strslice.StrSlice // Command to run under busybox
		expected int                // expected results
	}{
		{&strslice.StrSlice{"touch", getMountpoint(vol) + "/" + file}, 0},
		{&strslice.StrSlice{"stat", getMountpoint(vol) + "/" + file}, 0},
	}

	for _, i := range cases {
		code := runBusyboxCmd(t, c, vol, i.cmd)
		if code != i.expected {
			t.Errorf("Expected  %d, got %d (cmd: %v)", i.expected, code, i.cmd)
		}
	}
}

// returns true if volume exists
// still fails the test if driver for this volume is not vmdk
func volumeVmdkExists(t *testing.T, c *client.Client, vol string) bool {
	reply, err := c.VolumeList(filters.Args{})
	if err != nil {
		t.Fatalf("Failed to enumerate  volumes: %v", err)
	}

	// TBD: check we do not have volumeName there
	for _, v := range reply.Volumes {
		//	t.Log(v.Name, v.Driver, v.Mountpoint)
		if v.Name == vol {
			if v.Driver != driverName {
				t.Errorf("wrong driver (%s) for volume %s", v.Driver, v.Name)
			}
			return true
		}
	}
	return false
}

// Sanity test for VMDK volumes
// - check we can attach/detach correct volume (we use 'touch' and 'stat' to validate
// - check volumes are correctly created and deleted.
// - check we see it properly from another docker VM (-H2 flag)
func TestSanity(t *testing.T) {

	fmt.Printf("Running tests on  %s (may take a while)...", endPoint1)
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
			t.Fatalf("Failed to connect to %s, err: %v", endPoint1, err)
		}
		clients[idx].client = c
	}

	c := clients[0].client // this is the endpoint we use as master
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

	checkTouch(t, c, volumeName, "file_to_touch")

	for _, elem := range clients {
		if volumeVmdkExists(t, elem.client, volumeName) == false {
			t.Errorf("Volume=%s is missing on %s after create",
				volumeName, elem.endPoint)
		}
	}

	err = c.VolumeRemove(volumeName)
	if err != nil {
		t.Fatalf("Failed to delete volume, err: %v", err)
	}

	for _, elem := range clients {
		if volumeVmdkExists(t, elem.client, volumeName) == true {
			t.Errorf("Volume=%s is still present on %s after removal",
				volumeName, elem.endPoint)
		}
	}
}
