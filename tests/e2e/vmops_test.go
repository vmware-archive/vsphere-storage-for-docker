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

// This test suite contains tests to verify VM operations (snapshot) with
// vSphere docker volumes attached to the VM.

// +build runonce

package e2e

import (
	"log"

	adminutils "github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/esx"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

// VMOpsTest - struct for vmgroup tests
type VMOpsTest struct {
	config        *inputparams.TestConfig
	testContainer string
	volName1      string
	volName2      string
}

var _ = Suite(&VMOpsTest{})

func (vm *VMOpsTest) SetUpSuite(c *C) {
	vm.config = inputparams.GetTestConfig()
	if vm.config == nil {
		c.Skip("Unable to retrieve test config, skipping vmgroup tests")
	}
	adminutils.ConfigInit(vm.config.EsxHost)

	vm.testContainer = inputparams.GetUniqueContainerName("Test_VMOps")

	vm.volName1 = inputparams.GetUniqueVolumeName("Test_VMOps")
	vm.volName2 = inputparams.GetUniqueVolumeName("Test_VMOps")

	// Create the test volumes
	_, err := dockercli.CreateVolumeWithOptions(vm.config.DockerHosts[0], vm.volName1, "-o attach-as=persistent")
	c.Assert(err, IsNil, Commentf("Failed creating volume %s\n", vm.volName1))

	_, err = dockercli.CreateVolume(vm.config.DockerHosts[0], vm.volName2)
	c.Assert(err, IsNil, Commentf("Failed creating volume %s\n", vm.volName2))

	log.Printf("Done creating vmops test config.")
}

func (vm *VMOpsTest) TearDownSuite(c *C) {
	dockercli.DeleteVolume(vm.config.DockerHosts[0], vm.volName1)
	dockercli.DeleteVolume(vm.config.DockerHosts[0], vm.volName2)

	// Remove Config DB
	adminutils.ConfigRemove(vm.config.EsxHost)
	log.Printf("Done cleanup of vmops test config.")
}

// Tests to validate VM snapshots with vSphere volumes attached.

// TestVMSnapWithPersistentDisk - Snapshot a VM with a persistent
// disk attached.
// 1. Run container with a volume attached as persistent.
// 2. Snapshot the VM, should succeed
// 3. Revert snapshot and verify volume is still attached
// 4. Stop container and verify volume is detached
func (vm *VMOpsTest) TestVMSnapWithPersistentDisk(c *C) {
	misc.LogTestStart(c.TestName())
	// 1. Start container with volume attached as persistent
	_, err := dockercli.AttachVolume(vm.config.DockerHosts[0], vm.volName1, vm.testContainer)
	c.Assert(err, IsNil, Commentf("Failed running container %s with volume %s attached as persistent\n", vm.testContainer, vm.volName1))

	status := verification.VerifyAttachedStatus(vm.volName1, vm.config.DockerHosts[0], vm.config.EsxHost, vm.config.DockerHostNames[0])
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", vm.volName1))

	// 2. Snapshot VM and verify attached state of volume
	out, err := esx.TakeSnapshot(vm.config.DockerHostNames[0], "snap1")
	c.Assert(err, IsNil, Commentf(out))

	// 3. Remove snapshot and verify volume is still attached
	out, err = esx.RemoveSnapshot(vm.config.DockerHostNames[0], "snap1")
	c.Assert(err, IsNil, Commentf(out))

	status = verification.VerifyAttachedStatus(vm.volName1, vm.config.DockerHosts[0], vm.config.EsxHost, vm.config.DockerHostNames[0])
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", vm.volName1))

	// 4. Stop container and verify volume is detached
	_, err = dockercli.RemoveContainer(vm.config.DockerHosts[0], vm.testContainer)
	c.Assert(err, IsNil, Commentf("Failed to remove container %s with volume %s attached as persistent\n", vm.testContainer, vm.volName1))

	status = verification.VerifyDetachedStatus(vm.volName1, vm.config.DockerHosts[0], vm.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", vm.volName1))

	misc.LogTestEnd(c.TestName())
}

// TestVMSnapWithIndependentDisk - Snapshot a VM with a independent
// disk attached.
// 1. Run a container with a volume attached as independent
// 2. Take snapshot of the vm operation should fail as vm has independent disk attached
// 3. Stop the container
// 4. Again try taking the snapshot of vm - operation must succeed
func (vm *VMOpsTest) TestVMSnapWithIndependentDisk(c *C) {
	misc.LogTestStart(c.TestName())
	// 1. Start container with volume attached as independent-persistent
	_, err := dockercli.AttachVolume(vm.config.DockerHosts[0], vm.volName2, vm.testContainer)
	c.Assert(err, IsNil, Commentf("Failed running container %s with volume %s attached as independent disk\n", vm.testContainer, vm.volName2))

	// 2. Snapshot VM (should fail) and verify attached state of volume
	out, err := esx.TakeSnapshot(vm.config.DockerHostNames[0], "snap1")
	c.Assert(err, Not(IsNil), Commentf(out))

	status := verification.VerifyAttachedStatus(vm.volName2, vm.config.DockerHosts[0], vm.config.EsxHost, vm.config.DockerHostNames[0])
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", vm.volName2))

	// 3. Stop container and verify volume is detached
	_, err = dockercli.RemoveContainer(vm.config.DockerHosts[0], vm.testContainer)
	c.Assert(err, IsNil, Commentf("Failed to remove container %s with volume %s attached as independent disk\n", vm.testContainer, vm.volName2))

	status = verification.VerifyDetachedStatus(vm.volName2, vm.config.DockerHosts[0], vm.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", vm.volName1))

	// 4. Snapshot VM, should succeed
	out, err = esx.TakeSnapshot(vm.config.DockerHostNames[0], "snap1")
	c.Assert(err, IsNil, Commentf(out))

	// 5. Remove snapshot
	out, err = esx.RemoveSnapshot(vm.config.DockerHostNames[0], "snap1")
	c.Assert(err, IsNil, Commentf(out))
	misc.LogTestEnd(c.TestName())
}

