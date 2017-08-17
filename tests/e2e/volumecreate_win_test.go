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

// Windows specific options for volume creation tests.

// +build runoncewin

package e2e

import "github.com/vmware/vsphere-storage-for-docker/tests/utils/inputparams"

var (
	// invalidVolNameList is a slice of volume names for the TestInvalidName test.
	// 1. having more than 100 chars
	// 2. ending -NNNNNN (6Ns)
	// 3. contains @invalid datastore name
	// 4. having various chars including alphanumerics
	invalidVolNameList = []string{
		inputparams.GetVolumeNameOfSize(101),
		"volume-000000",
		inputparams.GetUniqueVolumeName("volume") + "@invaliddatastore",
		"volume-0000000-****-###",
		"volume\\abc",
	}

	// validFstype is a valid fstype for the TestValidOptions test.
	validFstype = "ntfs"
)

// validVolNames returns a slice of volume names for the TestValidName test.
// 1. having 100 chars
// 2. ending in 5Ns
// 3. ending in 7Ns
// 4. contains @datastore (valid name)
// 5. contains multiple '@'
// 6. contains unicode character
// 7. contains space
func (s *VolumeCreateTestSuite) validVolNames() []string {
	return []string{
		inputparams.GetVolumeNameOfSize(100),
		"volume-00000",
		"volume-0000000",
		inputparams.GetUniqueVolumeName("abc") + "@" + s.config.Datastores[0],
		inputparams.GetUniqueVolumeName("abc") + "@@@@" + s.config.Datastores[0],
		inputparams.GetUniqueVolumeName("volume-你"),
		"\"volume space\"",
	}
}
