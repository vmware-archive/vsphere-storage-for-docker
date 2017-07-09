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

// This test suite includes test cases involving restart docker and plugin. #1252
// End goal is to make sure volumes get detached smoothly in different scenarios

// +build runonce

package e2e

import (
	. "gopkg.in/check.v1"
	"strings"
	"log"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	adminconst "github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	adminutil "github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
)

type RestartTestData struct {
	config        *inputparams.TestConfig
	volumeName    string
	containerNameList []string
}

var ds2 string

func (s *RestartTestData) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil || len(s.config.Datastores) < 2 {
		log.Printf("Restart tests setup skipped - missing or incomplete config")
		c.Skip("Restart tests setup skipped - missing or incomplete config")
	}

	adminutil.ConfigInit(s.config.EsxHost)

	s.volumeName = inputparams.GetUniqueVolumeName("restart_test")
	s.containerNameList = []string{inputparams.GetUniqueContainerName("restart_test"),
				inputparams.GetUniqueContainerName("restart_test"),
				inputparams.GetUniqueContainerName("restart_test")}

	// Create a volume
	out, err := dockercli.CreateVolume(s.config.DockerHosts[1], s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	// Get volume status
	status, err := dockercli.GetVolumeStatus(s.config.DockerHosts[1], s.volumeName)
	c.Assert(err, IsNil, Commentf("Failed to fetch status for volume %s", s.volumeName))
	
	ds2 = s.config.Datastores[0]
	if strings.Compare(status["datastore"], s.config.Datastores[0]) == 0 {
		ds2 = s.config.Datastores[1]
	}

	// Add access for the second datastore to the VM's vmgroup with create access
	out, err = adminutil.AddCreateAccessForVMgroup(s.config.EsxHost, adminconst.DefaultVMgroup, ds2)
	c.Assert(err, IsNil, Commentf(out))

	// Create a volume on the second datastore
	out, err = dockercli.CreateVolume(s.config.DockerHosts[1], s.volumeName + "@" + ds2)
	c.Assert(err, IsNil, Commentf(out))

	log.Printf("Restart tests setup complete")
}

func (s *RestartTestData) TearDownSuite(c *C) {
	// Remove any containers that may have been created
	dockercli.RemoveContainer(s.config.DockerHosts[1], strings.Join(s.containerNameList, " "))

	// Ensure there are no remanants from an earlier run
	if s.config != nil {
		out, err := dockercli.DeleteVolume(s.config.DockerHosts[1], s.volumeName)
		c.Assert(err, IsNil, Commentf(out))

		// Remove access for second datastore from the default vmgroup
		out, err = adminutil.RemoveDatastoreFromVmgroup(s.config.EsxHost, adminconst.DefaultVMgroup, s.config.Datastores[1])
		c.Assert(err, IsNil, Commentf(out))
	}

	adminutil.ConfigRemove(s.config.EsxHost)

	log.Printf("Restart tests teardown complete")
}

func (s *RestartTestData) TearDownTest(c *C) {
}

var _ = Suite(&RestartTestData{})

// TestVolumeDetached - verifies a volume is detached when the docker
// daemon is restarted.
// 1. Attach a volume to a container on a host
// 2. Verify attached status
// 3. Restart docker
// 4. Verify detached status. Volume should be detached (within the timeout)
// 5. Verify a container can be started to use the same volume on another host
// 6. Restart the docker daemon on the other host
// 7. Verify detached status for the volume
// 8. Verify a container can be started to use the same volume on the original host
func (s *RestartTestData) TestVolumeDetached(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Attach a volume to a container on a host
	out, err := dockercli.AttachVolume(s.config.DockerHosts[1], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 2. Verify attached status
	status := verification.VerifyAttachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// 3. Restart docker
	out, err = dockercli.RestartDocker(s.config.DockerHosts[1])
	c.Assert(err, IsNil, Commentf(out))

	// 4. Verify detached status. Volume should be detached (within the timeout)
	status = verification.VerifyDetachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	out, err = dockercli.RemoveContainer(s.config.DockerHosts[1], s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 5. Verify a container can be started to use the same volume on another host
	out, err = dockercli.AttachVolume(s.config.DockerHosts[0], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 6. Restart the docker daemon on the other host
	out, err = dockercli.RestartDocker(s.config.DockerHosts[0])
	c.Assert(err, IsNil, Commentf(out))

	// Clean up the container on host0
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[0], s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 7. Verify detached status for the volume
	status = verification.VerifyDetachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	// 8. Verify a container can be started to use the same volume on the original host
	out, err = dockercli.ExecContainer(s.config.DockerHosts[1], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// TestPluginKill - Test vDVS plugin kill
// 1. Attach a volume with restart=always flag
// 2. Verify volume attached status
// 3. Kill vDVS plugin
// 4. Stop the container
// 5. Start the same container
// 6. Stop the container
// 7. Verify volume detached status
func (s *RestartTestData) TestPluginKill(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Attach a volume with restart=always flag
	out, err := dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 2. Verify volume attached status
	status := verification.VerifyAttachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// 3. Kill vDVS plugin
	out, err = dockercli.KillVDVSPlugin(s.config.DockerHosts[1])
	c.Assert(err, IsNil, Commentf(out))

	// 4. Stop the container
	out, err = dockercli.StopContainer(s.config.DockerHosts[1], s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 5. Start the same container
	out, err = dockercli.StartContainer(s.config.DockerHosts[1], s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 6. Stop the container
	out, err = dockercli.StopContainer(s.config.DockerHosts[1], s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 7. Verify volume detached status
	status = verification.VerifyDetachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	// Cleanup container
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[1], s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// TestRecoverMountsAfterRestart  - Verify that volume mounts are recovered after
// docker is restarted and volume is usable by any VM.
// 1. Start 2 containers using same volume with restart=always
// 2. Restart docker and wait for containers to start and reference counts to be initialized.
// 3. Verify volume is attached to the VM
// 4. Start 1 container using same volume
// 5. Stop all 3 containers and confirm disk is detached
// 6. Run container on other host to verify the volume is detached after stopping the containers
func (s *RestartTestData) TestRecoverMountsAfterRestart(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Run containers with restart-always flag
	out, err := dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// Run second container 
	out, err = dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerNameList[1])
	c.Assert(err, IsNil, Commentf(out))

	// 2. Restart docker
	out, err = dockercli.RestartDocker(s.config.DockerHosts[1])
	c.Assert(err, IsNil, Commentf(out))

	misc.SleepForSec(20)

	// 3. Verify the volume is attached to the VM
	status := verification.VerifyAttachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// 4. Run third container
	out, err = dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerNameList[2])
	c.Assert(err, IsNil, Commentf(out))

	// 5. Stop the three containers
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[1], strings.Join(s.containerNameList, " "))
	c.Assert(err, IsNil, Commentf(out))

	status = verification.VerifyDetachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	// 6. Run container on other host to verify the volume is detached after stopping the containers
	out, err = dockercli.ExecContainer(s.config.DockerHosts[0], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// TestLongVolumeName - Verifies volumes can be created on the default
// datastore by long and short names.
// 1. Start container with restart flag and use short name for volume (vol1)
// 2. Restart docker
// 3. Start another instance of container with short name for volume (vol1)
// 4. Get the volume properties and figure the datastore of the volume
// 4. Start another instance of container with long name for volume (vol1@<datastore-name>)
// 5. Stop all 3 containers and confirm disk is detached (verify by exec'ing a container on another host).
func (s *RestartTestData) TestLongVolumeName(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Run container with restart-always flag
	out, err := dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 2. Restart docker
	out, err = dockercli.RestartDocker(s.config.DockerHosts[1])
	c.Assert(err, IsNil, Commentf(out))

	misc.SleepForSec(20)

	// 3. Run second container 
	out, err = dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerNameList[1])
	c.Assert(err, IsNil, Commentf(out))

	// 3. Get volume status
	status, err := dockercli.GetVolumeStatus(s.config.DockerHosts[1], s.volumeName)
	c.Assert(err, IsNil, Commentf("Failed to fetch status for volume %s", s.volumeName))

	// 4. Run third container with long volume name
	out, err = dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName + "@" + status["datastore"], s.containerNameList[2])
	c.Assert(err, IsNil, Commentf(out))

	// 5. Stop the three containers
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[1], strings.Join(s.containerNameList, " "))
	c.Assert(err, IsNil, Commentf(out))

	// 6. Run container on other host to verify the volume is detached after stopping the containers
	out, err = dockercli.ExecContainer(s.config.DockerHosts[0], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))
	misc.LogTestEnd(c.TestName())
}

// TestDuplicateVolumeName - Verifies volumes with same same can be created on the default
// datastore and non-default datastores
// 1. Start container with restart flag and short name "vol1" (assuming vol1@ds1)
// 2. Start container with restart flag with same volume name on another ds: vol1@ds2
// 3. Restart docker
// 4. Start container with long name: vol1@ds1
// 5. Stop all 3 containers and make sure both vol1@ds1 ad vol1@ds2 are detached.
func (s *RestartTestData) TestDuplicateVolumeName(c *C) {
	misc.LogTestStart(c.TestName())

	// 1. Run container with restart-always flag
	out, err := dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	// 2. Run second container with same volume name on the other datastore 
	out, err = dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName + "@" + ds2, s.containerNameList[1])
	c.Assert(err, IsNil, Commentf(out))

	status, err := dockercli.GetVolumeStatus(s.config.DockerHosts[1], s.volumeName)
	c.Assert(err, IsNil, Commentf("Failed to fetch status for volume %s", s.volumeName))

	// 3. Restart docker
	out, err = dockercli.RestartDocker(s.config.DockerHosts[1])
	c.Assert(err, IsNil, Commentf(out))

	misc.SleepForSec(20)

	// 4. Run third container with same volume as the first container
	out, err = dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName + "@" + status["datastore"], s.containerNameList[2])
	c.Assert(err, IsNil, Commentf(out))

	// 5. Stop the three containers
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[1], strings.Join(s.containerNameList, " "))
	c.Assert(err, IsNil, Commentf(out))

	// Cleanup volume on the second datastore
	out, err = dockercli.DeleteVolume(s.config.DockerHosts[1], s.volumeName + "@" + ds2)
	c.Assert(err, IsNil, Commentf(out))

	// 6. Run container on other host to verify the volume is detached after stopping the containers
	out, err = dockercli.ExecContainer(s.config.DockerHosts[0], s.volumeName, s.containerNameList[0])
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}
