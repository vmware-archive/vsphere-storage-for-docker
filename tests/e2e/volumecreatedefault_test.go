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

// Options and additional volume creation tests for non-windows platforms.

// +build runonce

package e2e

import (
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"

	. "gopkg.in/check.v1"
)

var (
	// invalidVolNameList is a slice of volume names for the TestInvalidName test.
	// 1. having more than 100 chars
	// 2. ending -NNNNNN (6Ns)
	// 3. contains @invalid datastore name
	invalidVolNameList = []string{
		inputparams.GetVolumeNameOfSize(101),
		"Volume-000000",
		inputparams.GetUniqueVolumeName("Volume") + "@invalidDatastore",
	}

	// validFstype is a valid fstype for the TestValidOptions test.
	validFstype = "ext4"
)

// validVolNames returns a slice of volume names for the TestValidName test.
// 1. having 100 chars
// 2. having various chars including alphanumerics
// 3. ending in 5Ns
// 4. ending in 7Ns
// 5. contains @datastore (valid name)
// 6. contains multiple '@'
// 7. contains unicode character
// 8. contains space
func (s *VolumeCreateTestSuite) validVolNames() []string {
	return []string{
		inputparams.GetVolumeNameOfSize(100),
		"Volume-0000000-****-###",
		"Volume-00000",
		"Volume-0000000",
		inputparams.GetUniqueVolumeName("abc") + "@" + s.config.Datastores[0],
		inputparams.GetUniqueVolumeName("abc") + "@@@@" + s.config.Datastores[0],
		inputparams.GetUniqueVolumeName("Volume-你"),
		"\"Volume Space\"",
	}
}

// TestValidXFSOption tests valid volume creation with fstype xfs.
func (s *VolumeCreateTestSuite) TestValidXFSOption(c *C) {
	misc.LogTestStart(c.TestName())

	// xfs file system needs volume name upto than 12 characters
	xfsVolName := inputparams.GetVolumeNameOfSize(12)
	s.volumeList = append(s.volumeList, xfsVolName)
	out, err := dockercli.CreateVolumeWithOptions(s.config.DockerHosts[0], xfsVolName, " -o fstype=xfs")
	c.Assert(err, IsNil, Commentf(out))
	s.accessCheck(s.config.DockerHosts[0], s.volumeList, c)

	misc.LogTestEnd(c.TestName())
}
