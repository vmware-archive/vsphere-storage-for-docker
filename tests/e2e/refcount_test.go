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

// This test suite includes refcount test on plugin side

// +build runonce

package e2e

import (
	"strconv"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

type RefcountTestSuite struct {
	config *inputparams.TestConfig

	volumeName        string
	containerNameList []string
}

func (s *RefcountTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping refcount tests")
	}
}
func (s *RefcountTestSuite) SetUpTest(c *C) {
	s.volumeName = inputparams.GetUniqueVolumeName("RefcountTest")

	// Create a volume
	out, err := dockercli.CreateVolume(s.config.DockerHosts[0], s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

func (s *RefcountTestSuite) TearDownTest(c *C) {
	// If containers are leftover due to test fail, clean here
	cName := strings.Join(s.containerNameList, " ")
	if cName != "" {
		out, err := dockercli.RemoveContainer(s.config.DockerHosts[0], cName)
		c.Assert(err, IsNil, Commentf(out))
	}
}

var _ = Suite(&RefcountTestSuite{})

// Test for refcount recovery
// 1. Create a volume
// 2 Attach 5 containers to it with restart flag
// 3. Verify attached status after each attach
// 4. Check the refcount log in log file
// 5. Verify the content on volume created by containers
// 6. Check the volume entry on mount file : /proc/mounts
// 7. Try to delete volume. Should Fail
// 8. Kill Docker (ungraceful)
// 9. Check the recovery log record in the log file
// 10. Cleanup the containers
// 11. Delete the volume. Should succeed
// 12. There should be no entry in /proc/mounts

func (s *RefcountTestSuite) TestRefcountingRecovery(c *C) {
	misc.LogTestStart(c.TestName())

	// Check if volume is available in docker host
	accessible := verification.CheckVolumeAvailability(s.config.DockerHosts[0], s.volumeName)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available on docker host %s",
		s.volumeName, s.config.DockerHosts[0]))

	// Check if volume is available in docker ESX
	accessible = admincli.IsVolumeAvailableOnESX(s.config.EsxHost, s.volumeName)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available on ESX %s",
		s.volumeName, s.config.DockerHosts[0]))

	numContainers := 5

	// Each container creates a file with its index as the filename suffix. After that it keeps running.
	for i := 0; i < numContainers; i++ {
		cName := inputparams.GetUniqueContainerName(c.TestName())
		out, err := dockercli.WriteToVolumeWithRestart(s.config.DockerHosts[0], s.volumeName, cName, "file_"+strconv.Itoa(i))
		c.Assert(err, IsNil, Commentf(out))

		s.containerNameList = append(s.containerNameList, cName)

		status := verification.VerifyAttachedStatus(s.volumeName, s.config.DockerHosts[0], s.config.EsxHost, s.config.DockerHostNames[0])
		c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))
	}

	// The refcount log should indicate the refcount of volume. It should be equal to number of containers
	isRefcounted := verification.CheckRefCount(s.config.DockerHosts[0], s.volumeName, 1, 5)
	c.Assert(isRefcounted, Equals, true, Commentf("Volume %s refcount error", s.volumeName))

	// start another container from the same VM and verify this container can see all the files created before
	listFiles, err := dockercli.ListFilesOnVolume(s.config.DockerHosts[0], s.volumeName)
	for i := 0; i < numContainers; i++ {
		c.Assert(strings.Contains(listFiles, "file_"+strconv.Itoa(i)), Equals, true,
			Commentf("File %d is not available in %s", listFiles))
	}

	// Verify the entry of volume in mount file
	isAvailable := verification.CheckVolumeInProcMounts(s.config.DockerHosts[0], s.volumeName)
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s unavailable in /proc/mounts", s.volumeName))

	// This delete should fail since the volume is still being used
	out, err := dockercli.DeleteVolume(s.config.DockerHosts[0], s.volumeName)
	c.Assert(err, Not(IsNil), Commentf(out))

	// Kill Docker (ungraceful). It restarts automatically
	out, err = dockercli.KillDocker(s.config.DockerHosts[0])
	c.Assert(err, IsNil, Commentf(out))

	// Verify the recovery record.
	isRefcounted = verification.VerifyRecoveryRecord(s.config.DockerHosts[0], s.volumeName, 10, 5)
	c.Assert(isRefcounted, Equals, true, Commentf("Volume %s refcount error", s.volumeName))

	// Cleanup the containers
	cName := strings.Join(s.containerNameList, " ")
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[0], cName)
	c.Assert(err, IsNil, Commentf(out))
	s.containerNameList = s.containerNameList[:0]

	// This delete should succeed
	out, err = dockercli.DeleteVolume(s.config.DockerHosts[0], s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// No entry in /proc/mounts
	isAvailable = verification.CheckVolumeInProcMounts(s.config.DockerHosts[0], s.volumeName)
	c.Assert(isAvailable, Equals, false, Commentf("Volume %s should not available in /proc/mounts", s.volumeName))

	misc.LogTestEnd(c.TestName())
}
