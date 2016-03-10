// basic sanity test

package main

import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"testing"
)

const (
	endpoint   = "unix:///var/run/docker.sock"
	apiVersion = "v1.22"
)

// Connects to docker via unix socket and runs basic tests.

func TestSanity(t *testing.T) {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}

	fmt.Println("Connecting to ", endpoint)
	cli, err := client.NewClient(endpoint, apiVersion, nil, defaultHeaders)
	if err != nil {
		t.Fatal("Failed to connect to the client at ", endpoint, " , err: ", err)
	}

	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(options)
	if err != nil {
		t.Fatal("Failed to enumerate options, err: ", err)
	}
	fmt.Println("Containers count: ", len(containers))

	reply, err := cli.VolumeList(filters.Args{})
	if err != nil {
		t.Fatal("Failed to enumerate  volumes")
	}
	fmt.Println("Volumes count:", len(reply.Volumes))
	//	for _, v := range reply.Volumes {
	//		fmt.Println(v.Name, v.Driver, v.Mountpoint)
	//	}
}
