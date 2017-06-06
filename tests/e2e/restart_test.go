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

// This test suite includes test cases involving restart docker and plugin. #1252
// End goal is to make sure volumes get detached smoothly in different scenarios

// +build runonce

package e2e

import (
	"log"
	"os"

	. "gopkg.in/check.v1"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

type PluginSuite struct {
	volumeName    string
	hostIP        string
	containerName string
	esxIP         string
}

func (s *PluginSuite) SetUpTest(c *C) {
	s.hostIP = os.Getenv("VM2")
	s.esxIP = os.Getenv("ESX")
	s.volumeName = inputparams.GetVolumeNameWithTimeStamp("restart_test")
	s.containerName = inputparams.GetContainerNameWithTimeStamp("restart_test")

	// Create a volume
	out, err := dockercli.CreateVolume(s.hostIP, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

func (s *PluginSuite) TearDownTest(c *C) {
	out, err := dockercli.RemoveContainer(s.hostIP, s.containerName)
	c.Assert(err, IsNil, Commentf(out))
	out, err = dockercli.DeleteVolume(s.hostIP, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

var _ = Suite(&PluginSuite{})

// Test Stale mount
// 1. Attach a volume to a container
// 2. Verify attached status
// 3. Restart docker
// 4. Verify detached status. Volume should be detached (within the timeout)
func (s *PluginSuite) TestVolumeStaleMount(c *C) {
	log.Printf("START: restart_test.TestVolumeStaleMount")

	out, err := dockercli.AttachVolume(s.hostIP, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	status := verification.VerifyAttachedStatus(s.volumeName, s.hostIP, s.esxIP)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	out, err = dockercli.KillDocker(s.hostIP)
	c.Assert(err, IsNil, Commentf(out))

	status = verification.VerifyDetachedStatus(s.volumeName, s.hostIP, s.esxIP)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	log.Printf("END: restart_test.TestVolumeStaleMount")
}

// Test vDVS plugin restart
// 1. Attach a volume with restart=always flag
// 2. Verify volume attached status
// 3. Kill vDVS plugin
// 4. Stop the container
// 5. Start the same container
// 6. Stop the container
// 7. Verify volume detached status
func (s *PluginSuite) TestPluginKill(c *C) {
	log.Printf("START: restart_test.TestPluginKill")

	out, err := dockercli.AttachVolumeWithRestart(s.hostIP, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	status := verification.VerifyAttachedStatus(s.volumeName, s.hostIP, s.esxIP)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	out, err = dockercli.KillVDVSPlugin(s.hostIP)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.StopContainer(s.hostIP, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.StartContainer(s.hostIP, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.StopContainer(s.hostIP, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	status = verification.VerifyDetachedStatus(s.volumeName, s.hostIP, s.esxIP)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	log.Printf("END: restart_test.TestPluginKill")
}
