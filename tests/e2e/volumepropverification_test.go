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

// This is an end-to-end test. Test will ssh to a vm and create a volume using docker cli.
// After creating a volume, test will ssh to the ESX and verify the properties of the volume.
// Properties being verified - capacity, disk-format and vm-attached field.

// Test assumes that SSH cert has been setup to enable password-less login to VM and ESX.

package e2e

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

var (
	esx              = os.Getenv("ESX")
	volSizes         = []string{"100MB"}
	vms              = []string{os.Getenv("VM1")}
	formatTypes      = []string{"thin", "zeroedthick", "eagerzeroedthick"}
	vmIP             string
	dockerVolmRmvCmd = "docker volume rm "
	dockerCliCheck   bool
	containerName    string
)

func TestMain(m *testing.M) {
	retCode := m.Run()
	teardownFunction()
	os.Exit(retCode)
}

// clean-up function
func teardownFunction() {
	verification.ExecCmd(vmIP, dockerVolmRmvCmd)
	log.Println("-----Clean-up finished - current time: ", time.Now())
}

/*
Test steps:
1. SSH to a vm and create a volume of size 100 mb and specified disk format ["thin", "zeroedthick", "eagerzeroedthick"]
2. Do docker inspect on the volume and verify size, disk format and attached to vm field.
   Expected value: {"100MB", "thin/zeroedthick/eagerzeroedthick", no-value} respectively.
3. SSH to the esx and verify the capacity, disk format and attached-to-vm field for the volume using Admin cli
   Expected value: {"100MB", "thin/zeroedthick/eagerzeroedthick", "detached"} respectively.
4. SSH to the vm and run a container and mount the volume created
5. SSH to the vm and esx and verify the attached-to-vm field for volume - both docker cli and admin cli values should be same.
6. Again verify capacity and disk format using docker cli and admin cli to make sure things are fine after running the container.
*/
func TestVolumeProperties(t *testing.T) {
	for vmIndx := 0; vmIndx < len(vms); vmIndx++ {
		log.Println("running end_to_end tests - current time: ", time.Now())
		vmIP = vms[vmIndx]
		log.Println("Running test on VM - ", vmIP)
		dockerCliCheck = verification.IsDockerCliCheckNeeded(vms[vmIndx])
		for i := 0; i < len(volSizes); i++ {
			for k := 0; k < len(formatTypes); k++ {
				containerName = inputparams.GetContainerNameWithTimeStamp("volumeprop_test")
				log.Println("Creating a volume of Format Type - ", formatTypes[k])
				volName := inputparams.GetVolumeNameWithTimeStamp("volumeprop_test")
				_, err := ssh.InvokeCommand(vms[vmIndx], "docker volume create --driver=vsphere --name="+
					volName+" -o size="+volSizes[i]+" -o diskformat="+formatTypes[k])
				if err != nil {
					log.Fatalf("Failed to create a volume named: %s. Error - %v ", volName, err)
				} else {
					log.Println("\n successfully created a  volume -- ", volName)
					dockerVolmRmvCmd = dockerVolmRmvCmd + " " + volName
				}
				log.Println("Verifying volume properties like size, disk-format and attached-to-vm fields "+
					"at vm and esx for volume - ", volName)
				volmPropertiesAdminCli := verification.GetVolumePropertiesAdminCli(volName, esx)
				expctdPropsAdmin := []string{"100MB", formatTypes[k], "detached"}
				if !hasElement(expctdPropsAdmin, volmPropertiesAdminCli) {
					log.Fatal("Volume properties on ESX fetched using admin cli does not matches with the expected values")
				}

				if dockerCliCheck {
					volumePropertiesDockerCli := verification.GetVolumePropertiesDockerCli(volName, vms[vmIndx])
					expctdPropsDkr := []string{"100MB", formatTypes[k], "<no value>"}
					if !hasElement(expctdPropsDkr, volumePropertiesDockerCli) {
						log.Fatal("Volume properties fetched using docker cli do not matches with the expected values")
					}
				}
				ssh.InvokeCommand(vms[vmIndx], "docker run -d -v "+volName+":/vol --name "+containerName+" busybox tail -f /dev/null")
				// Verifying attached to the vm field for volume
				vmNameFrmAdminCli := verification.GetVMAttachedToVolUsingAdminCli(volName, esx)
				if dockerCliCheck {
					vmNameFrmDockerCli := verification.GetVMAttachedToVolUsingDockerCli(volName, vms[vmIndx])
					// TODO: get vm name based on ip and compare ith with the docker cli and admin cli
					if vmNameFrmDockerCli != vmNameFrmAdminCli {
						log.Fatalf("Information mis-match - Attached-to-VM field for volume from docker cli is [%s]"+
							"and attched-to-vm field from admin cli is [%s]", vmNameFrmDockerCli, vmNameFrmAdminCli)
					}
					volumePropertiesDockerCli := verification.GetVolumePropertiesDockerCli(volName, vms[vmIndx])
					expctdPropsDkr := []string{"100MB", formatTypes[k]}
					if !hasElement(expctdPropsDkr, volumePropertiesDockerCli) {
						log.Fatal("Volume properties on ESX fetched using docker cli do not matches with the expected values")
					}
				}
				volmPropertiesAdminCli = verification.GetVolumePropertiesAdminCli(volName, esx)
				expctdPropsAdmin = []string{"100MB", formatTypes[k]}
				if !hasElement(expctdPropsAdmin, volmPropertiesAdminCli) {
					log.Fatal("Volume properties on admin cli do not matches with the expected values")
				}
				log.Println("Finished verifying volume properties like size, disk-format and attached-to-vm fields"+
					" at vm and esx for volume - ", volName)
				verification.ExecCmd(vms[vmIndx], "docker stop "+containerName+" ; docker rm "+containerName)
			}
		}
	}
}

// method checks if the output from the ssh commands contains the expected values
func hasElement(volmProps []string, op string) bool {
	for i := 0; i < len(volmProps); i++ {
		if !strings.Contains(op, volmProps[i]) {
			log.Printf("Actual values : %s. Does not contains %s ", op, volmProps[i])
			return false
		}
	}
	return true
}
