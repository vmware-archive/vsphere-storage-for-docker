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
// in most common configurations

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

type BasicTestSuite struct {
	esxName       string
	volumeName    string
	containerName string
}

func (s *BasicTestSuite) SetUpTest(c *C) {
	s.esxName = os.Getenv("ESX")
	s.volumeName = inputparams.GetVolumeNameWithTimeStamp("basic_test")
	s.containerName = inputparams.GetContainerNameWithTimeStamp("basic_test")
}

var _ = Suite(&BasicTestSuite{})

// Test volume lifecycle management on different datastores:
// VM1 - local VMFS datastore
// VM2 - shared VMFS datastore
// VM3 - shared VSAN datastore
//
// Test steps:
// 1. Create a volume
// 2. Verify the volume is available
// 3. Attach the volume
// 4. Verify volume status is attached
// 5. Remove the container
// 6. Verify volume status is detached
// 7. Remove the volume
// 8. Verify the volume is unavailable
func (s *BasicTestSuite) TestVolumeLifecycle(c *C) {
	log.Printf("START: basic-test.TestVolumeLifecycle")

	dockerHosts := []string{os.Getenv("VM1"), os.Getenv("VM2"), os.Getenv("VM3")}
	for _, host := range dockerHosts {
		//TODO: Remove this check once VM3 is available from the CI testbed
		if host == "" {
			continue
		}

		out, err := dockercli.CreateVolume(host, s.volumeName)
		c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

		accessible := verification.CheckVolumeAvailability(host, s.volumeName)
		c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volumeName))

		out, err = dockercli.AttachVolume(host, s.volumeName, s.containerName)
		c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

		status := verification.VerifyAttachedStatus(s.volumeName, host, s.esxName)
		c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

		out, err = dockercli.RemoveContainer(host, s.containerName)
		c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

		status = verification.VerifyDetachedStatus(s.volumeName, host, s.esxName)
		c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

		out, err = dockercli.DeleteVolume(host, s.volumeName)
		c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

		accessible = verification.CheckVolumeAvailability(host, s.volumeName)
		c.Assert(accessible, Equals, false, Commentf("Volume %s is still available", s.volumeName))
	}

	log.Printf("END: basic-test.TestVolumeLifecycle")
}

// Test volume isolation: Volume is created on the local datastore attached to
// the ESX where VM1 resides on. It's expected that this volume is visible to
// VM1, but invisible to VM2 which resides on a different ESX that has no access
// to this datastore.
//
// Test steps:
// 1. Create a volume from VM1
// 2. Verify the volume is available from VM1
// 3. Verify the volume is unavailable from VM2
// 4. Remove the volume
func (s *BasicTestSuite) TestVolumeIsolation(c *C) {
	log.Printf("START: basic-test.TestVolumeIsolation")

	vm1, vm2 := os.Getenv("VM1"), os.Getenv("VM2")

	out, err := dockercli.CreateVolume(vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

	accessible := verification.CheckVolumeAvailability(vm1, s.volumeName)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volumeName))

	accessible = verification.CheckVolumeAvailability(vm2, s.volumeName)
	//TODO: VM2 inaccessible to this volume is currently not available
	//c.Assert(accessible, Equals, false, Commentf("Volume %s is still available", s.volumeName))

	out, err = dockercli.DeleteVolume(vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(misc.FormatOutput(out)))

	log.Printf("END: basic-test.TestVolumeIsolation")
}
