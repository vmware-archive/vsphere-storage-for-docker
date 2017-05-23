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

package e2e

import (
	"log"
	"os"

	. "gopkg.in/check.v1"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

type PluginSuite struct {
	volumeName    string
	hostName      string
	containerName string
	esxName       string
}

func (s *PluginSuite) SetUpTest(c *C) {
	s.hostName = os.Getenv("VM2")
	s.esxName = os.Getenv("ESX")
	s.volumeName = inputparams.GetVolumeNameWithTimeStamp("restart_test")
	s.containerName = inputparams.GetContainerNameWithTimeStamp("restart_test")
}

func (s *PluginSuite) TearDownTest(c *C) {
	out, err := dockercli.RemoveContainer(s.hostName, s.containerName)
	c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))
	out, err = dockercli.DeleteVolume(s.hostName, s.volumeName)
	c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))
}

var _ = Suite(&PluginSuite{})

// Test Stale mount
// 1. Create a volume
// 2. Attach it to a container
// 3. Restart docker
// 4. Volume should be detached (within the timeout)
func (s *PluginSuite) TestVolumeStaleMount(c *C) {
	log.Printf("START: Stale mount test")

	out, err := dockercli.CreateVolume(s.hostName, s.volumeName)
	c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

	out, err = dockercli.AttachVolume(s.hostName, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

	status := verification.VerifyAttachedStatus(s.volumeName, s.hostName, s.esxName)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	out, err = dockercli.KillDocker(s.hostName)
	c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

	status = verification.VerifyDetachedStatus(s.volumeName, s.hostName, s.esxName)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	log.Printf("END: Stale mount test")
}
