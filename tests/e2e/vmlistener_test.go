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

	adminconst "github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/constants/properties"
	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
	esxutil "github.com/vmware/docker-volume-vsphere/tests/utils/esx"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

const (
	restartHostd = "/etc/init.d/hostd restart"
	killHostd = "kill -9 `pidof hostd`"
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
	vsanDSName    string
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
	s.vsanDSName = esxutil.GetDatastoreByType("vsan")
}

func (s *VMListenerTestParams) SetUpTest(c *C) {
	s.volumeName = inputparams.GetUniqueVolumeName("vmlistener_test")
	s.containerName = inputparams.GetUniqueContainerName("vmlistener_test")
}

var _ = Suite(&VMListenerTestParams{})

// Test vmdkops service restart
// 1. Create a volume
// 2. Attach the volume to a container
// 3. Verification: volume is in attached status
// 4. Restart the vmdkops service
// 5. Verification: volume stays as attached
// 6. Remove the container
// 7. Remove the volume
func (s *VMListenerTestParams) TestServiceRestart(c *C) {
	misc.LogTestStart(c.TestName())

	// Create the volume
	out, err := dockercli.CreateVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Attach the volume
	out, err = dockercli.AttachVolume(s.vm1, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Verify the volume is in attached status
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Restart vmdkops service
	out, err = admincli.RestartVmdkopsService(s.esx)
	c.Assert(err, IsNil, Commentf(out))

	// Make sure volume stays attached after vmdkopsd restart
	status = verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Remove the container
	out, err = dockercli.RemoveContainer(s.vm1, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Remove the volume
	out, err = dockercli.DeleteVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// Test basic failover scenario by killing a VM and then power it on
// 1. Create a volume
// 2. Attach the volume to a container
// 3. Verification: volume is in attached status
// 4. Kill the VM
// 5. Verification: volume status should be detached
// 6. Power on the VM
// 7. Verification: volume status should still be detached
// 8. Remove the container if it still exists
// 9. Remove the volume
func (s *VMListenerTestParams) TestBasicFailover(c *C) {
	misc.LogTestStart(c.TestName())

	// Create the volume
	out, err := dockercli.CreateVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Attach the volume
	out, err = dockercli.AttachVolume(s.vm1, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Verify the volume is in attached status
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Kill the VM
	isVMKilled := killVM(s.esx, s.vm1Name)
	c.Assert(isVMKilled, Equals, true, Commentf("VM [%s] was not killed successfully", s.vm1Name))

	// Status should be detached
	volStatus := verification.GetVMAttachedToVolUsingAdminCli(s.volumeName, s.esx)
	c.Assert(volStatus, Equals, properties.DetachedStatus, Commentf("Volume %s is still attached", s.volumeName))

	// Power on VM
	esxutil.PowerOnVM(s.vm1Name)
	isVDVSRunning := esxutil.IsVDVSRunningAfterVMRestart(s.vm1, s.vm1Name)
	c.Assert(isVDVSRunning, Equals, true, Commentf("vDVS is not running after VM [%s] being restarted", s.vm1Name))

	// Status should be detached
	status = verification.PollDetachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	// Remove the container if it still exists
	if dockercli.IsContainerExist(s.vm1, s.containerName) {
		out, err := dockercli.RemoveContainer(s.vm1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}

	// Remove the volume
	out, err = dockercli.DeleteVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// Test advanced failover scenario on VMFS datastore by killing a VM,
// mounting the volume to a different VM, unmounting it, and then
// power on the original VM.
// 1. Create a volume from VM1
// 2. Attach it to a container on VM1
// 3. Verification: volume is in attached status
// 4. Kill VM1
// 5. Verification: volume status should be detached
// 6. Attach it to a container on a different VM2
// 7. Verification: volume status should attached
// 8. Remove the container from VM2
// 9. Verification: volume status should be detached
// 10. Power on VM1
// 11. Verification: volume status should be still detached
// 12. Remove the container if it still exists
// 13. Remove the volume
func (s *VMListenerTestParams) TestFailoverAcrossVmOnVmfs(c *C) {
	misc.LogTestStart(c.TestName())

	// Create the volume
	out, err := dockercli.CreateVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Attach the volume
	out, err = dockercli.AttachVolume(s.vm1, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Verify the volume is in attached status
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Kill VM1
	isVMKilled := killVM(s.esx, s.vm1Name)
	c.Assert(isVMKilled, Equals, true, Commentf("VM [%s] was not killed successfully", s.vm1Name))

	// Status should be detached
	volStatus := verification.GetVMAttachedToVolUsingAdminCli(s.volumeName, s.esx)
	c.Assert(volStatus, Equals, properties.DetachedStatus, Commentf("Volume %s is still attached", s.volumeName))

	// Attach to a container on a different VM2
	out, err = dockercli.AttachVolume(s.vm2, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be attached
	status = verification.VerifyAttachedStatus(s.volumeName, s.vm2, s.esx, s.vm2Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Remove the container from VM2
	out, err = dockercli.RemoveContainer(s.vm2, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be detached
	status = verification.VerifyDetachedStatus(s.volumeName, s.vm2, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// Power on VM1 which has been killed
	esxutil.PowerOnVM(s.vm1Name)
	isVDVSRunning := esxutil.IsVDVSRunningAfterVMRestart(s.vm1, s.vm1Name)
	c.Assert(isVDVSRunning, Equals, true, Commentf("vDVS is not running after VM [%s] being restarted", s.vm1Name))

	// Status should be still detached
	status = verification.VerifyDetachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// Remove the container if it still exists
	if dockercli.IsContainerExist(s.vm1, s.containerName) {
		out, err := dockercli.RemoveContainer(s.vm1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}

	// Remove the volume
	out, err = dockercli.DeleteVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// Test advanced failover scenario on VSAN datastore by killing a VM,
// mounting the volume to a different VM, unmounting it, and then
// power on the original VM.
// 1. Create a VSAN policy
// 2. Create a volume with VSAN policy from VM1
// 3. Attach the volume to a container on VM1
// 4. Verification: volume is in attached status
// 5. Kill VM1
// 6. Verification: volume status should be detached
// 7. Attach it to a container on a different VM2
// 8. Verification: volume status should attached
// 9. Remove the container from VM2
// 10. Verification: volume status should be detached
// 11. Power on VM1
// 12. Verification: volume status should be still detached
// 13. Remove the container if it still exists
// 14. Remove the volume
// 15. Remove the VSAN policy
func (s *VMListenerTestParams) TestFailoverAcrossVmOnVsan(c *C) {
	misc.LogTestStart(c.TestName())

	// Create the VSAN policy
	out, err := admincli.CreatePolicy(s.config.EsxHost, adminconst.PolicyName, adminconst.PolicyContent)
	c.Assert(err, IsNil, Commentf(out))

	// Create the volume with VSAN policy
	fullVolumeName := s.volumeName + "@" + s.vsanDSName
	vsanOpts := " -o " + adminconst.VsanPolicyFlag + "=" + adminconst.PolicyName

	out, err = dockercli.CreateVolumeWithOptions(s.vm1, fullVolumeName, vsanOpts)
	c.Assert(err, IsNil, Commentf(out))

	// Attach the volume
	out, err = dockercli.AttachVolume(s.vm1, fullVolumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Verify the volume is in attached status
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Kill VM1
	isVMKilled := killVM(s.esx, s.vm1Name)
	c.Assert(isVMKilled, Equals, true, Commentf("VM [%s] was not killed successfully", s.vm1Name))

	// Status should be detached
	volStatus := verification.GetVMAttachedToVolUsingAdminCli(s.volumeName, s.esx)
	c.Assert(volStatus, Equals, properties.DetachedStatus, Commentf("Volume %s is still attached", s.volumeName))

	// Attach to a container on a different VM2
	out, err = dockercli.AttachVolume(s.vm2, fullVolumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be attached
	status = verification.VerifyAttachedStatus(s.volumeName, s.vm2, s.esx, s.vm2Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// Remove the container from VM2
	out, err = dockercli.RemoveContainer(s.vm2, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be detached
	status = verification.VerifyDetachedStatus(s.volumeName, s.vm2, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// Power on VM1 which has been killed
	esxutil.PowerOnVM(s.vm1Name)
	isVDVSRunning := esxutil.IsVDVSRunningAfterVMRestart(s.vm1, s.vm1Name)
	c.Assert(isVDVSRunning, Equals, true, Commentf("vDVS is not running after VM [%s] being restarted", s.vm1Name))

	// Status should be still detached
	status = verification.PollDetachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// Remove the container if it still exists
	if dockercli.IsContainerExist(s.vm1, s.containerName) {
		out, err := dockercli.RemoveContainer(s.vm1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}

	// Remove the volume
	out, err = dockercli.DeleteVolume(s.vm1, fullVolumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Remove the VSAN policy
	out, err = admincli.RemovePolicy(s.esx, adminconst.PolicyName)
	log.Printf("Remove vsanPolicy \"%s\" returns with %s", adminconst.PolicyName, out)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// TestVolumeAttachedForHostdRestart - verify an in use volume
// remains attached when hostd is restarted.
// 1. Restart hostd
// 2. Verify volume stays as attached
// 3. Kill the vm
// 4. Status of the volume should be detached.
func (s *VMListenerTestParams) TestVolumeAttachedForHostdRestart(c *C) {
	misc.LogTestStart(c.TestName())

	// Create the volume
	out, err := dockercli.CreateVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Attach the volume
	out, err = dockercli.AttachVolume(s.vm1, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 1. Restart hostd
	out, err = ssh.InvokeCommand(s.esx, restartHostd)
	c.Assert(err, IsNil, Commentf(out))

	// Hostd reports running status in about a second but
	// takes around 10 - 15 seconds to be able to serve requests
	misc.SleepForSec(15)

	// 2. Verify volume stays attached
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// 3. Kill this VM
	isVMKilled := killVM(s.esx, s.vm1Name)
	c.Assert(isVMKilled, Equals, true, Commentf("VM [%s] was not killed successfully", s.vm1Name))

	// Give some time for the VM event to be handled
	misc.SleepForSec(5)

	// 4. Status should be detached
	status = verification.VerifyDetachedStatus(s.volumeName, s.vm2, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// 5. Restore VM
	esxutil.PowerOnVM(s.vm1Name)
	isVDVSRunning := esxutil.IsVDVSRunningAfterVMRestart(s.vm1, s.vm1Name)
	c.Assert(isVDVSRunning, Equals, true, Commentf("vDVS is not running after VM [%s] being restarted", s.vm1Name))

	// 6. Status should be detached (verify on the VM)
	status = verification.PollDetachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// Remove the container if it still exists, after step 6 it should not exist ideally
	if dockercli.IsContainerExist(s.vm1, s.containerName) {
		out, err := dockercli.RemoveContainer(s.vm1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}

	// Remove the volume
	out, err = dockercli.DeleteVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// TestVolumeAttachedForVMSuspend - verify an in use volume
// remains attached when VM is suspended/resumed
// 1. Suspend the VM and resume it again.
// 2. Resume is like a power-on and so include the same checks
// 3. Verify volume stays attached
// 4. Remove the container if it still exists
// 5. Remove the volume
func (s *VMListenerTestParams) TestVolumeAttachedForVMSuspend(c *C) {
	misc.LogTestStart(c.TestName())

	// Create the volume
	out, err := dockercli.CreateVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Attach the volume
	out, err = dockercli.AttachVolume(s.vm1, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 1. Suspend and resume the VM
	out, err = esxutil.SuspendResumeVM(s.esx, s.vm1Name)
	c.Assert(err, IsNil, Commentf(out))

	// 2. Resume is like a power-on and so include the same checks
	isVDVSRunning := esxutil.IsVDVSRunningAfterVMRestart(s.vm1, s.vm1Name)
	c.Assert(isVDVSRunning, Equals, true, Commentf("vDVS is not running after VM [%s] being restarted", s.vm1Name))

	// 3. Verify volume stays attached
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// 4. Remove the container if it still exists
	if dockercli.IsContainerExist(s.vm1, s.containerName) {
		out, err := dockercli.RemoveContainer(s.vm1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}

	// 5. Remove the volume
	out, err = dockercli.DeleteVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// TestVolumeAttachedWhenHostdKilled verify volume remains
// attached when hostd is killed.
// 1. Kill hostd
// 2. Verify volume is attached
// 3. Kill the vm
// 4. Status of the volume should be detached.
func (s *VMListenerTestParams) TestVolumeAttachedWhenHostdKilled(c *C) {
	misc.LogTestStart(c.TestName())

	// Create the volume
	out, err := dockercli.CreateVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Attach the volume
	out, err = dockercli.AttachVolume(s.vm1, s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 1. Kill hostd
	out, err = ssh.InvokeCommand(s.esx, killHostd)
	c.Assert(err, IsNil, Commentf(out))

	// Hostd reports running status in about a second but
	// takes around 10 - 15 seconds to be able to serve requests
	misc.SleepForSec(15)

	// 2. Verify volume stays attached
	status := verification.VerifyAttachedStatus(s.volumeName, s.vm1, s.esx, s.vm1Name)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// 3. Kill this VM
	isVMKilled := killVM(s.esx, s.vm1Name)
	c.Assert(isVMKilled, Equals, true, Commentf("VM [%s] was not killed successfully", s.vm1Name))

	// Give time for the listener to detach the vmdk
	misc.SleepForSec(5)

	// 4. Status should be detached
	status = verification.VerifyDetachedStatus(s.volumeName, s.vm2, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// 5. Restore VM
	esxutil.PowerOnVM(s.vm1Name)
	isVDVSRunning := esxutil.IsVDVSRunningAfterVMRestart(s.vm1, s.vm1Name)
	c.Assert(isVDVSRunning, Equals, true, Commentf("vDVS is not running after VM [%s] being restarted", s.vm1Name))

	// 6. Status should be detached (verify on the VM)
	status = verification.PollDetachedStatus(s.volumeName, s.vm1, s.esx)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))

	// Remove the container if it still exists, shouldn't exist if the volume was detached
	if dockercli.IsContainerExist(s.vm1, s.containerName) {
		out, err := dockercli.RemoveContainer(s.vm1, s.containerName)
		c.Assert(err, IsNil, Commentf(out))
	}

	// Remove the volume
	out, err = dockercli.DeleteVolume(s.vm1, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

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
