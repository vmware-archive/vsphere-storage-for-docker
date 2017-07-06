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

// This test suite contains tests to verify behaviors of non-default vmgroup

// +build runonce

package e2e

import (
	"log"
	"strings"

	adminconst "github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	adminutils "github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

const (
	vgTestVMgroup1     = "vmgroup_test1"
	vgTestVMgroup2     = "vmgroup_test2"
	vgTestContainer    = "vmgroupContainer"
	vmgroupVMRemoveErr = "Cannot complete vmgroup vm rm"
)

// VmGroupTest - struct for vmgroup tests
type VmGroupTest struct {
	config        *inputparams.TestConfig
	vmgroup       string
	testContainer string
	volName1      string
	volName2      string
	volName3      string
}

var _ = Suite(&VmGroupTest{})

func (vg *VmGroupTest) SetUpSuite(c *C) {
	vg.testContainer = inputparams.GetUniqueContainerName(vgTestContainer)

	vg.config = inputparams.GetTestConfig()
	if vg.config == nil {
		c.Skip("Unable to retrieve test config, skipping vmgroup tests")
	}
	adminutils.ConfigInit(vg.config.EsxHost)

	cmd := adminconst.GetAccessForVMgroup + vgTestVMgroup1
	out, err := ssh.InvokeCommand(vg.config.EsxHost, cmd)
	if err == nil {
		log.Printf(out)
	}
	// Create the test VM group1
	cmd = adminconst.CreateVMgroup + vgTestVMgroup1 + " --default-datastore " + vg.config.Datastores[0]
	log.Printf("Creating test vmgroup %s", vgTestVMgroup1)
	out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
	c.Assert(err, IsNil, Commentf(out))

	// Add the VM to vmgroup vgTestVMgroup1
	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	// Add the VM to vmgroup vgTestVMgroup1
	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[1])
	c.Assert(err, IsNil, Commentf(out))

	// Create the test VM group2
	cmd = adminconst.CreateVMgroup + vgTestVMgroup2 + " --default-datastore " + vg.config.Datastores[1]
	log.Printf("Creating test vmgroup %s", vgTestVMgroup2)
	out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
	c.Assert(err, IsNil, Commentf(out))

	cmd = adminconst.ListVMgroups
	out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
	log.Printf(out)

	log.Printf("Done creating vmgroups test config.")
}

func (vg *VmGroupTest) SetUpTest(c *C) {
	// Create volume names used for the test
	vg.vmgroupGetVolName(c)
}

func (vg *VmGroupTest) TearDownSuite(c *C) {
	// A failed test may leave the VM in either of these groups
	adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[1])
	adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup2, vg.config.DockerHostNames[0])
	adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup2, vg.config.DockerHostNames[1])

	// Remove both test vmgroups
	out, err := adminutils.DeleteVMgroup(vg.config.EsxHost, vgTestVMgroup1, true)
	c.Assert(err, IsNil, Commentf(out))

	out, err = adminutils.DeleteVMgroup(vg.config.EsxHost, vgTestVMgroup2, true)
	c.Assert(err, IsNil, Commentf(out))

	cmd := adminconst.ListVMgroups
	out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
	log.Printf(out)

	// Remove Config DB
	adminutils.ConfigRemove(vg.config.EsxHost)

	log.Printf("Done cleanup of vmgroups test config.")
}

func (vg *VmGroupTest) vmgroupGetVolName(c *C) {
	vg.volName1 = inputparams.GetUniqueVolumeName(c.TestName())
	vg.volName2 = inputparams.GetUniqueVolumeName(c.TestName())
	vg.volName3 = inputparams.GetUniqueVolumeName(c.TestName())
}

// Tests to validate behavior with the __DEFAULT_ vmgroup.

func (vg *VmGroupTest) createVolumes(c *C, name string) {
	// 1. Create the volume on host
	out, err := dockercli.CreateVolume(vg.config.DockerHosts[0], name)
	c.Assert(err, IsNil, Commentf(out))

	// 2. Verify the volume is created on the default vm group
	val, err := dockercli.ListVolumes(vg.config.DockerHosts[0])
	c.Assert(err, IsNil, Commentf(val))
	c.Assert(strings.Contains(val, name), Equals, true, Commentf("Volume %s not found in default vmgroup", name))
}

// TestVmGroupVolumeCreate - Verify that volumes can be created on the
// default vmgroup with default permissions, then attached and deleted
// Assumes: VM (VM1) belongs to the default VM group.
// 1. Create a volume in the default vmgroup
// 2. Verify the VM is able to attach and run a container with the volume
// 3. Delete the volume
func (vg *VmGroupTest) TestVmGroupVolumeCreate(c *C) {
	misc.LogTestStart(c.TestName())

	// Create a volume in the default group
	vg.createVolumes(c, vg.volName1)

	// 1. Verify volume can be mounted and unmounted
	out, err := dockercli.ExecContainer(vg.config.DockerHosts[0], vg.volName1, vg.testContainer)
	c.Assert(err, IsNil, Commentf(out))

	// Docker may not have completed the detach yet with the host.
	misc.SleepForSec(2)

	// 2. Delete the volume in the default group
	out, err = dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName1)
	c.Assert(err, IsNil, Commentf(out))
	c.Logf("Passed - Volume create and attach on default vmgroup")

	misc.LogTestEnd(c.TestName())
}

// TestVmGroupVolumeAccessAcrossVmGroups - Verify volumes can be accessed only
// from VMs that belong to the vmgroup
// Assumes: VMs (VM1i and VM2) belongs to the default VM group.
// 1. Create a volume in the default VM group from VM1
// 2. Create a new vmgroup and add VM1 to it with vg.config.Datastore[1] as its default
// 3. Try attaching the volume created in the default group from VM1 - expect error
// 4. Try deleteing the volume in the default group from VM1 - expect error
// 5. Try deleting the volume in th default group from VM2
// 6. Remove the newly created vmgroup
func (vg *VmGroupTest) TestVmGroupVolumeAccessAcrossVmGroups(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Create a volume in the default group
	vg.createVolumes(c, vg.volName1)

	// 2. Remove VM from test group 1 and add to test group 2
	out, err := adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup2, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	// 3. Try to inspect the volume created in the default vmgroup, trying to run a container
	// causes Docker to figure the volume isn't there and creates a local volume.
	out, err = dockercli.InspectVolume(vg.config.DockerHosts[0], vg.volName1)
	c.Assert(err, Not(IsNil), Commentf(out))

	// 4. Try deleting volume in default group
	out, err = dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName1)
	c.Assert(err, Not(IsNil), Commentf(out))

	// 5. Remove the volume from the default , from the other VM
	out, err = dockercli.DeleteVolume(vg.config.DockerHosts[1], vg.volName1)
	c.Assert(err, IsNil, Commentf(out))

	// 6. Remove from the test vmgroup 2 and add back to test vmgroup 1
	out, err = adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup2, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	c.Logf("Passed - Volume access across vmgroups")
	misc.LogTestEnd(c.TestName())
}

// TestVmGroupCreateAccessPrivilege - Verify volumes can be
// created by a VM as long as the vmgroup has the allow-create setting
// enabled on it
// Assumes: VM1 is in the default vmgroup
// 1. Create volume in default group from vm VM1
// 2. Try attaching volume from VM1 and run a container
// 3. Remove the create privilege from the default vmgroup
// 4. Try create a volume in the default vmgroup - expect error
// 5. Restore create privilege on default vmgroup
// 6. Remove volume created in (1).
func (vg *VmGroupTest) TestVmGroupCreateAccessPrivilege(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Create a volume in the default vmgroup
	vg.createVolumes(c, vg.volName1)

	// 2. Attach volume from default vmgroup and unmount
	out, err := dockercli.ExecContainer(vg.config.DockerHosts[0], vg.volName1, vg.testContainer)
	c.Assert(err, IsNil, Commentf(out))

	// 3. Remove the create privilege on the default  for specified datastore
	out, err = adminutils.RemoveCreateAccessForVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.Datastores[0])
	c.Assert(err, IsNil, Commentf(out))

	cmd := adminconst.GetAccessForVMgroup + vgTestVMgroup1 + " | grep " + vg.config.Datastores[0] + " | grep False"
	out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
	c.Assert(err, IsNil, Commentf(out))

	// 4. Try creating a volume on the default vmgroup
	out, err = dockercli.CreateVolume(vg.config.DockerHosts[0], vg.volName2)
	if err == nil {
		cmd = adminconst.GetAccessForVMgroup + vgTestVMgroup1
		out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
		log.Printf(out)
	}
	c.Assert(err, Not(IsNil), Commentf(out))

	// 5. Restore the create privilege on the default  for specified datastore
	out, err = adminutils.SetCreateAccessForVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.Datastores[0])
	c.Assert(err, IsNil, Commentf(out))

	// 6. Remove the volume created earlier
	out, err = dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName1)
	c.Assert(err, IsNil, Commentf(out))

	c.Logf("Passed - create privilege on default vmgroup")
	misc.LogTestEnd(c.TestName())
}

// TestVmGroupVolumeCreateOnVg - Verify basic volume create/attach/delete
// on non-default vmgroup
// 1. Create a new vmgroup and place VM VM1 in it
// 2. Create volume in vmgroup
// 3. Attach volume and run a container
// 4. Delete volume created in (2)
// 5. Destroy the VM group
func (vg *VmGroupTest) TestVmGroupVolumeCreateOnVg(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Remove VM from test group 1 and add to test group 2
	out, err := adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup2, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	// 2. Create a volume in the new vmgroup
	out, err = dockercli.CreateVolume(vg.config.DockerHosts[0], vg.volName2)
	c.Assert(err, IsNil, Commentf(out))

	// 3. Try attaching volume in new vmgroup and detach
	out, err = dockercli.ExecContainer(vg.config.DockerHosts[0], vg.volName2, vg.testContainer)
	c.Assert(err, IsNil, Commentf(out))

	// Docker may not have completed the detach yet with the host.
	misc.SleepForSec(2)

	// 4. Remove the volume from the new vmgroup
	out, err = dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName2)
	c.Assert(err, IsNil, Commentf(out))

	// 5. Remove from the test vmgroup 2 and add back to test vmgroup 1
	out, err = adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup2, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	c.Logf("Passed - create and attach volumes on a non-default vmgroup")
	misc.LogTestEnd(c.TestName())
}

// TestVmGroupVerifyMaxFileSizeOnVg - Verify that enough volumes can be created
// to match the totalsize for a vmgroup and verify that volumes of the
// maxsize can be created.
// 1. Set maxsize and totalsize to 1G each in the new vmgroup
// 2. Try creating a volume of 1gb
// 3. Try creating another volume of 1gb, 1023mb, 1024mb, 1025mb - expect error
// 4. Set maxsize and total size as 1gb and 2gb respectively
// 5. Retry step (4) - expect success this time
// 6. Remove both volumes
func (vg *VmGroupTest) TestVmGroupVerifyMaxFileSizeOnVg(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Ensure the max file size and total size is set to 1G each.
	out, err := adminutils.SetVolumeSizeForVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.Datastores[0], "1gb", "1gb")
	c.Assert(err, IsNil, Commentf(out))

	// 2. Try creating volumes up to the max filesize and the totalsize
	out, err = dockercli.CreateVolumeWithOptions(vg.config.DockerHosts[0], vg.volName1, "-o size=1gb")
	c.Assert(err, IsNil, Commentf(out))
	if err != nil {
		cmd := adminconst.GetAccessForVMgroup + vgTestVMgroup1
		out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
		log.Printf(out)
	}

	// 3. Try creating a volume of 1gb again, should fail as totalsize is already reached
	out, err = dockercli.CreateVolumeWithOptions(vg.config.DockerHosts[0], vg.volName2, "-o size=1gb")
	if err == nil {
		cmd := adminconst.GetAccessForVMgroup + vgTestVMgroup1
		out, err = ssh.InvokeCommand(vg.config.EsxHost, cmd)
		log.Printf(out)
	}
	c.Assert(err, Not(IsNil), Commentf(out))

	out, err = dockercli.CreateVolumeWithOptions(vg.config.DockerHosts[0], vg.volName3, "-o size=1023mb")
	c.Assert(err, Not(IsNil), Commentf(out))

	/*
		out, err = dockercli.CreateVolumeWithOptions(vg.config.DockerHosts[0], vg.volName2, "-o size=1024mb")
		c.Assert(err, Not(IsNil), Commentf(out))

		out, err = dockercli.CreateVolumeWithOptions(vg.config.DockerHosts[0], vg.volName2, "-o size=1025mb")
		c.Assert(err, Not(IsNil), Commentf(out))
	*/

	// 4. Ensure the max file size and total size is set to 1G and 2G each.
	out, err = adminutils.SetVolumeSizeForVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.Datastores[0], "1gb", "2gb")
	c.Assert(err, IsNil, Commentf(out))

	// 5. Try creating a volume of 1gb again, should succeed as totalsize is increased to 2gb
	out, err = dockercli.CreateVolumeWithOptions(vg.config.DockerHosts[0], vg.volName2, "-o size=1024mb")
	c.Assert(err, IsNil, Commentf(out))

	// 6. Delete both volumes
	dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName1)
	dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName2)

	c.Logf("Passed - verified volumes can be created to match total size assigned to a vmgroup")
	misc.LogTestEnd(c.TestName())
}

// TestVmGroupVolumeMobility - verify a VM with a volume
// attached cannot move across vmgroups or be removed from
// a vmgroup to default
// 1. Create a volume in the VM's vmgroup
// 2. Run a container and keep it running
// 3. Attempt removing the VM from vmgroup (to default) (should fail)
// 4. Delete container and attempt 3 (should pass)
func (vg *VmGroupTest) TestVmGroupVolumeMobility(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Create a volume in the default group
	vg.createVolumes(c, vg.volName1)

	// 2. Run a container
	out, err := dockercli.AttachVolume(vg.config.DockerHosts[0], vg.volName1, vg.testContainer)
	c.Assert(err, IsNil, Commentf(out))

	// 3. Attempt removing VM from test group 1
	out, err = adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	if !strings.Contains(out, vmgroupVMRemoveErr) {
		c.Fail()
	}

	// 4. Remove the container
	out, err = dockercli.RemoveContainer(vg.config.DockerHosts[0], vg.testContainer)
	c.Assert(err, IsNil, Commentf(out))

	// 5. Attempt removing VM from test group 1
	out, err = adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	// 6. And add back to test vmgroup 1
	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.DockerHostNames[0])
	c.Assert(err, IsNil, Commentf(out))

	// 7. Remove the volume
	dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName1)

	c.Logf("Passed - VM removal from vmgroups, with volume attached")
	misc.LogTestEnd(c.TestName())
}

// TestVmGroupVolumeClone - try cloning a volume from a non-default
// vm group
// 1. Create a volume in the VM's vmgroup
// 2. Clone a volume from the volume created in (1)
// 3. Cleanup and remove the volume
func (vg *VmGroupTest) TestVmGroupVolumeClone(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Create a volume in the VM's vmgroup
	vg.createVolumes(c, vg.volName1)

	// 2. Clone a volume from the one created in (1)
	cloneVolOpt := "-o clone-from=" + vg.volName1
	out, err := dockercli.CreateVolumeWithOptions(vg.config.DockerHosts[0], vg.volName2, cloneVolOpt)
	c.Assert(err, IsNil, Commentf(out))

	// 3. Remove the volume
	out, err = dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName1)
	c.Assert(err, IsNil, Commentf(out))
	out, err = dockercli.DeleteVolume(vg.config.DockerHosts[0], vg.volName2)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// Restore VMgroup
func (vg *VmGroupTest) restoreVmgroup(c *C) {
	vmList := vg.config.DockerHostNames[0] + "," + vg.config.DockerHostNames[1]
	cmd := adminconst.CreateVMgroup + vgTestVMgroup1 + " --default-datastore " + vg.config.Datastores[0]
	log.Printf("Recreating test vmgroup %s", vgTestVMgroup1)
	out, err := ssh.InvokeCommand(vg.config.EsxHost, cmd)
	c.Assert(err, IsNil, Commentf(out))

	out, err = adminutils.AddVMToVMgroup(vg.config.EsxHost, vgTestVMgroup1, vmList)
	c.Assert(err, IsNil, Commentf(out))
}

// test to verify volume is removed after user created vmgroup is removed with "--remove-volume" option
// 1. create a user created vmgroup "vmgroup_test1" and add VMs to this vmgroup
// 2. create two volumes for this vmgroup, one is on default datastore, and the other is on non default_datastore
// 3. verify mount/unmount for those two volumes
// 4. after unmount, verify volume is detached for those two volumes
// 5. remove VMs from this vmgroup, and remove the vmgroup with "--remove-volume" option
// 6. verify that volumes do not exist after removing vmgroup
func (vg *VmGroupTest) TestVmgroupRemoveWithRemoveVol(c *C) {
	misc.LogTestStart(c.TestName())

	// Create a volume in vgTestVMgroup1 on default_datastore
	vg.createVolumes(c, vg.volName1)

	// Add create privilege for the second datastore
	out, err := adminutils.AddCreateAccessForVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.Datastores[1])
	c.Assert(err, IsNil, Commentf(out))

	// Create another volume on second datastore
	// need to use full name for volume name
	vg.createVolumes(c, vg.volName2+"@"+vg.config.Datastores[1])

	// Run a container and then remove it using the first volume
	out, err = dockercli.ExecContainer(vg.config.DockerHosts[0], vg.volName1, vg.testContainer)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be detached after removing the container
	status := verification.VerifyDetachedStatus(vg.volName1, vg.config.DockerHosts[0], vg.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", vg.volName1))

	// Run a container and then remove it using the first volume
	out, err = dockercli.ExecContainer(vg.config.DockerHosts[0], vg.volName2+"@"+vg.config.Datastores[1], vg.testContainer)
	c.Assert(err, IsNil, Commentf(out))

	// Status should be detached after removing the container
	status = verification.VerifyDetachedStatus(vg.volName2, vg.config.DockerHosts[0], vg.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", vg.volName2+"@"+vg.config.Datastores[1]))

	// remove VM1 and VM2 from vgTestVMgroup1 and then remove the vmgroup
	vmList := vg.config.DockerHostNames[0] + "," + vg.config.DockerHostNames[1]
	out, err = adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vmList)
	c.Assert(err, IsNil, Commentf(out))

	out, err = adminutils.DeleteVMgroup(vg.config.EsxHost, vgTestVMgroup1, true)
	c.Assert(err, IsNil, Commentf(out))

	// vmgroup has been removed and all volumes belong to it have been removed
	// "err.Error()" will be filled with "exit status 1"
	volumeList := []string{vg.volName1, vg.volName2}
	for _, volume := range volumeList {
		out, err = verification.GetVMGroupForVolume(vg.config.EsxHost, volume)
		log.Println("GetVMGroupForVolume return out[%s] err[%s] for volume %s", out, err, volume)
		c.Assert(err.Error(), Equals, "exit status 1", Commentf("volume %s should be removed", volume))
	}
	// Restore vmgroup
	vg.restoreVmgroup(c)
	misc.LogTestEnd(c.TestName())
}

// TODO: Need to enable or remove the following tests after we have conclusion on issue #1469
// test to verify volume is removed after user created vmgroup is removed without "--remove-volume" option
// 1. create a user created vmgroup "vmgroup_test1" and add VMs to this vmgroup
// 2. create two volumes for this vmgroup, one is on default datastore, and the other is on non-default datastore
// 3. verify mount/unmount for this volume
// 4. after unmount, verify the volume is detached for both volumes
// 5. remove VMs from this vmgroup, and remove the vmgroup without "--remove-volume" option
// 6. verify that volumes exist but do not belong to any vmgroup
// func (vg *VmGroupTest) TestVmgroupRemove(c *C) {
// 	misc.LogTestStart(c.TestName())

// 	Create a volume in vgTestVMgroup1
// 	vg.createVolumes(c, vg.volName1)

// 	Add create privilege for the second datastore
// 	out, err := adminutils.AddCreateAccessForVMgroup(vg.config.EsxHost, vgTestVMgroup1, vg.config.Datastores[1])
// 	c.Assert(err, IsNil, Commentf(out))

// 	Create another volume on second datastore
// 	vg.createVolumes(c, vg.volName2+"@"+vg.config.Datastores[1])

// 	Verify volume can be mounted and unmounted for the first volume
// 	out, err = dockercli.ExecContainer(vg.config.DockerHosts[0], vg.volName1, vg.testContainer)
// 	c.Assert(err, IsNil, Commentf(out))

// 	Status should be detached
// 	status := verification.VerifyDetachedStatus(vg.volName1, vg.config.DockerHosts[0], vg.config.EsxHost)
// 	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", vg.volName1))

// 	Verify volume can be mounted and unmounted for the second volume
// 	out, err = dockercli.ExecContainer(vg.config.DockerHosts[0], vg.volName2+"@"+vg.config.Datastores[1], vg.testContainer)
// 	c.Assert(err, IsNil, Commentf(out))

// 	Status should be detached
// 	status = verification.VerifyDetachedStatusNonDefaultDS(vg.volName2, vg.config.DockerHosts[0], vg.config.EsxHost)
// 	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", vg.volName2+"@"+vg.config.Datastores[1]))

// 	remove VM1 and VM2 from vgTestVMgroup1 and then remove the vmgroup
// 	vmList := []string{vg.config.DockerHostNames[0], vg.config.DockerHostNames[1]}
// 	for _, vm := range vmList {
// 		out, err = adminutils.RemoveVMFromVMgroup(vg.config.EsxHost, vgTestVMgroup1, vm)
// 		c.Assert(err, IsNil, Commentf(out))
// 	}

// 	log.Printf("Removing test vmgroup %s", vgTestVMgroup1)
// 	out, err = adminutils.DeleteVMgroup(vg.config.EsxHost, vgTestVMgroup1, false)
// 	c.Assert(err, IsNil, Commentf(out))

// 	volumeList := []string{vg.volName1, vg.volName2}
// 	for _, volume := range volumeList {
// 		volume does not belong to any vmgroup, so "out" is expected to be "N/A"
// 		out, err = verification.GetVMGroupForVolume(vg.config.EsxHost, volume)
// 		log.Println("GetVMGroupForVolume return out[%s] err[%s] for volume %s", out, err, volume)
// 		c.Assert(out, Equals, "N/A", Commentf("volume %s should not belong to any vmgroup", volume))
// 	}

// Restore vmgroup
//  vg.restoreVmgroup(c)

// 	misc.LogTestEnd(c.TestName())
// }
