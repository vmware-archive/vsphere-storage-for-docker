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

// This test suite contains tests to verify behavior of default vmgroup

// +build runonce

package e2e

import (
	"strings"

	admincliconst "github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	dockerconst "github.com/vmware/docker-volume-vsphere/tests/constants/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

const (
	vgErrorMsg = "This feature is not supported for vmgroup _DEFAULT."
)

type DefaultVMGroupTestSuite struct {
	config      *inputparams.TestConfig
	volumeNames []string
}

func (s *DefaultVMGroupTestSuite) SetUpSuite(c *C) {
	defaultVG := "defaultVGTest"
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping vmgroupbasic tests.")
	}
	out, err := admincli.ConfigInit(s.config.EsxHost)
	c.Assert(err, IsNil, Commentf(out))

	// Verify DB successfully initialized
	c.Assert(admincli.GetDBmode(s.config.EsxHost), Equals, admincliconst.DBSingleNode, Commentf("Failed to init the DB mode on ESX -  .", s.config.EsxHost))
	s.volumeNames = []string{inputparams.GetUniqueVolumeName(defaultVG), inputparams.GetUniqueVolumeName(defaultVG)}
}

func (s *DefaultVMGroupTestSuite) TearDownSuite(c *C) {
	out, err := admincli.ConfigRemove(s.config.EsxHost)
	c.Assert(err, IsNil, Commentf(out))

	// Verifying DB successfully removed
	c.Assert(admincli.GetDBmode(s.config.EsxHost), Equals, admincliconst.DBNotConfigured, Commentf("Failed to remove the DB mode on ESX -  .", s.config.EsxHost))
}

var _ = Suite(&DefaultVMGroupTestSuite{})

// Test step:
// Add a vm to the _DEFAULT Vmgroup
func (s *DefaultVMGroupTestSuite) TestAddVMToDefaultTenant(c *C) {
	misc.LogTestStart(c.TestName())

	out, err := admincli.AddVMToVMgroup(s.config.EsxHost, admincliconst.DefaultVMgroup, s.config.DockerHostNames[1])
	c.Assert(err, Not(IsNil), Commentf(out))
	c.Assert(strings.TrimRight(string(out), "\n"), Equals, vgErrorMsg, Commentf("Unexpected Behavior: "+
		"We are able to add vm to the _DEFAULT tenant which is NOT allowed as per current spec."))

	misc.LogTestEnd(c.TestName())
}

// Test step:
// Remove a vm from the _DEFAULT Vmgroup
func (s *DefaultVMGroupTestSuite) TestRemoveDefaultTenantVMs(c *C) {
	misc.LogTestStart(c.TestName())

	out, err := admincli.RemoveVMFromVMgroup(s.config.EsxHost, admincliconst.DefaultVMgroup, s.config.DockerHostNames[1])
	c.Assert(err, Not(IsNil), Commentf(out))
	c.Assert(strings.TrimRight(string(out), "\n"), Equals, vgErrorMsg, Commentf("Unexpected Behavior: "+
		"We are able to remove vm from the _DEFAULT tenant which is NOT allowed as per current spec."))

	misc.LogTestEnd(c.TestName())
}

// Test step:
//  Replace a vm from the _DEFAULT Vmgroup
func (s *DefaultVMGroupTestSuite) TestReplaceDefaultTenantVMs(c *C) {
	misc.LogTestStart(c.TestName())

	out, err := admincli.ReplaceVMFromVMgroup(s.config.EsxHost, admincliconst.DefaultVMgroup, s.config.DockerHostNames[1])
	c.Assert(err, Not(IsNil), Commentf(out))
	c.Assert(strings.TrimRight(string(out), "\n"), Equals, vgErrorMsg, Commentf("Unexpected Behavior: "+
		"We are able to replace vm from the _DEFAULT tenant which is NOT allowed as per current spec."))

	misc.LogTestEnd(c.TestName())
}

// Verify volume creation after deleting _DEFAULT tenant
// 1. Create a volume
// 2. Verify volume from docker host and esx
// 2. Delete default tenant
// 3. Verify not able to create volume from docker host - Error: VM does not belong to any vmgroup
// 4. Verify docker volume ls does not show any volumes
// 5. Verify admin cli shows volume with VMGroup set as N/A
// 6. Create _DEFAULT vmgroup with —default-datastore=_VM_DS
// 7. Again create new volume - operation should succeed
// 8. Verify volume from esx and docker host

func (s *DefaultVMGroupTestSuite) TestDeleteDefaultVmgroup(c *C) {
	misc.LogTestStart(c.TestName())
	nullVmgroup := "N/A"
	noVgErrorMsg := "VolumeDriver.Create: VM " + s.config.DockerHostNames[0] + " does not belong to any vmgroup"

	// Verify default vmgroup is present
	isVmgroupAvailable := admincli.IsVmgroupPresent(s.config.EsxHost, admincliconst.DefaultVMgroup)
	c.Assert(isVmgroupAvailable, Equals, true, Commentf(" Vmgroup %s does not exist", admincliconst.DefaultVMgroup))

	// Create a volume
	out, err := dockercli.CreateVolume(s.config.DockerHosts[0], s.volumeNames[0])
	c.Assert(err, IsNil, Commentf(out))

	// Check if volume was successfully created
	isAvailable := verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[0])
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", s.volumeNames[0]))

	// Verify if volume exists in default vmgroup
	isVolInVmgroup := admincli.IsVolumeExistInVmgroup(s.config.EsxHost, admincliconst.DefaultVMgroup, s.volumeNames[0])
	c.Assert(isVolInVmgroup, Equals, true, Commentf("Volume [%s] does not belong to vmgroup [%s]", admincliconst.DefaultVMgroup, s.volumeNames[0]))

	// Delete default vmgroup and verify it does not exist
	admincli.DeleteVMgroup(s.config.EsxHost, admincliconst.DefaultVMgroup, false)

	isVmgroupAvailable = admincli.IsVmgroupPresent(s.config.EsxHost, admincliconst.DefaultVMgroup)
	c.Assert(isVmgroupAvailable, Equals, false, Commentf("Failed to delete vmgroup %s .", admincliconst.DefaultVMgroup))

	// Create a volume - operation should fail as default vmgroup no longer exist
	out, err = dockercli.CreateVolume(s.config.DockerHosts[0], s.volumeNames[1])
	c.Assert(err, Not(IsNil), Commentf(out))
	c.Assert(strings.Contains(out, noVgErrorMsg), Equals, true, Commentf(out))

	// Verify existing volume - volumeName1 does not belong to any vmgroup (N/A)
	isVolInVmgroup = admincli.IsVolumeExistInVmgroup(s.config.EsxHost, nullVmgroup, s.volumeNames[0])
	c.Assert(isVolInVmgroup, Equals, true, Commentf("Unexpected Behavior: Vmgroup for volume [%s] is not N/A. ", s.volumeNames[0]))

	isVmgroupAvailable = admincli.CreateDefaultVmgroup(s.config.EsxHost)
	c.Assert(isVmgroupAvailable, Equals, true, Commentf("Failed to create vmgroup %s .", admincliconst.DefaultVMgroup))

	// Create a volume - operation should succeed as default vmgroup exists now
	out, err = dockercli.CreateVolume(s.config.DockerHosts[0], s.volumeNames[1])
	c.Assert(err, IsNil, Commentf(out))

	// Check if volume was successfully created
	isAvailable = verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeNames[1])
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", s.volumeNames[1]))

	// Check volumes now again belong to default vmgroup
	isVolInVmgroup = admincli.IsVolumeListExistInVmgroup(s.config.EsxHost, admincliconst.DefaultVMgroup, s.volumeNames)
	c.Assert(isVolInVmgroup, Equals, true, Commentf("All volumes [%s] do not belong to vmgroup [%s]", s.volumeNames, admincliconst.DefaultVMgroup))

	out, err = ssh.InvokeCommand(s.config.DockerHosts[0], dockerconst.RemoveVolume+strings.Join(s.volumeNames, " "))
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}
