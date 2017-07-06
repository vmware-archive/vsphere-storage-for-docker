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

// This util holds various helper methods related to vmgroup to be consumed by testcases.
// vmgroup creation, deletion or adding, removing and replacing vm from vmgroup is supported currently.

package admincli

import (
	"log"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// CreateVMgroup method is going to create a vmgroup and adds vm to it.
func CreateVMgroup(ip, name, vmName, dsName string) (string, error) {
	log.Printf("Creating a vmgroup [%s] on esx [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, admincli.CreateVMgroup+name+" --default-datastore="+dsName+admincli.VMlist+vmName)
}

// DeleteVMgroup method deletes a vmgroup and removes its volumes as well if "delete_vol" is set
func DeleteVMgroup(ip, name string, delete_vol bool) (string, error) {
	log.Printf("Deleting a vmgroup [%s] on esx [%s] with delete_vol[%d]\n", name, ip, delete_vol)
	if delete_vol {
		return ssh.InvokeCommand(ip, admincli.RemoveVMgroup+name+admincli.RemoveVolumes)
	} else {
		return ssh.InvokeCommand(ip, admincli.RemoveVMgroup+name)
	}
}

// AddVMToVMgroup - Adds vm to vmgroup
func AddVMToVMgroup(ip, name, vmName string) (string, error) {
	log.Printf("Adding VM [%s] to a vmgroup [%s] on esx [%s] \n", vmName, name, ip)
	return ssh.InvokeCommand(ip, admincli.AddVMToVMgroup+name+admincli.VMlist+vmName)
}

// RemoveVMFromVMgroup - Removes a vm from vmgroup
func RemoveVMFromVMgroup(ip, name, vmName string) (string, error) {
	log.Printf("Removing VM [%s] from a vmgroup [%s] on esx [%s] \n", vmName, name, ip)
	return ssh.InvokeCommand(ip, admincli.RemoveVMFromVMgroup+name+admincli.VMlist+vmName)
}

// ReplaceVMFromVMgroup - Replaces a vm from vmgroup
func ReplaceVMFromVMgroup(ip, name, vmName string) (string, error) {
	log.Printf("Replacing VM [%s] from a vmgroup [%s] on esx [%s] \n", vmName, name, ip)
	return ssh.InvokeCommand(ip, admincli.ReplaceVMFromVMgroup+name+admincli.VMlist+vmName)
}

// AddCreateAccessForVMgroup - add allow-create access on the vmgroup
func AddCreateAccessForVMgroup(ip, name, datastore string) (string, error) {
	log.Printf("Creating create access for vmgroup %s, datastore %s on esx [%s] \n", name, datastore, ip)
	return ssh.InvokeCommand(ip, admincli.AddAccessForVMgroup+name+" --allow-create --datastore "+datastore)
}

// SetCreateAccessForVMgroup - set allow-create access on the vmgroup
func SetCreateAccessForVMgroup(ip, name, datastore string) (string, error) {
	log.Printf("Enabling create access for vmgroup %s, datastore %s on esx [%s] \n", name, datastore, ip)
	return ssh.InvokeCommand(ip, admincli.SetAccessForVMgroup+name+" --allow-create True --datastore "+datastore)
}

// RemoveCreateAccessForVMgroup - remove allow-create access on the vmgroup
func RemoveCreateAccessForVMgroup(ip, name, datastore string) (string, error) {
	log.Printf("Removing create access for vmgroup %s, datastore %s on esx [%s] \n", name, datastore, ip)
	return ssh.InvokeCommand(ip, admincli.SetAccessForVMgroup+name+" --allow-create False --datastore "+datastore)
}

// SetVolumeSizeForVMgroup - set max and total volume size for vmgroup
func SetVolumeSizeForVMgroup(ip, name, ds, msize, tsize string) (string, error) {
	log.Printf("Setting max %s and total %s for vmgroup %s, datastore %s on esx [%s] \n", msize, tsize, name, ds, ip)
	cmd := admincli.SetAccessForVMgroup + name + " --datastore " + ds + " --volume-maxsize=" + msize + " --volume-totalsize=" + tsize + " --allow-create True"
	return ssh.InvokeCommand(ip, cmd)
}

// ConfigInit - Initialize the (local) Single Node Config DB
func ConfigInit(ip string) (string, error) {
	dbMode := GetDBmode(ip)
	if dbMode != admincli.DBNotConfigured {
		log.Printf("DB is already configured on esx [%s]. Removing the DB...\n", ip)
		ConfigRemove(ip) // Ignore the error
	}

	log.Printf("Initializing the SingleNode Config DB on esx [%s] \n", ip)
	return ssh.InvokeCommand(ip, admincli.InitLocalConfigDb)
}

// ConfigRemove - Remove the (local) Single Node Config DB
func ConfigRemove(ip string) (string, error) {
	log.Printf("Removing the SingleNode Config DB on esx [%s] \n", ip)
	return ssh.InvokeCommand(ip, admincli.RemoveLocalConfigDb)
}

// IsVmgroupPresent - checks for a vmgroup in list of vmgroups
func IsVmgroupPresent(esxIP, vmgroupName string) bool {
	log.Printf("Checking if vmgroup [%s] exists from esx [%s] \n", vmgroupName, esxIP)
	cmd := admincli.ListVMgroups + "2>/dev/null | grep " + vmgroupName
	out, _ := ssh.InvokeCommand(esxIP, cmd)
	return strings.Contains(out, vmgroupName)
}

// IsVolumeExistInVmgroup returns true if the given volume is available
// from the specified vmgroup; false otherwise.
func IsVolumeExistInVmgroup(esxIP, vmgroupName, volName string) bool {
	return IsVolumeListExistInVmgroup(esxIP, vmgroupName, []string{volName})
}

// IsVolumeListExistInVmgroup returns true if the given volumes specified in list are
// from the specified vmgroup; false otherwise.
func IsVolumeListExistInVmgroup(esxIP, vmgroupName string, volList []string) bool {
	log.Printf("Checking if volumes [%s] belongs to vmgroup [%s] from esx [%s] \n", volList, vmgroupName, esxIP)
	cmd := admincli.ListVolumes + " -c volume,vmgroup 2>/dev/null | awk -v OFS='\t' '{print $1, $2}' | sed '1,2d' "
	out, _ := ssh.InvokeCommand(esxIP, cmd)

	vgValues := strings.Fields(out)
	VolumeToVgMap := make(map[string]string)
	for i := 0; i < len(vgValues); {
		VolumeToVgMap[vgValues[i]] = vgValues[i+1]
		i = i + 2
	}
	for _, volName := range volList {
		if val, ok := VolumeToVgMap[volName]; !ok || val != vmgroupName {
			log.Printf("Volume %s does not belongs to vmgroup - %s", volName, vmgroupName)
			return false
		}
	}
	return true
}

// IsVMInVmgroup - verify vm's vmgroup is same as expected
func IsVMInVmgroup(esxIP, vmName, vmgroupName string) bool {
	log.Printf("Checking if vm [%s] belongs to vmgroup [%s] from esx [%s] \n", vmName, vmgroupName, esxIP)
	cmd := admincli.ListVmgroupVMs + vmgroupName + " 2>/dev/null | cut -d ' ' -f3 "
	out, _ := ssh.InvokeCommand(esxIP, cmd)
	return strings.Contains(out, vmName)
}

// AddDatastoreToVmgroup - Grants datastore to a vmgroup whose access is controlled
func AddDatastoreToVmgroup(ip, name, datastore string) (string, error) {
	log.Printf("Adding datastore %s to  vmgroup %s,  on esx [%s] \n", datastore, name, ip)
	return ssh.InvokeCommand(ip, admincli.AddAccessForVMgroup+name+" --datastore="+datastore)
}

// IsDSAccessibleForVMgroup - Verifies if vmgroup has access rights to a datastore
func IsDSAccessibleForVMgroup(ip, name, datastore string) bool {
	log.Printf("Getting vmgroup's [%s] access rights to datastore %s ,  on esx [%s] \n", name, datastore, ip)
	cmd := admincli.GetAccessForVMgroup + name + " 2>/dev/null | awk -v OFS='\t' '{print $1, $2}' | grep " + datastore
	out, _ := ssh.InvokeCommand(ip, cmd)
	accessRights := strings.Fields(out)
	log.Printf("Access rights to datastore %s for vmgroup - %s are: %s . ", datastore, name, accessRights)
	if len(accessRights) > 1 && accessRights[1] == "True" {
		return true
	}
	return false

}

// CreateDefaultVmgroup - Create the default Vmgroup
func CreateDefaultVmgroup(ip string) bool {
	log.Printf("Creating default vmgroup %s on esx [%s] ", admincli.DefaultVMgroup, ip)
	cmd := admincli.CreateVMgroup + admincli.DefaultVMgroup + " --default-datastore " + admincli.VMHomeDatastore

	// Create a default vmgroup with default datastore set to _VM_DS
	ssh.InvokeCommand(ip, cmd)

	// Verify default vmgroup exists
	isVmgroupAvailable := IsVmgroupPresent(ip, admincli.DefaultVMgroup)
	if !isVmgroupAvailable {
		log.Println("Failed to create default vmgroup")
		return false
	}

	// Set datastore to _ALL_DS to allow access to all datastore for default vmgroup
	ssh.InvokeCommand(ip, admincli.AddAccessForVMgroup+admincli.DefaultVMgroup+" --datastore="+admincli.AllDatastore+" --allow-create ")
	return IsDSAccessibleForVMgroup(ip, admincli.DefaultVMgroup, admincli.AllDatastore)
}

// GetDBmode - Returns current DB mode
func GetDBmode(esxIP string) string {
	log.Printf("Get db mode from esx [%s] \n", esxIP)
	cmd := admincli.GetDBMode + "| grep DB_Mode"
	out, _ := ssh.InvokeCommand(esxIP, cmd)
	if strings.Contains(out, admincli.DBNotConfigured) {
		return admincli.DBNotConfigured
	} else if strings.Contains(out, admincli.DBSingleNode) {
		return admincli.DBSingleNode
	} else {
		return ""
	}
}
