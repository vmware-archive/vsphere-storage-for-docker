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
	"os"

	. "gopkg.in/check.v1"

	"github.com/vmware/docker-volume-vsphere/tests/constants/properties"
	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/esxcli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/govc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

type VMListenerTestParams struct {
	volumeName     string
	dockerHostIP   string
	dockerHostName string
	containerName  string
	esxIP          string
}

func (s *VMListenerTestParams) SetUpSuite(c *C) {
	s.dockerHostIP = os.Getenv("VM2")
	s.dockerHostName = govc.RetrieveVMNameFromIP(s.dockerHostIP)
	s.esxIP = os.Getenv("ESX")
}

func (s *VMListenerTestParams) SetUpTest(c *C) {
	s.volumeName = inputparams.GetUniqueVolumeName("vmlistener_test")
	s.containerName = inputparams.GetUniqueContainerName("vmlistener_test")

	out, err := dockercli.CreateVolume(s.dockerHostIP, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.AttachVolume(s.dockerHostIP, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	status := verification.VerifyAttachedStatus(s.volumeName, s.dockerHostIP, s.esxIP)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))
}

func (s *VMListenerTestParams) TearDownTest(c *C) {
	if dockercli.IsContainerExist(s.dockerHostIP, s.containerName) {
		out, err := dockercli.RemoveContainer(s.dockerHostIP, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}
	out, err := dockercli.DeleteVolume(s.dockerHostIP, s.volumeName)
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

	// restart vmdkops service
	out, err := admincli.RestartVmdkopsService(s.esxIP)
	c.Assert(err, IsNil, Commentf(out))

	// make sure volume stays attached after vmdkopsd restart
	status := verification.VerifyAttachedStatus(s.volumeName, s.dockerHostIP, s.esxIP)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	misc.LogTestEnd(c.TestName())
}

// Test kill VM process
// 1. Setup: Create a volume
// 2. Setup: Attach it to a container
// 3. Kill the VM
// 4. Verification: volume status should be detached
// 5. Power on the VM
// 6. Verification: volume status should still be detached
func (s *VMListenerTestParams) TestKillVM(c *C) {
	misc.LogTestStart(c.TestName())

	// make sure vm was powered on
	powerState := govc.GetVMPowerState(s.dockerHostName)
	log.Printf("VM[%s]'s current power state is [%s]", s.dockerHostName, powerState)
	c.Assert(powerState, Equals, properties.PowerOnState, Commentf("VM [%s] should be powered on state", s.dockerHostName))

	// grab worldID/vmProcessID
	processID := esxcli.GetVMProcessID(s.esxIP, s.dockerHostName)
	log.Printf("VM's process ID is: %s", processID)

	// kill VM
	isVMKilled := esxcli.KillVM(s.esxIP, processID)
	c.Assert(isVMKilled, Equals, true, Commentf("Unable to kill VM %s ...", s.dockerHostName))

	isStatusChanged := misc.WaitForExpectedState(govc.GetVMPowerState, s.dockerHostName, properties.PowerOffState)
	c.Assert(isStatusChanged, Equals, true, Commentf("VM [%s] should be powered off state", s.dockerHostName))

	// status should be detached
	volStatus := verification.GetVMAttachedToVolUsingAdminCli(s.volumeName, s.esxIP)
	c.Assert(volStatus, Equals, properties.DetachedStatus, Commentf("Volume %s is still attached", s.volumeName))

	// power on VM
	govc.PowerOnVM(s.dockerHostName)
	isStatusChanged = misc.WaitForExpectedState(govc.GetVMPowerState, s.dockerHostName, properties.PowerOnState)
	c.Assert(isStatusChanged, Equals, true, Commentf("VM [%s] should be powered on state", s.dockerHostName))

	// status should be detached
	status := verification.VerifyDetachedStatus(s.volumeName, s.dockerHostIP, s.esxIP)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	misc.LogTestEnd(c.TestName())
}
