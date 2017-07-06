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

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
)

type RestartTestData struct {
	config        *inputparams.TestConfig
	volumeName    string
	containerName string
}

func (s *RestartTestData) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping docker/plugin restart tests")
	}

	s.volumeName = inputparams.GetUniqueVolumeName("restart_test")
	s.containerName = inputparams.GetUniqueContainerName("restart_test")

	// ensure there are no remanants from an earlier run
	out, err := dockercli.RemoveContainer(s.config.DockerHosts[1], s.containerName)

	// Create a volume
	out, err = dockercli.CreateVolume(s.config.DockerHosts[1], s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

func (s *RestartTestData) TearDownSuite(c *C) {
	out, err := dockercli.DeleteVolume(s.config.DockerHosts[1], s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
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
	out, err := dockercli.AttachVolume(s.config.DockerHosts[1], s.volumeName, s.containerName)
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

	out, err = dockercli.RemoveContainer(s.config.DockerHosts[1], s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 5. Verify a container can be started to use the same volume on another host
	out, err = dockercli.AttachVolume(s.config.DockerHosts[0], s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 6. Restart the docker daemon on the other host
	out, err = dockercli.RestartDocker(s.config.DockerHosts[0])
	c.Assert(err, IsNil, Commentf(out))

	// Clean up the container on host0
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[0], s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 7. Verify detached status for the volume
	status = verification.VerifyDetachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	// 8. Verify a container can be started to use the same volume on the original host
	out, err = dockercli.ExecContainer(s.config.DockerHosts[1], s.volumeName, s.containerName)
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
	out, err := dockercli.AttachVolumeWithRestart(s.config.DockerHosts[1], s.volumeName, s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 2. Verify volume attached status
	status := verification.VerifyAttachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	// 3. Kill vDVS plugin
	out, err = dockercli.KillVDVSPlugin(s.config.DockerHosts[1])
	c.Assert(err, IsNil, Commentf(out))

	// 4. Stop the container
	out, err = dockercli.StopContainer(s.config.DockerHosts[1], s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 5. Start the same container
	out, err = dockercli.StartContainer(s.config.DockerHosts[1], s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 6. Stop the container
	out, err = dockercli.StopContainer(s.config.DockerHosts[1], s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	// 7. Verify volume detached status
	status = verification.VerifyDetachedStatus(s.volumeName, s.config.DockerHosts[1], s.config.EsxHost)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	// Cleanup container
	out, err = dockercli.RemoveContainer(s.config.DockerHosts[1], s.containerName)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}
