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

// This is an end-to-end test. Test creates volumes of different format-types
// and verifies their properties from ESX as well as docker host.
// Properties being verified - capacity, disk-format and attached-to-vm field.

// Test assumes that SSH cert has been setup to enable password-less login to VM and ESX.

// +build runalways

package e2e

import (
	"log"
	"reflect"
	"strings"

	dockerconst "github.com/vmware/vsphere-storage-for-docker/tests/constants/dockercli"
	"github.com/vmware/vsphere-storage-for-docker/tests/constants/properties"
	admincliconst "github.com/vmware/vsphere-storage-for-docker/tests/utils/admincli"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/dockercli"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/inputparams"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/misc"
	"github.com/vmware/vsphere-storage-for-docker/tests/utils/ssh"
	. "gopkg.in/check.v1"
)

const (
	size             = "100MB"
	diskFormatOption = " -o diskformat="
)

type VolumePropertyTestSuite struct {
	volumeNames   []string
	containerName string
	formatTypes   []string
	volumeStatus  string
	config        *inputparams.TestConfig
}

var _ = Suite(&VolumePropertyTestSuite{})

// Map where string is volume name and properties values fetched from docker host
var dockerCliMap = make(map[string][]string)

// Map where string is volume name and properties values fetched from esx
var adminCliMap = make(map[string][]string)

// Map where string is volume name and expected values for different properties
var expectedValuesMap = make(map[string][]string)

func (s *VolumePropertyTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping volumeproperty_test.TestVolumeProperties.")
	}
}

func (s *VolumePropertyTestSuite) SetUpTest(c *C) {
	s.containerName = inputparams.GetUniqueContainerName(c.TestName())
	s.formatTypes = []string{"thin", "zeroedthick", "eagerzeroedthick"}
	s.volumeStatus = properties.DetachedStatus
}

// TearDownSuite - Removes all the containers and deletes the volumes
func (s *VolumePropertyTestSuite) TearDownSuite(c *C) {
	out, err := dockercli.RemoveAllContainers(s.config.DockerHosts[1])
	c.Assert(err, IsNil, Commentf(out))
	out, err = ssh.InvokeCommand(s.config.DockerHosts[1], dockerconst.RemoveVolume+strings.Join(s.volumeNames, " "))
	c.Assert(err, IsNil, Commentf(out))
}

/*
Test steps:
1. SSH to a vm and create volumes of size 100 mb and specified disk format ["thin", "zeroedthick", "eagerzeroedthick"]
2. Do docker inspect on the volume get size, disk format and attached-to-vm field.
   Expected value: {"100MB", "thin/zeroedthick/eagerzeroedthick", no-value} respectively.
3. SSH to the esx and get capacity, disk format and attached-to-vm field for the volume using Admin cli
   Expected value: {"100MB", "thin/zeroedthick/eagerzeroedthick", "detached"} respectively.
4. Verifies values obtained from admin cli and docker cli.
5. SSH to the vm and run a container and mount the volume created.
6. SSH to the vm and esx and verify the attached-to-vm field for volume - both docker cli and admin cli values should be same.
7. Again verify capacity and disk format using docker cli and admin cli to make sure things are fine after running the container.

NOTE: Do steps 5 ,6 and 7 only for volume of 'thin' disk format type.
*/

func (s *VolumePropertyTestSuite) TestVolumeProperties(c *C) {
	misc.LogTestStart(c.TestName())

	// create volumes of all three disk formats
	s.createVolumes(c)

	// get properties for all three volumes from ESX and docker host
	// and adds them to the map
	s.populateVolumePropertiesMaps(c)

	// Verify if docker and ESX properties of all three volumes are same and as expected.
	s.verifyProperties(c)

	// attach only thin volume
	out, err := dockercli.AttachVolume(s.config.DockerHosts[1], s.volumeNames[0], s.containerName)
	c.Assert(err, IsNil, Commentf("Failed to attach the volume [%s]", out))
	s.volumeStatus = properties.AttachedStatus

	// Get volumes properties for the thin volume from docker and ESX.
	s.populateVolPropsFromEsx(c, s.volumeNames[0])
	s.populateVolPropsFromDockerHost(c, s.volumeNames[0])

	// Verify if docker and ESX properties of volumes are same and as expected.
	s.verifyProperties(c)
	misc.LogTestEnd(c.TestName())
}

// createVolumes - creates volumes of each format type
func (s *VolumePropertyTestSuite) createVolumes(c *C) {
	for _, formatType := range s.formatTypes {
		vname := inputparams.GetUniqueVolumeName(formatType)
		out, err := dockercli.CreateVolumeWithOptions(s.config.DockerHosts[1], vname, diskFormatOption+formatType)
		s.volumeNames = append(s.volumeNames, vname)
		c.Assert(err, IsNil, Commentf(out))
	}
}

// populateVolumePropertiesMaps - get properties of three volumes from ESX and docker host
// adds them to dockerCliMap and adminCliMap
func (s *VolumePropertyTestSuite) populateVolumePropertiesMaps(c *C) {
	for _, volumeName := range s.volumeNames {
		s.populateVolPropsFromEsx(c, volumeName)
	}
	for _, volumeName := range s.volumeNames {
		s.populateVolPropsFromDockerHost(c, volumeName)
	}
}

// populateVolPropsFromEsx - gets properties of a volume from the ESX and
// populate admin cli map
func (s *VolumePropertyTestSuite) populateVolPropsFromEsx(c *C, volumeName string) {
	admincliValues := admincliconst.GetVolumeProperties(volumeName, s.config.EsxHost)
	c.Assert(admincliValues, HasLen, 3)
	adminCliMap[volumeName] = []string{admincliValues[0], admincliValues[1], admincliValues[2]}
}

// populateVolPropsFromDockerHost - gets properties of a volume from the docker host and
// populate docker cli map
func (s *VolumePropertyTestSuite) populateVolPropsFromDockerHost(c *C, volumeName string) {
	out, _ := dockercli.GetVolumeProperties(volumeName, s.config.DockerHosts[1])
	dockerCliValues := strings.Fields(out)
	dockerCliMap[volumeName] = []string{dockerCliValues[0], dockerCliValues[1], dockerCliValues[2]}
}

// verifyProperties - Verify docker and ESX properties of all three volumes are same and as expected.
func (s *VolumePropertyTestSuite) verifyProperties(c *C) {
	log.Println("Verify docker and ESX properties of all three volumes are same")
	status := reflect.DeepEqual(dockerCliMap, adminCliMap)
	c.Assert(status, Equals, true, Commentf("Property values from ESX: %s . Property values from Docker Host: %s"+
		"ESX and docker properties of volumes are not same.", adminCliMap, dockerCliMap))

	// Only for volume with 'thin' format type, after s.volumeStatus=properties.AttachedStatus,
	// modify attached-to-vm field with vm name.
	if s.volumeStatus != properties.AttachedStatus {
		for i, volumeName := range s.volumeNames {
			expectedValuesMap[volumeName] = []string{size, s.formatTypes[i], properties.DetachedStatus}
		}
	} else {
		tmp := expectedValuesMap[s.volumeNames[0]]
		tmp[2] = s.config.DockerHostNames[1]
		expectedValuesMap[s.volumeNames[0]] = tmp
	}

	// Verify ESX properties of all three volumes are as expected.
	status = reflect.DeepEqual(expectedValuesMap, adminCliMap)
	c.Assert(status, Equals, true, Commentf("Actual property values from ESX: %s . Expected values: %s ."+
		"ESX properties and expected property values of volumes are not same", adminCliMap, expectedValuesMap))
}
