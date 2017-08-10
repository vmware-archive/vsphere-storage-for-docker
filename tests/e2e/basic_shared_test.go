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

// This test suite includes test cases to verify basic functionality
// before upgrade for upgrade test

// +build runonceshared

package e2e

import (
	"log"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

type BasicSharedTestSuite struct {
	config        *inputparams.TestConfig
	esx           string
	vm1           string
	vm2           string
	vm1Name       string
	vm2Name       string
	volName1      string
	volName2      string
	containerName string
}

func (s *BasicSharedTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping basic sharedtests")
	}

	s.esx = s.config.EsxHost
	s.vm1 = s.config.DockerHosts[0]
	s.vm1Name = s.config.DockerHostNames[0]
	if len(s.config.DockerHosts) == 2 {
		s.vm2 = s.config.DockerHosts[1]
		s.vm2Name = s.config.DockerHostNames[1]
	}
}

func (s *BasicSharedTestSuite) SetUpTest(c *C) {
	s.volName1 = inputparams.GetUniqueVolumeName(c.TestName())
	s.volName2 = inputparams.GetUniqueVolumeName(c.TestName())
	s.containerName = inputparams.GetUniqueContainerName(c.TestName())
}

var _ = Suite(&BasicSharedTestSuite{})

// All VMs are created in a shared datastore
// Test steps:
// 1. Create a volume
// 2. Verify the volume is available
// 3. Attach the volume
// 4. Verify volume status is attached
// 5. Remove the volume (expect fail)
// 6. Remove the container
// 7. Verify volume status is detached
// 8. Remove the volume
// 9. Verify the volume is unavailable
// TODO: step 3-7 currently is not available since volume mount/unmount is not available yet
func (s *BasicSharedTestSuite) TestVolumeLifecycle(c *C) {
	misc.LogTestStart(c.TestName())

	for _, host := range s.config.DockerHosts {
		out, err := dockercli.CreateSharedVolume(host, s.volName1)
		c.Assert(err, IsNil, Commentf(out))

		accessible := verification.CheckVolumeAvailability(host, s.volName1)
		c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

		// out, err = dockercli.AttachVolume(host, s.volName1, s.containerName)
		// c.Assert(err, IsNil, Commentf(out))

		// status := verification.VerifyAttachedStatus(s.volName1, host, s.esx)
		// c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volName1))

		// out, err = dockercli.DeleteVolume(host, s.volName1)
		// c.Assert(err, Not(IsNil), Commentf(out))

		// out, err = dockercli.RemoveContainer(host, s.containerName)
		// c.Assert(err, IsNil, Commentf(out))

		// status = verification.VerifyDetachedStatus(s.volName1, host, s.esx)
		// c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volName1))
		out = verification.GetSharedVolumeStatusHost(s.volName1, host)
		log.Println("GetSharedVolumeStatusHost return out[%s] for volume %s", out, s.volName1)
		c.Assert(out, Equals, "Ready", Commentf("Volume %s status is expected to be [Ready], actual status is [%s]",
			s.volName1, out))

		accessible = verification.CheckVolumeAvailability(host, s.volName1)
		c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

		// delete the volume from master until issue 1715 is fixed
		out, err = dockercli.DeleteVolume(s.config.DockerHosts[0], s.volName1)
		c.Assert(err, IsNil, Commentf(out))
	}

	misc.LogTestEnd(c.TestName())
}
