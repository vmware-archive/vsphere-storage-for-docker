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

// This test is going to create volume on the fresh testbed very first time.
// After installing vmdk volume plugin/driver, volume creation should not be
// failed very first time.

// This test is going to cover the issue reported at #656
// TODO: as of now we are running the test against photon vm it should be run
// against various/applicable linux distros.

package e2e

import (
	"os"
	"testing"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	"github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
)

const (
	volName1                 = "vmgroup_test_vol_1"
	volName2                 = "vmgroup_test_vol_2"
	defaultVg                = "_DEFAULT"
	defCname                 = "def_test_ctr"
	defVgName                = "def_test_vg"
)

// Tests to validate behavior with the __DEFAULT_ vmgroup.

var dockerHosts = []string{os.GetEnv{"Vm1"}, os.GetEnv("VM2")}
var host, vmgroup, ds1, ds2, cname string

func getEnv() {
	host := os.GetEnv("ESX")
	vmgroup := os.GetEnv("T1")
	ds1 := os.GetEnv("DS1")
	ds1 := os.GetEnv("DS2")
	cname := os.GetEnv("CNAME")

	if vg == "" {
		vg = defVgName
	}

	if ds1 == "" || ds2 == "" {
		t.Errorf("Unknown datastores, please set DS1 and DS2 to valid datastore names in the environment")
	}

	if cname == "" {
		cname = defCname
	}
}

func createVolumeOnDefaultVg(t *testing.T, name string) {
	// 1. Create the volume on each host?
	for _, vm := range dockerHosts {
		err := dockercli.CreateVolume(vm, name, "")
		if err != nil {
			t.Errorf("Error while creating volume - %s on VM - %s, err - %s", name, vm, err.Error())
		}
	}

	// 2. Verify the volume is created on the default vm group
	cmd := admincli.ListVolumes + " | grep " + defaultVg + " | grep " + name
	val, err := ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error while listing volumes [%s] in default vmgroup on host %s [%s] err - %s", cmd, host, err.Error())
	}
	if val == "" {
		t.Errorf("Volume %s not found in default vmgroup", name)
	}
}

// TestVolumeCreateOnDefaultVg - Verify that volumes can be created on the
// default volume group with default permissions, then attached and deleted
func TestVolumeCreateAndAttachOnDefaultVg(t *testing.T) {
	getEnv()
	// Create a volume in the default group
	createVolumeOnDefaultVg(t, volName1)

	// 1. Verify volume can be mounted and used on at least one of the VMs
	_, err := dockercli.AttachVolume(dockerHosts[0], volName1, cname)
	if err != nil {
		t.Errorf("Error while attaching volume - %s on VM - %s, err - %s", volName1, dockerHosts[0], err.Error())
	}

	// 2. Delete the volume in the default group
	_, err := dockercli.DeleteVolume(dockerHosts[0], volName1)
	if err != nil {
		t.Errorf("Error while deleting volume - %s in the default vmgroup, err - %s", volName1, err.Error())
	}
	t.Logf("Passed - Volume create and attach on default vmgroup")
}

// TestVolumeAccessAcrossVolumeGroups - Verify volumes can be accessed only
// from VMs that belong to the volume group
func TestVolumeAccessAcrossVolumeGroups(t *testing.T) {
	getEnv()
	// Create a volume in the default group
	createVolumeOnDefaultVg(t, volName1)

	// 1. Create a new vmgroup, with ds2 as the datastore
	cmd := admincli.CreateVolumeGroup + vmgroup + " --vm-list " + dockerHosts[0] + " --default-datastore " + ds2
	_, err := ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error while creating volume group - [%s] err - %s", cmd, err.Error())
	}

	// 2. Try to attach the volume created in the default volume group
	_, err = dockercli.AttachVolume(dockerHosts[0], volName1, cname)
	if err == nil {
		t.Errorf("Expected error when attaching volume %s in default vmgroup from VM %s in vmgroup %s", volName1, dockerHosts[0], vmgroup)
	}

	// 3. Try deleting volume in default group
	_, err = dockercli.DeleteVolume1(dockerHosts[0], volName1)
	if err == nil {
		t.Errorf("Expected error when deleting volume %s in default vmgroup from VM %s in vmgroup %s", volName1, dockerHosts[0], vmgroup)
	}

	// 4. Remove the volume from the default vmgroup, from the other VM
	_, err = dockercli.DeleteVolume(dockerHosts[1], volName1)
	if err != nil {
		t.Errorf("Error when deleting volume %s in default vmgroup from VM %s in vmgroup %s", volName1, dockerHosts[1], vmgroup)
	}

	// 5. Remove the VM from the new vmgroup
	cmd = admincli.RemoveVMsForVolumeGroup + vmgroup + " --vm-list " + dockerHosts[0]
	_, err = ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error when removing VM %s from vmgroup on host %s - [%s], err %s", dockerHosts[1], host, cmd, err.Error())
	}

	// 6. Remove the new vmgroup
	cmd = admincli.DeleteVolumeGroup + vmgroup
	_, err = ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error when removing vmgroup %s on host %s - [%s], err %s", dockerHosts[1], host, cmd, err.Error())
	}

	t.Logf("Passed - Volume access across vmgroups")
}

// TestCreateAccessPrivilegeOnDefaultVg - Verify volumes can be
// created by a VM as long as the vmgroup has the allow-create setting
// enabled on it
func TestCreateAccessPrivilegeOnDefaultVg(t *testing.T) {
	getEnv()
	// Create a volume in the default group
	createVolumeOnDefaultVg(t, volName1)

	// 1. Attach volume from default vmgroup
	_, err := dockercli.AttachVolume(dockerHosts[0], volName1, cname)
	if err != nil {
		t.Errorf("Error while attaching volume - %s on VM - %s, err - %s", volName1, dockerHosts[0], err.Error())
	}

	// 2. Remove the create privilege on the default vmgroup for specified datastore
	cmd := admincli.ModifyAccessForVolumeGroup + defaultVg + "--allow-create False --datastore " + ds1
	_, err = ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error when setting access privileges [%s] on default vmgroup on host %s, err - %s", cmd, host, err.Error())
	}

	// 3. Try creating a volume on the default vmgroup
	err := dockercli.CreateVolume(dockerHosts[0], volName2, "")
	if err == nil {
		t.Errorf("Expected error while creating volume - %s from VM - %s, on default vmgroup, err - %s", volName2, dockerHosts[0], err.Error())
	}

	// 4. Restore the create privilege on the default vmgroup for specified datastore
	cmd := admincli.ModifyAccessForVolumeGroup + defaultVg + "--allow-create True --datastore " + ds1
	_, err = ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error when restoring access privileges [%s] on default vmgroup on host %s, err - %s", cmd, host, err.Error())
	}

	// 5. Remove the volume created earlier
	_, err := dockercli.DeleteVolume(dockerHosts[0], volName1)
	if err != nil {
		t.Errorf("Error while deleting volume - %s in the default vmgroup, err - %s", volName1, err.Error())
	}
	t.Logf("Passed - create privilege on default vmgroup")
}

func TestVolumeCreateOnVg(t *testing.T) {
	getEnv()
	// 1. Create a new vmgroup
	cmd := admincli.CreateVolumeGroup + vmgroup + " --vm-list " + dockerHosts[0] + " --default-datastore " + ds1
	_, err := ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error while creating volume group - [%s] err - %s", cmd, err.Error())
	}

	// 2. Create a volume in the new vmgroup
	err := dockercli.CreateVolume(dockerHosts[0], volName2, "")
	if err != nil {
		t.Errorf("Error while creating volume - %s from VM - %s, err - %s", volName2, dockerHosts[0], err.Error())
	}

	// 3. Try attaching volume in new vmgroup
	_, err := dockercli.AttachVolume(dockerHosts[0], volName2, cname)
	if err != nil {
		t.Errorf("Error while attaching volume - %s on VM - %s, err - %s", volName1, dockerHosts[0], err.Error())
	}

	// 4. Remove the volume from the new vmgroup
	_, err := dockercli.DeleteVolume(dockerHosts[0], volName2)
	if err != nil {
		t.Errorf("Expected error when deleting volume %s in default vmgroup from VM %s in vmgroup %s", volName2, dockerHosts[0], vmgroup)
	}

	// 5. Remove the VM from the new vmgroup
	cmd = admincli.RemoveVMsForVolumeGroup + vmgroup + " --vm-list " + dockerHosts[0]
	_, err = ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error when removing VM %s from vmgroup on host %s - [%s], err %s", dockerHosts[1], host, cmd, err.Error())
	}

	// 6. Remove the new vmgroup
	cmd = admincli.DeleteVolumeGroup + vmgroup
	ssh.ExecCmd(host, cmd)

	t.Logf("Passed - create and attach volumes on a non-default vmgroup")
}

func TestVerifyMaxFileSizeOnVg(t *testing.T) {
	getEnv()
	// 1. Create a vmgroup and add VM to it
	cmd := admincli.CreateVolumeGroup + vmgroup + " --default-datastore " + ds1
	_, err := ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error while creating volume group - [%s] err - %s", cmd, err.Error())
	}

	// 2. Ensure the max file size and total size is set to 1G each.
	cmd := admincli.ModifyAccessForVolumeGroup + defaultVg + " --datastore " + ds1 + " --volume-maxsize=1gb --volume-totalsize=1gb --allow-create=True"
	_, err = ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error when setting max and total size [%s] on vmgroup %s on host %s, err - %s", cmd, vmgroup, host, err.Error())
	}

	// 3. Try creating volumes up to the max filesize and the totalsize
	err := dockercli.CreateVolume(dockerHosts[0], volName1, "-o size=1gb")
	if err != nil {
		t.Errorf("Error while creating volume - %s from VM - %s, err - %s", volName1, dockerHosts[0], err.Error())
	}

	// 4. Try creating a volume of 1gb again, should fail as totalsize is already reached
	err := dockercli.CreateVolume(dockerHosts[0], volName2, "-o size=1gb")
	if err == nil {
		t.Errorf("Expected error while creating volume - %s from VM - %s, err - %s", volName2, dockerHosts[0], err.Error())
	}

	// 5. Ensure the max file size and total size is set to 1G each.
	cmd := admincli.ModifyAccessForVolumeGroup + defaultVg + " --datastore " + ds1 + " --volume-maxsize=1gb --volume-totalsize=2gb --allow-create=True"
	_, err = ssh.ExecCmd(host, cmd)
	if err != nil {
		t.Errorf("Error when setting max and total size [%s] on vmgroup %s on host %s, err - %s", cmd, vmgroup, host, err.Error())
	}

	// 6. Try creating a volume of 1gb again, should succeed as totalsize is increased to 2gb
	err := dockercli.CreateVolume(dockerHosts[0], volName2, "-o size=1gb")
	if err != nil {
		t.Errorf("Error while creating volume - %s from VM - %s, err - %s", volName2, dockerHosts[0], err.Error())
	}

	// 7. Delete both volumes
	dockercli.DeleteVolume(dockerHosts[0], volName1)
	dockercli.DeleteVolume(dockerHosts[0], volName2)

	// 8. Remove the vmgroup
	cmd = admincli.DeleteVolumeGroup + vmgroup
	ssh.ExecCmd(host, cmd)

	t.Logf("Passed - verified volumes can be created to match total size assigned to a vmgroup")
}

func TestVolumeVisibilityOnVg(t *testing.T) {
	t.Skip("Not supported")
}

func TestVolumeMaxSizeAndTotalSizeOndVg(t *testing.T) {
	t.Skip("Not supported")

}

