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

// This test suite contains miscellaneous tests to verify behavior of non-default vmgroup

// +build runonce

package e2e

import (
	admincliconst "github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/esx"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

const (
	vmGroupName = "vg_basictest"
)

type vgBasicSuite struct {
	config      *inputparams.TestConfig
	volumeNames []string
}

func (s *vgBasicSuite) SetUpSuite(c *C) {
	basicVG := "basicVGTest"
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping vmgroupbasic tests.")
	}
	admincli.ConfigInit(s.config.EsxHost)

	// Verify DB successfully initialized
	c.Assert(admincli.GetDBmode(s.config.EsxHost), Equals, admincliconst.DBSingleNode, Commentf("Failed to init the DB mode on ESX -  .", s.config.EsxHost))
	s.volumeNames = []string{inputparams.GetUniqueVolumeName(basicVG), inputparams.GetUniqueVolumeName(basicVG)}
}

func (s *vgBasicSuite) SetUpTest(c *C) {
	// Creating non-default vmgroup only if it does not exists
	if admincli.IsVmgroupPresent(s.config.EsxHost, vmGroupName) {
		return
	}
	admincli.CreateVMgroup(s.config.EsxHost, vmGroupName, s.config.DockerHostNames[0], s.config.Datastores[0])

	// Verify if vmgroup exists
	isVmgroupAvailable := admincli.IsVmgroupPresent(s.config.EsxHost, vmGroupName)
	c.Assert(isVmgroupAvailable, Equals, true, Commentf("vmgroup ls command does not lists the vmgroup %s .", vmGroupName))

	// Verify vm belongs to vmgroup
	isVMPartofVg := admincli.IsVMInVmgroup(s.config.EsxHost, s.config.DockerHostNames[0], vmGroupName)
	c.Assert(isVMPartofVg, Equals, true, Commentf("VM %s does not belong to vmgroup %s .", s.config.DockerHostNames[0], vmGroupName))
}

func (s *vgBasicSuite) TearDownSuite(c *C) {
	admincli.RemoveVMFromVMgroup(s.config.EsxHost, vmGroupName, s.config.DockerHostNames[0])

	// Verify vm does not belongs to vmgroup
	isVMPartofVg := admincli.IsVMInVmgroup(s.config.EsxHost, s.config.DockerHostNames[0], vmGroupName)
	c.Assert(isVMPartofVg, Equals, false, Commentf("Unexpected Behavior: VM %s belong to vmgroup %s .", s.config.DockerHostNames[0], vmGroupName))

	admincli.DeleteVMgroup(s.config.EsxHost, vmGroupName, true)
	// Verify vmgroup does not exist
	isVmgroupAvailable := admincli.IsVmgroupPresent(s.config.EsxHost, vmGroupName)
	c.Assert(isVmgroupAvailable, Equals, false, Commentf("Failed to delete the vmgroup [%s] .", vmGroupName))

	// Removing the DB at the end of suite
	admincli.ConfigRemove(s.config.EsxHost)

	// Verifying DB successfully removed
	c.Assert(admincli.GetDBmode(s.config.EsxHost), Equals, admincliconst.DBNotConfigured, Commentf("Failed to remove the DB mode on ESX -  .", s.config.EsxHost))
}

var _ = Suite(&vgBasicSuite{})

/*
//TO DO: Please reenable or remove this test based on the conclusion of the Issue # 1469

// Verify vmgroup for orphaned volumes is specified as N/A
// 1. Create a vmgroup and add vm to it.
// 2. Verify vmgroup was successfully created.
// 3. Create a volume from vm-2.
// 4. Verify volumes vmgroup from esx.
// 5. Remove vm-2 from vmgroup.
// 6. Verify volume is not visible from vm-2.
// 7. Delete the tenant.
// 8. Again verify volumes vmgroup is listed as N/A.

func (s *vgBasicSuite) TestVGNameForOrphanedVolumes(c *C) {
	misc.LogTestStart(c.TestName())
	nullVmgroup := "N/A"

	// Create a volume from vmgroup's vm
	out, err := dockercli.CreateVolume(s.config.DockerHosts[0], s.volumeNames[0])
	c.Assert(err, IsNil, Commentf(out))

	// Check if volume was successfully created
	isAvailable := verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[0])
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", s.volumeNames[0]))

	// Verify if volume exists in vmgroup
	isVolInVmgroup := admincli.IsVolumeExistInVmgroup(s.config.EsxHost, vmGroupName, s.volumeNames[0])
	c.Assert(isVolInVmgroup, Equals, true, Commentf("Volume [%s] does not belong to vmgroup [%s]", s.volumeNames[0], vmGroupName))

	admincli.RemoveVMFromVMgroup(s.config.EsxHost, vmGroupName, s.config.DockerHostNames[0])

	// Check if volume was not visible from vm since it does not belong to vmgroup
	isAvailable = verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[0])
	c.Assert(isAvailable, Equals, false, Commentf("Unexpected Behavior: Volume %s belonging to "+
		"vmgroup [%s] is visible from host [%s] which does not belong to the same vmgroup", s.volumeNames[0], vmGroupName, s.config.DockerHosts[0]))

	// Now delete the vmgroup
	admincli.DeleteVMgroup(s.config.EsxHost, vmGroupName)

	// Verify vmgroup does not exist
	isVmgroupAvailable := admincli.IsVmgroupPresent(s.config.EsxHost, vmGroupName)
	c.Assert(isVmgroupAvailable, Equals, false, Commentf("Failed to delete the vmgroup [%s] .", vmGroupName))

	// Verify vmgroup for volume is N/A
	isVolInVmgroup = admincli.IsVolumeExistInVmgroup(s.config.EsxHost, nullVmgroup, s.volumeNames[0])
	c.Assert(isVolInVmgroup, Equals, true, Commentf("Unexpected Behavior: Vmgroup [%s] for volume [%s] is not N/A. ", s.volumeNames[0], vmGroupName))

	// verify orpahned volume is not visible from host
	isAvailable = verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[0])
	c.Assert(isAvailable, Equals, false, Commentf("Unexpected Behavior: Orphaned volume %s "+
		"is visible from host [%s] ", s.volumeNames[0], s.config.DockerHosts[0]))

	misc.LogTestEnd(c.TestName())
}
*/

// TestUserVGDatastoreAccessPrivilege - Verify volumes can be
// created by a VM as long as the non-default vmgroup has "allow_create" right given
// 1. Create vmgroup VG1 and add vm VM2 to it
// 2. Add datastore to VG1 - By default no "allow_create" right is given
// 3. Create volume in VG1 and expect error as allow-create is false
// 4. Create volume creation from VM1 (In _DEFAULT vmgroup) and expects successful volume creation
// 5. Change allow-create access to true
// 6. Repeat step-3 and this time volume creation should succeed
// 7. Update datastore access for VG1 (remove --allow-create rule)
// 8. Repeat step#5 and this time expect the error
func (s *vgBasicSuite) TestDSAccessPrivilegeForUserVG(c *C) {
	misc.LogTestStart(c.TestName())

	// Add another datastore to vmgroup
	admincli.AddDatastoreToVmgroup(s.config.EsxHost, vmGroupName, s.config.Datastores[1])

	// Verify if vmgroup does not have access-rights for DS
	isDatastoreAccessible := admincli.IsDSAccessibleForVMgroup(s.config.EsxHost, vmGroupName, s.config.Datastores[1])
	c.Assert(isDatastoreAccessible, Equals, false, Commentf("Unexpected Behavior: Datastore %s is accessible for vmgroup %s .", s.config.Datastores[1], vmGroupName))

	// Create a volume from _DEFAULT vmgroup's vm - operation should succeed
	out, err := dockercli.CreateVolume(s.config.DockerHosts[1], s.volumeNames[1])
	c.Assert(err, IsNil, Commentf(out))

	// Check if volume was successfully created
	isAvailable := verification.CheckVolumeAvailability(s.config.DockerHosts[1], s.volumeNames[1])
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", s.volumeNames[1]))

	// This volume create will fail because of trying to create volume on the non-default DS.
	out, err = dockercli.CreateVolume(s.config.DockerHosts[0], s.volumeNames[0]+"@"+s.config.Datastores[1])
	c.Assert(err, Not(IsNil), Commentf(out))

	// Check volume was not created
	isAvailable = verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[0]+"@"+s.config.Datastores[1])
	c.Assert(isAvailable, Equals, false, Commentf("Unexpected behavior: Volume %s is successfully created "+
		" even though vmgroup [%s] does not have access rights for the datastore %s", s.volumeNames[1], vmGroupName, s.config.Datastores[1]))

	// Set the create privilege on the vmgroup  for specified datastore
	out, _ = admincli.SetCreateAccessForVMgroup(s.config.EsxHost, vmGroupName, s.config.Datastores[1])
	isDatastoreAccessible = admincli.IsDSAccessibleForVMgroup(s.config.EsxHost, vmGroupName, s.config.Datastores[1])
	c.Assert(isDatastoreAccessible, Equals, true, Commentf("Datstore %s is not accessible for vmgroup %s .", s.config.Datastores[1], vmGroupName))

	// Create a volume from non-default vmgroup's vm - operation should succeed this time
	// as access-rights have been changed to True
	out, err = dockercli.CreateVolume(s.config.DockerHosts[0], s.volumeNames[0]+"@"+s.config.Datastores[1])
	c.Assert(err, IsNil, Commentf(out))

	// Check if volume was successfully created
	isAvailable = verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[0]+"@"+s.config.Datastores[1])
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", s.volumeNames[1]))

	// verify able to delete  volume
	out, err = dockercli.DeleteVolume(s.config.DockerHosts[0], s.volumeNames[0]+"@"+s.config.Datastores[1])
	c.Assert(err, IsNil, Commentf(out))

	// Check if volume was successfully deleted
	isAvailable = verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[0]+"@"+s.config.Datastores[1])
	c.Assert(isAvailable, Equals, false, Commentf("Failed to delete volume %s", s.volumeNames[0]))

	// Remove the create privilege on the non-default vmgroup for specified datastore
	admincli.RemoveCreateAccessForVMgroup(s.config.EsxHost, vmGroupName, s.config.Datastores[1])

	// Verify if vmgroup does not have access-rights for DS
	isDatastoreAccessible = admincli.IsDSAccessibleForVMgroup(s.config.EsxHost, vmGroupName, s.config.Datastores[1])
	c.Assert(isDatastoreAccessible, Equals, false, Commentf("Datastore %s is accessible for vmgroup %s .", s.config.Datastores[1], vmGroupName))

	// This volume create will fail because of trying to create volume on ds with no create privilege.
	volumeName := inputparams.GetUniqueVolumeName(vmGroupName)
	out, err = dockercli.CreateVolume(s.config.DockerHostNames[0], volumeName+"@"+s.config.Datastores[1])
	c.Assert(err, Not(IsNil), Commentf(out))

	// Check if volume was not created
	isAvailable = verification.CheckVolumeAvailability(s.config.DockerHosts[0], volumeName+"@"+s.config.Datastores[1])
	c.Assert(isAvailable, Equals, false, Commentf("Unexpected behavior: Volume %s is successfully created "+
		" even though vmgroup [%s] does not have access rights for the datastore %s ", s.volumeNames[1], vmGroupName, s.config.Datastores[1]))

	out, err = dockercli.DeleteVolume(s.config.DockerHosts[1], s.volumeNames[1])
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// Test steps:
// 1. Create a vm VM1 using govc
// 2. Create a vmgroup and associate VM1 to it.
// 2. Execute vmgroup ls command.
// 3. Delete the vm that was added to the vmgroup.
// 4. Again execute vmgroup ls command to verify command works fine.
func (s *vgBasicSuite) TestDeleteVMFromVmgroup(c *C) {
	misc.LogTestStart(c.TestName())
	vmName := "VM_" + inputparams.GetRandomNumber()
	vgName := "VG_" + inputparams.GetRandomNumber()
	networkAdapterType := "vmxnet3"

	// Create a vm - we need this to add to vmgroup and later on delete this vm
	esx.CreateVM(vmName, s.config.Datastores[0], networkAdapterType)
	c.Assert(esx.IsVMExist(vmName), Equals, true, Commentf("Failed to create VM - %s .", vmName, vgName))

	out, err := admincli.CreateVMgroup(s.config.EsxHost, vgName, vmName, admincliconst.VMHomeDatastore)
	c.Assert(err, IsNil, Commentf(out))

	// Verify if vmgroup exists
	isVmgroupAvailable := admincli.IsVmgroupPresent(s.config.EsxHost, vgName)
	c.Assert(isVmgroupAvailable, Equals, true, Commentf("vmgroup %s does not exists.", vgName))

	// Verify vm belongs to vmgroup
	isVMPartofVg := admincli.IsVMInVmgroup(s.config.EsxHost, vmName, vgName)
	c.Assert(isVMPartofVg, Equals, true, Commentf("VM %s does not belong to vmgroup %s .", vmName, vgName))

	// Destroy the vm
	esx.DestroyVM(vmName)
	c.Assert(esx.IsVMExist(vmName), Equals, false, Commentf("Failed to delete VM - %s .", vmName, vgName))

	// Check vmgroup ls
	isVmgroupAvailable = admincli.IsVmgroupPresent(s.config.EsxHost, vgName)
	c.Assert(isVmgroupAvailable, Equals, true, Commentf("vmgroup %s does not exists.", vgName))

	// TO DO: Due to product behavior vmgroup deletion is not possible - Issue # 1484
	// Please add step to delete vmgroup after issue 1484 is fixed.
	misc.LogTestEnd(c.TestName())
}
