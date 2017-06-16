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

// This test suite holds testcases related with vm listener functionality.
// In case of vm/esx host restart vDVS cleans up stale attachment details
// the volume.

// +build runonce

package e2e

import (
	"log"

	. "gopkg.in/check.v1"

	"github.com/vmware/docker-volume-vsphere/tests/constants/properties"
	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	esxutil "github.com/vmware/docker-volume-vsphere/tests/utils/esx"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

type VMListenerTestParams struct {
	config        *inputparams.TestConfig
	esx           string
	vm1           string
	vm2           string
	vm1Name       string
	vm2Name       string
	volumeName    string
	containerName string
}

func (s *VMListenerTestParams) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping basic tests")
	}

	s.esx = s.config.EsxHost
	s.vm1 = s.config.DockerHosts[0]
	s.vm2 = s.config.DockerHosts[1]
	s.vm1Name = s.config.DockerHostNames[0]
	s.vm2Name = s.config.DockerHostNames[1]
}

func (s *VMListenerTestParams) SetUpTest(c *C) {
	s.volumeName = inputparams.GetUniqueVolumeName("vmlistener_test")
	s.containerName = inputparams.GetUniqueContainerName("vmlistener_test")

	out, err := dockercli.CreateVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.AttachVolume(s.vm1, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))
}

func (s *VMListenerTestParams) TearDownTest(c *C) {
	// After killing a VM, the container may or may not be wiped off depending
	// on OS type: different Linux distribution has different behavior. So need
	// to check if the container exists or not.
	if dockercli.IsContainerExist(s.vm1, s.containerName) {
		out, err := dockercli.RemoveContainer(s.vm1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}
	out, err := dockercli.DeleteVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

var _ = Suite(&VMListenerTestParams{})

// Test vmdkops service restart
// 1. Setup: Create a volume
// 2. Setup: Attach it to a container
// 3. Restart the vmdkops service
// 4. Verification: volume stays as attached
func (s *VMListenerTestParams) TestServiceRestart(c *C) {
	misc.LogTestStart(c.TestName())

	// Restart vmdkops service
	out, err := admincli.RestartVmdkopsService(s.esx)
	c.Assert(err, IsNil, Commentf(out))

	// Make sure volume stays attached after vmdkopsd restart
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	misc.LogTestEnd(c.TestName())
}

// Test basic failover scenario by killing a VM and then power it on
// 1. Setup: Create a volume
// 2. Setup: Attach it to a container
// 3. Kill the VM
// 4. Verification: volume status should be detached
// 5. Power on the VM
// 6. Verification: volume status should still be detached
func (s *VMListenerTestParams) TestBasicFailover(c *C) {
	misc.LogTestStart(c.TestName())

	// Kill the VM
	isVMKilled := killVM(s.esx, s.vm1Name)
	c.Assert(isVMKilled, Equals, true, Commentf("VM [%s] was not killed successfully", s.vm1Name))

	// Status should be detached
	volStatus := verification.GetVMAttachedToVolUsingAdminCli(s.volumeName, s.esx)
	c.Assert(volStatus, Equals, properties.DetachedStatus, Commentf("Volume %s is still attached", s.volumeName))

	// Power on VM
	esxutil.PowerOnVM(s.vm1Name)
	isStatusChanged := esxutil.WaitForExpectedState(esxutil.GetVMPowerState, s.vm1Name, properties.PowerOnState)
	c.Assert(isStatusChanged, Equals, true, Commentf("VM [%s] should be powered on state", s.vm1Name))

	// Status should be detached
	status := verification.VerifyDetachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	misc.LogTestEnd(c.TestName())
}

// Test advanced failover scenario by killing a VM, mounting the volume to a different VM,
// unmounting it, and then power on the original VM.
// 1. Setup: Create a volume from VM1
// 2. Setup: Attach it to a container on VM1
// 3. Kill VM1
// 4. Verification: volume status should be detached
// 5. Attach it to a container on a different VM2
// 6. Verification: volume status should attached
// 7. Remove the container from VM2
// 8. Verification: volume status should be detached
// 9. Power on VM1
// 10. Verification: volume status should be still detached
func (s *VMListenerTestParams) TestFailoverAcrossVM(c *C) {
	misc.LogTestStart(c.TestName())

	// Kill VM1
	isVMKilled := killVM(s.esx, s.vm1Name)
	c.Assert(isVMKilled, Equals, true, Commentf("VM [%s] was not killed successfully", s.vm1Name))

	// Status should be detached
	volStatus := verification.GetVMAttachedToVolUsingAdminCli(s.volumeName, s.esx)
	c.Assert(volStatus, Equals, properties.DetachedStatus, Commentf("Volume %s is still attached", s.volumeName))

	// Attach to a container on a different VM2
	out, err := dockercli.AttachVolume(s.vm2, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be attached
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm2, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Remove the container from VM2
	out, err = dockercli.RemoveContainer(s.vm2, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be detached
	status = verification.VerifyDetachedStatus(s.volumeName, s.vm2, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// Power on VM1 which has been killed
	esxutil.PowerOnVM(s.vm1Name)
	isStatusChanged := esxutil.WaitForExpectedState(esxutil.GetVMPowerState, s.vm1Name, properties.PowerOnState)
	c.Assert(isStatusChanged, Equals, true, Commentf("VM [%s] should be powered on state", s.vm1Name))

	// Status should be still detached
	status = verification.VerifyDetachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	misc.LogTestEnd(c.TestName())
}

func killVM(esx, vmName string) bool {
	// Make sure VM is powered on
	vmState := esxutil.GetVMPowerState(vmName)
	if vmState != properties.PowerOnState {
		log.Printf("VM [%s] is in [%s] state", vmName, vmState)
		return false
	}

	// Kill VM
	isVMKilled := esxutil.KillVM(esx, vmName)
	if isVMKilled != true {
		log.Printf("Unable to kill VM [%s]", vmName)
		return false
	}

	isStatusChanged := esxutil.WaitForExpectedState(esxutil.GetVMPowerState, vmName, properties.PowerOffState)
	if isStatusChanged != true {
		log.Printf("VM [%s] is not in expected %s state", vmName, properties.PowerOffState)
		return false
	}

	// VM has been killed successfully and turned to powered-off state
	return true
}
