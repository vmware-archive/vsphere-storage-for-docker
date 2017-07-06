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

// This test is going to cover various vsan related test cases

// +build runonce

package e2e

import (
	"log"
	"strings"

	adminclicon "github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	dockerclicon "github.com/vmware/docker-volume-vsphere/tests/constants/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/esx"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"

	. "gopkg.in/check.v1"
)

type VsanTestSuite struct {
	config *inputparams.TestConfig

	vsanDSName string
	volumeName string
	policyList []string
}

func (s *VsanTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping vsan tests")
	}

	s.vsanDSName = esx.GetDatastoreByType("vsan")
	if s.vsanDSName == "" {
		c.Skip("Vsan datastore unavailable")
	}
}

func (s *VsanTestSuite) SetUpTest(c *C) {
	s.policyList = []string{}
}

func (s *VsanTestSuite) TearDownTest(c *C) {
	if s.volumeName != "" {
		out, err := dockercli.DeleteVolume(s.config.DockerHosts[0], s.volumeName)
		c.Assert(err, IsNil, Commentf(out))
	}
	if len(s.policyList) > 0 {
		for _, policyName := range s.policyList {
			out, err := admincli.RemovePolicy(s.config.EsxHost, policyName)
			c.Assert(err, IsNil, Commentf(out))
		}
	}
}

var _ = Suite(&VsanTestSuite{})

// Steps:
// 1. Create a valid vsan policy
// 2. Volume creation with valid policy should pass
// 3. Valid volume should be accessible
// 4. Start a container to mount the volume, and write message to the volume
// 5. Start another container to mount the volume, and read message from volume and verify the
// message is the same as the message written in step 4
func (s *VsanTestSuite) TestValidPolicy(c *C) {
	misc.LogTestStart(c.TestName())
	data1 := "message_by_host1"
	testFile := "test.txt"

	s.volumeName = ""
	policyName := "validPolicy"
	out, err := admincli.CreatePolicy(s.config.EsxHost, policyName, adminclicon.PolicyContent)
	c.Assert(err, IsNil, Commentf(out))
	s.policyList = append(s.policyList, policyName)

	s.volumeName = inputparams.GetUniqueVolumeName("vsanVol") + "@" + s.vsanDSName
	vsanOpts := " -o " + adminclicon.VsanPolicyFlag + "=" + policyName

	out, err = dockercli.CreateVolumeWithOptions(s.config.DockerHosts[0], s.volumeName, vsanOpts)
	c.Assert(err, IsNil, Commentf(out))
	isAvailable := verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeName)
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", s.volumeName))

	out, err = dockercli.WriteToVolume(s.config.DockerHosts[0], s.volumeName, inputparams.GetUniqueContainerName("vsan_test"),
		testFile, data1)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.ReadFromVolume(s.config.DockerHosts[0], s.volumeName, inputparams.GetUniqueContainerName("vsan_test"),
		testFile)
	c.Assert(err, IsNil, Commentf(out))
	c.Assert(out, Equals, data1)

	misc.LogTestEnd(c.TestName())
}

// Steps:
// 1. Create an invalid vsan policy (wrong content)
// 2. Volume creation with non existing policy should fail
// 3. Volume creation with invalid policy should fail
func (s *VsanTestSuite) TestInvalidPolicy(c *C) {
	misc.LogTestStart(c.TestName())
	s.volumeName = ""
	invalidContentPolicyName := "invalidPolicy"
	out, err := admincli.CreatePolicy(s.config.EsxHost, invalidContentPolicyName, "'((\"wrongKey\" i50)'")
	c.Assert(err, IsNil, Commentf(out))
	s.policyList = append(s.policyList, invalidContentPolicyName)

	invalidVsanOpts := [2]string{"-o " + adminclicon.VsanPolicyFlag + "=IDontExist", "-o " +
		adminclicon.VsanPolicyFlag + "=" + invalidContentPolicyName}
	for _, option := range invalidVsanOpts {
		invalidVolName := inputparams.GetUniqueVolumeName("vsanVol") + "@" + s.vsanDSName
		out, _ = dockercli.CreateVolumeWithOptions(s.config.DockerHosts[0], invalidVolName, option)
		c.Assert(strings.HasPrefix(out, dockerclicon.ErrorVolumeCreate), Equals, true)
	}

	misc.LogTestEnd(c.TestName())
}

// The purpose of this test is to verify:
// 1) a volume can be created with vsan policy specified
// 2) A vsan policy cannot be removed if volumes still use it

// Test step:
// 1. create a vsan policy
// 2. run "vmdkops_admin policy ls", check the "Active" column of the output to make sure it
// is shown as "Unused"
// 3. create a volume on vsanDatastore with the vsan policy we created
// 4. run "docker volume inspect" on the volume to verify the output "vsan-policy-name" field
// 5. run "vmdkops_admin policy ls", check the "Active" column of the output to make sure it
// is shown as "In use by 1 volumes"
// 6. run "vmdkops_admin policy rm" to remove the policy, which should fail since the volume is still
// use the vsan policy
func (s *VsanTestSuite) TestDeleteVsanPolicyAlreadyInUse(c *C) {
	misc.LogTestStart(c.TestName())
	s.volumeName = ""
	out, err := admincli.CreatePolicy(s.config.EsxHost, adminclicon.PolicyName, adminclicon.PolicyContent)
	c.Assert(err, IsNil, Commentf(out))

	s.policyList = append(s.policyList, adminclicon.PolicyName)

	res := admincli.VerifyActiveFromVsanPolicyListOutput(s.config.EsxHost, adminclicon.PolicyName, "Unused")
	c.Assert(res, Equals, true, Commentf("vsanPolicy should be \"Unused\""))

	s.volumeName = inputparams.GetUniqueVolumeName("vsanVol") + "@" + s.vsanDSName
	vsanOpts := " -o " + adminclicon.VsanPolicyFlag + "=" + adminclicon.PolicyName
	out, err = dockercli.CreateVolumeWithOptions(s.config.DockerHosts[0], s.volumeName, vsanOpts)
	c.Assert(err, IsNil, Commentf(out))

	policyName, err := verification.GetAssociatedPolicyName(s.config.DockerHosts[0], s.volumeName)
	c.Assert(err, IsNil, Commentf("Get associated policy for volume %s failed", s.volumeName))
	c.Assert(policyName, Equals, adminclicon.PolicyName, Commentf("The name of vsan policy used by volume "+s.vsanDSName+" returns incorrect value "+policyName))

	res = admincli.VerifyActiveFromVsanPolicyListOutput(s.config.EsxHost, adminclicon.PolicyName, "In use by 1 volumes")
	c.Assert(res, Equals, true, Commentf("vsanPolicy should be \"In use by 1 volumes\""))

	out, err = admincli.RemovePolicy(s.config.EsxHost, adminclicon.PolicyName)
	log.Printf("Remove vsanPolicy \"%s\" returns with %s", adminclicon.PolicyName, out)
	c.Assert(out, Matches, "Error: Cannot remove.*", Commentf("vsanPolicy is still used by volumes and cannot be removed"))

	misc.LogTestEnd(c.TestName())

}

// Test creation of a vsan volume on a non vsan datastore.
// 1. Create a valid vsan policy
// 2. Get the name of VMFS datastore
// 3. Create the volume on VMFS datastore but specify the vsan policy
// 4. Creation should fail. Also verify that volume is not available
func (s *VsanTestSuite) TestNonVSANDatastore(c *C) {
	misc.LogTestStart(c.TestName())

	// create valid policy
	out, err := admincli.CreatePolicy(s.config.EsxHost, adminclicon.PolicyName, adminclicon.PolicyContent)
	c.Assert(err, IsNil, Commentf(out))
	s.policyList = append(s.policyList, adminclicon.PolicyName)

	// get list of non-vsan (vmfs) datastores
	localDSList := esx.GetDatastoreByType("VMFS")
	if localDSList == "" {
		log.Printf("Local datastore unavailable")
		c.Skip("Local datastore unavailable")
	}

	localDS := strings.Fields(localDSList)[0]

	// specify vsan policy name as volume create option
	invalidVolName := inputparams.GetUniqueVolumeName("vsanVol") + "@" + localDS
	vsanOpts := " -o " + adminclicon.VsanPolicyFlag + "=" + adminclicon.PolicyName

	// this create should fail
	out, err = dockercli.CreateVolumeWithOptions(s.config.DockerHosts[0], invalidVolName, vsanOpts)
	c.Assert(err, Not(IsNil), Commentf(out))

	// this availibility check should fail
	isAvailable := verification.CheckVolumeAvailability(s.config.DockerHosts[0], invalidVolName)
	c.Assert(isAvailable, Equals, false, Commentf("Volume %s creation should have failed. Instead it succeeded.", invalidVolName))

	misc.LogTestEnd(c.TestName())
}
