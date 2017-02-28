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

// This test is going to cover the issue reported at #656
// TODO: as of now we are running the test against photon vm it should be run
// against various/applicable linux distros.

package e2e_test

import (
	volumeNameUtil "github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils"
	TestUtil "github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/test_util"
	"os"
	"testing"
)

// Test objective: create volume on fresh setup very first time and verifies the volume
// is created successfully.

/*

The issue was observed on photon so steps are mentioned for photon OS only in fact
test should be OS agnostic.

1. create docker volume using following command
	docker volume create --driver=vsphere --name=testVol -o size=10gb
2. verify volume is created correctly or not
3. delete created volume

expectation: volume should be created correctly

*/

func TestVolumeCreationFirstTime(t *testing.T) {
	var err error
	var out []byte
	volumeName := volumeNameUtil.GetVolumeNameWithTimeStamp("abc")

	// create volume
	out, err = TestUtil.CreateDefaultVolume(os.Getenv("VM1"), volumeName)

	if err != nil {
		t.Fatalf("\nError has occurred [%s] \n\twhile creating volume [%s] very first time: err -> %v", out, volumeName, err)
	} else {
		t.Logf("\nTestcase passed: successfully able to create volume [%s]\n", out)
		// delete volume
		TestUtil.DeleteVolume(volumeName, os.Getenv("VM1"))
	}
}
