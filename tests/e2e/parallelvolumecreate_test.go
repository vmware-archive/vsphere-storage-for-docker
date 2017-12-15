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

// This test creates volumes in parallel as per the noOfVolumes parameter
// value. This is added as to test performace impact while creating volumes
// in parallel.

// +build runoncewin

package e2e

import (
	"fmt"
	"strings"
	"sync"

	"github.com/vmware/vsphere-storage-for-docker/tests/utils/dockercli"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/inputparams"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/misc"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/verification"

	. "gopkg.in/check.v1"
)

type ParallelVolumeCreateTestSuite struct {
	config     *inputparams.TestConfig
	volumeList []string
}

func (s *ParallelVolumeCreateTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping parallel volume create test")
	}
}

func (s *ParallelVolumeCreateTestSuite) TearDownTest(c *C) {
	volList := strings.Join(s.volumeList, " ")

	if volList != "" {
		out, err := dockercli.DeleteVolume(s.config.DockerHosts[0], volList)
		c.Assert(err, IsNil, Commentf(out))
	}
}

var _ = Suite(&ParallelVolumeCreateTestSuite{})

// create volume and verifies creation
func (s *ParallelVolumeCreateTestSuite) createVolCheck(name string, c *C) {
	var out string
	var err error

	out, err = dockercli.CreateVolume(s.config.DockerHosts[0], name)

	// if creation is successful, add it to the list so that it gets cleaned later
	if err == nil {
		s.volumeList = append(s.volumeList, name)
	}

	c.Assert(err, IsNil, Commentf(out))
}

// parallely create volumes noOfVolumes times
func (s *ParallelVolumeCreateTestSuite) parallelVolumeCreation(noOfVolumes int, c *C) {
	var wg sync.WaitGroup
	for i := 0; i < noOfVolumes; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			volName := inputparams.GetUniqueVolumeName(fmt.Sprintf("%s%d", "pVol_", i))
			s.createVolCheck(volName, c)
		}(i)
	}
	wg.Wait()
}

func (s *ParallelVolumeCreateTestSuite) accessCheck(hostIP string, volList []string, c *C) {
	isAvailable := verification.CheckVolumeListAvailability(hostIP, volList)
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", volList))
}

// Create volume parallely
func (s *ParallelVolumeCreateTestSuite) TestVolumeCreationParallel(c *C) {
	misc.LogTestStart(c.TestName())

	// change this parameter to create as many volume in parallel
	const noOfVolumes = 10

	s.parallelVolumeCreation(noOfVolumes, c)
	s.accessCheck(s.config.DockerHosts[0], s.volumeList, c)

	misc.LogTestEnd(c.TestName())
}
