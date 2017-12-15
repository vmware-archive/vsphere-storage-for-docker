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

// This test suite includes additional test cases to verify basic functionality
// in most common configurations.

// +build runalways

package e2e

import (
	admincliconst "github.com/vmware/vsphere-storage-for-docker/tests/constants/admincli"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/admincli"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/dockercli"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/misc"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/verification"
	. "gopkg.in/check.v1"
)

// Test volume isolation between VMs backed by different datastores:
// Volume is created on the local datastore attached to the ESX where
// VM1 resides on. It's expected that this volume is visible to VM1,
// but invisible to VM2 which resides on a different ESX that has no
// access to this datastore.
//
// Test steps:
// 1. Create a volume from VM1
// 2. Verify the volume is available from VM1
// 3. Verify the volume is unavailable from VM2
// 4. Remove the volume
func (s *BasicTestSuite) TestBasicVolumeIsolation(c *C) {
	misc.LogTestStart(c.TestName())

	out, err := dockercli.CreateVolume(s.vm1, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	accessible := verification.CheckVolumeAvailability(s.vm1, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

	accessible = verification.CheckVolumeAvailability(s.vm2, s.volName1)
	//TODO: VM2 inaccessible to this volume is currently not available
	//c.Assert(accessible, Equals, false, Commentf("Volume %s is still available", s.volName1))

	out, err = dockercli.DeleteVolume(s.vm1, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// Test volume isolation between _DEFAULT and user defined vmgroups:
// Volume created in a VM belonging to _DEFAULT should be invisible
// to VMs inside a user defined vmgroup; and vice versa.
//
// Test steps:
// 1. Initialize Config DB
// 2. Create a volume from VM1
// 3. Verify the volume is visible to VM1 and VM2
// 4. Create a VmGroup T1, add VM1 to T1
// 5. Verify the volume is still visible to VM2, but invisible to VM1
// 6. Create the 2nd volume from VM1
// 7. Verify the 2nd volume is visible to VM1, but invisible to VM2
// 8. Remove the volumes
// 9. Remove Config DB
func (s *BasicTestSuite) TestVmGroupVolumeIsolation(c *C) {
	misc.LogTestStart(c.TestName())

	// Initialize Config DB
	out, err := admincli.ConfigInit(s.esx)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.CreateVolume(s.vm1, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	accessible := verification.CheckVolumeAvailability(s.vm1, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available on [%s]", s.volName1, s.vm1))

	accessible = verification.CheckVolumeAvailability(s.vm2, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available on [%s]", s.volName1, s.vm2))

	const vmgroup = "T1"
	out, err = admincli.CreateVMgroup(s.esx, vmgroup, s.vm1Name, admincliconst.VMHomeDatastore)
	c.Assert(err, IsNil, Commentf(out))

	accessible = verification.CheckVolumeAvailability(s.vm1, s.volName1)
	c.Assert(accessible, Equals, false, Commentf("Volume %s is still available on [%s]", s.volName1, s.vm1))

	accessible = verification.CheckVolumeAvailability(s.vm2, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available on [%s]", s.volName1, s.vm2))

	out, err = dockercli.CreateVolume(s.vm1, s.volName2)
	c.Assert(err, IsNil, Commentf(out))

	accessible = verification.CheckVolumeAvailability(s.vm1, s.volName2)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available on [%s]", s.volName2, s.vm1))

	accessible = verification.CheckVolumeAvailability(s.vm2, s.volName2)
	c.Assert(accessible, Equals, false, Commentf("Volume %s is available on [%s]", s.volName2, s.vm2))

	// Clean up the volumes
	out, err = dockercli.DeleteVolume(s.vm2, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.DeleteVolume(s.vm1, s.volName2)
	c.Assert(err, IsNil, Commentf(out))

	// Clean up the vm group
	out, err = admincli.RemoveVMFromVMgroup(s.esx, vmgroup, s.vm1Name)
	c.Assert(err, IsNil, Commentf(out))

	out, err = admincli.DeleteVMgroup(s.esx, vmgroup, false)
	c.Assert(err, IsNil, Commentf(out))

	// Remove Config DB
	out, err = admincli.ConfigRemove(s.esx)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}
