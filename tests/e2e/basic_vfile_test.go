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

// +build runoncevfile

package e2e

import (
	"log"
	"strconv"
	"time"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

type BasicVFileTestSuite struct {
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

func (s *BasicVFileTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping basic vfile tests")
	}

	s.esx = s.config.EsxHost
	s.vm1 = s.config.DockerHosts[0]
	s.vm1Name = s.config.DockerHostNames[0]
	if len(s.config.DockerHosts) == 2 {
		s.vm2 = s.config.DockerHosts[1]
		s.vm2Name = s.config.DockerHostNames[1]
	}
}

func (s *BasicVFileTestSuite) SetUpTest(c *C) {
	s.volName1 = inputparams.GetVFileVolumeName()
	s.volName2 = inputparams.GetVFileVolumeName()
	s.containerName = inputparams.GetUniqueContainerName(c.TestName())
}

var _ = Suite(&BasicVFileTestSuite{})

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
func (s *BasicVFileTestSuite) TestVolumeLifecycle(c *C) {
	misc.LogTestStart(c.TestName())

	for _, host := range s.config.DockerHosts {
		out, err := dockercli.CreateVFileVolume(host, s.volName1)
		c.Assert(err, IsNil, Commentf(out))

		accessible := verification.CheckVolumeAvailability(host, s.volName1)
		c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

		out, err = dockercli.AttachVFileVolume(host, s.volName1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))

		// Expect global refcount for this volume to be 1
		out = verification.GetVFileVolumeGlobalRefcount(s.volName1, host)
		grefc, _ := strconv.Atoi(out)
		c.Assert(grefc, Equals, 1, Commentf("Expected volume global refcount to be 1, found %s", out))

		out, err = dockercli.DeleteVolume(host, s.volName1)
		c.Assert(err, Not(IsNil), Commentf(out))

		out, err = dockercli.RemoveContainer(host, s.containerName)
		c.Assert(err, IsNil, Commentf(out))

		// Expect global refcount for this volume to be 0
		out = verification.GetVFileVolumeGlobalRefcount(s.volName1, host)
		grefc, _ = strconv.Atoi(out)
		c.Assert(grefc, Equals, 0, Commentf("Expected volume global refcount to be 0, found %s", out))

		time.Sleep(5 * time.Second) // wait for volume status back to Ready

		out = verification.GetVFileVolumeStatusHost(s.volName1, host)
		log.Println("GetVFileVolumeStatusHost return out[%s] for volume %s", out, s.volName1)
		c.Assert(out, Equals, "Ready", Commentf("Volume %s status is expected to be [Ready], actual status is [%s]",
			s.volName1, out))

		accessible = verification.CheckVolumeAvailability(host, s.volName1)
		c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

		out, err = dockercli.DeleteVolume(host, s.volName1)
		c.Assert(err, IsNil, Commentf(out))
	}

	misc.LogTestEnd(c.TestName())
}
