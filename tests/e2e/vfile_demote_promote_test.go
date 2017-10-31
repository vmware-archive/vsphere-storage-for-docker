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

// This test suite includes test cases to verify basic functionality
// before and after swarm node demote/promote operation

// +build runoncevfile

package e2e

import (
	"log"
	"strconv"
	"time"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

// vFile Demote/Promote test:
// This test is used to check vFile volume functionality after swarm node role change.
// In this test, we first create a vFile volume before the role change.
// Then a worker node is promoted to manager, and the original manager is demoted to worker.
// After the role change, we create another vFile volume.
// At last, we attach to both of the two vFile volumes to verify their functionality.

type VFileDemotePromoteTestSuite struct {
	config         *inputparams.TestConfig
	esx            string
	master         string
	worker1        string
	volName1       string
	volName2       string
	container1Name string
	container2Name string
}

func (s *VFileDemotePromoteTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping basic vfile tests")
	}

	s.esx = s.config.EsxHost
	s.master = inputparams.GetSwarmManager1()
	s.worker1 = inputparams.GetSwarmWorker1()
}

func (s *VFileDemotePromoteTestSuite) SetUpTest(c *C) {
	s.volName1 = inputparams.GetVFileVolumeName()
	s.volName2 = inputparams.GetVFileVolumeName()
	s.container1Name = inputparams.GetUniqueContainerName(c.TestName())
	s.container2Name = inputparams.GetUniqueContainerName(c.TestName())
}

var _ = Suite(&VFileDemotePromoteTestSuite{})

// All VMs are created in a shared datastore
// Test steps:
// 1. Create the 1st volume on worker1
// 2. Verify the 1st volume is available
// 3. Attach the 1st volume on worker1
// 4. Promote worker1 to manager
// 5. Sleep 20 seconds
// 6. Demote original manager to worker
// 7. Sleep 20 seconds
// 8. Create the 2nd volume on new worker (s.master)
// 9. Verify the 2nd volume is available
// 10. Attach the 2nd volume on new worker (s.master)
// 11. Verify the global refcounts of the two volumes are 1
// 12. Remove the both containers
// 14. Verify the global refcounts of the two volumes are back to 0
// 15. Verify status of both volumes are detached
// 16. Remove two volumes
// 17. Reset swarm roles
func (s *VFileDemotePromoteTestSuite) TestSwarmRoleChange(c *C) {
	misc.LogTestStart(c.TestName())

	out, err := dockercli.CreateVFileVolume(s.worker1, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	accessible := verification.CheckVolumeAvailability(s.master, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName2))

	out, err = dockercli.AttachVFileVolume(s.worker1, s.volName1, s.container1Name)
	c.Assert(err, IsNil, Commentf(out))

	err = dockercli.PromoteNode(s.master, s.worker1)
	c.Assert(err, IsNil, Commentf("Failed to promote worker1 %s to manager", s.worker1))

	log.Printf("Wait 20 seconds for new manager to be updated")
	time.Sleep(20 * time.Second)

	err = dockercli.DemoteNode(s.worker1, s.master)
	c.Assert(err, IsNil, Commentf("Failed to demote manager %s", s.master))

	log.Printf("Wait 20 seconds for new worker to be updated")
	time.Sleep(20 * time.Second)

	out, err = dockercli.CreateVFileVolume(s.master, s.volName2)
	c.Assert(err, IsNil, Commentf(out))

	accessible = verification.CheckVolumeAvailability(s.worker1, s.volName2)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

	out, err = dockercli.AttachVFileVolume(s.master, s.volName2, s.container2Name)
	c.Assert(err, IsNil, Commentf(out))

	out = verification.GetVFileVolumeGlobalRefcount(s.volName1, s.master)
	grefc, _ := strconv.Atoi(out)
	c.Assert(grefc, Equals, 1, Commentf("Expected volume %s global refcount to be 1, found %s", s.volName1, out))

	out = verification.GetVFileVolumeGlobalRefcount(s.volName2, s.worker1)
	grefc, _ = strconv.Atoi(out)
	c.Assert(grefc, Equals, 1, Commentf("Expected volume %s global refcount to be 1, found %s", s.volName2, out))

	out, err = dockercli.RemoveContainer(s.worker1, s.container1Name)
	c.Assert(err, IsNil, Commentf(out))

	out = verification.GetVFileVolumeGlobalRefcount(s.volName1, s.master)
	grefc, _ = strconv.Atoi(out)
	c.Assert(grefc, Equals, 0, Commentf("Expected volume %s global refcount to be 0, found %s", s.volName1, out))

	out, err = dockercli.RemoveContainer(s.master, s.container2Name)
	c.Assert(err, IsNil, Commentf(out))

	out = verification.GetVFileVolumeGlobalRefcount(s.volName2, s.worker1)
	grefc, _ = strconv.Atoi(out)
	c.Assert(grefc, Equals, 0, Commentf("Expected volume %s global refcount to be 0, found %s", s.volName2, out))

	log.Printf("Wait 20 seconds for volume status back to Ready")
	time.Sleep(20 * time.Second)

	out = verification.GetVFileVolumeStatusHost(s.volName1, s.master)
	log.Println("GetVFileVolumeStatusHost return out[%s] for volume %s", out, s.volName1)
	c.Assert(out, Equals, "Ready", Commentf("Volume %s status is expected to be [Ready], actual status is [%s]",
		s.volName1, out))

	out = verification.GetVFileVolumeStatusHost(s.volName2, s.worker1)
	log.Println("GetVFileVolumeStatusHost return out[%s] for volume %s", out, s.volName2)
	c.Assert(out, Equals, "Ready", Commentf("Volume %s status is expected to be [Ready], actual status is [%s]",
		s.volName2, out))

	accessible = verification.CheckVolumeAvailability(s.master, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

	accessible = verification.CheckVolumeAvailability(s.worker1, s.volName2)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName2))

	out, err = dockercli.DeleteVolume(s.master, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.DeleteVolume(s.worker1, s.volName2)
	c.Assert(err, IsNil, Commentf(out))

	log.Println("Finished swarm promote/demote test for vFile, start to reset the testbed swarm roles...")
	err = dockercli.PromoteNode(s.worker1, s.master)
	c.Assert(err, IsNil, Commentf("Failed to reset manager role for %s ", s.master))

	log.Printf("Wait 20 seconds for original manager to be updated")
	time.Sleep(20 * time.Second)

	err = dockercli.DemoteNode(s.master, s.worker1)
	c.Assert(err, IsNil, Commentf("Failed to reset worker role for %s", s.worker1))

	log.Printf("Wait 20 seconds for original worker to be updated")
	time.Sleep(20 * time.Second)

	misc.LogTestEnd(c.TestName())
}
