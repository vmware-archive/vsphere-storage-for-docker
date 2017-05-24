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

	"github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
)

// CreateVMgroup method is going to create a vmgroup and adds vm to it.
func CreateVMgroup(ip, name, vmName string) (string, error) {
	log.Printf("Creating a vmgroup [%s] on esx [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, admincli.CreateVMgroup+name+" --default-datastore="+admincli.VMHomeDatastore+" --vm-list="+vmName)
}

// DeleteVMgroup method deletes a vmgroup and removes its volumes as well
func DeleteVMgroup(ip, name string) (string, error) {
	log.Printf("Deleting a vmgroup [%s] on esx [%s]\n", name, ip)
	return ssh.InvokeCommand(ip, admincli.RemoveVMgroup+name+admincli.RemoveVolumes)
}

// AddVMToVMgroup - Adds vm to vmgroup
func AddVMToVMgroup(ip, name, vmName string) (string, error) {
	log.Printf("Adding VM [%s] to a vmgroup [%s] on esx [%s] \n", vmName, name, ip)
	return ssh.InvokeCommand(ip, admincli.AddVMToVMgroup+name+" --vm-list="+vmName)
}

// RemoveVMFromVMgroup - Removes a vm from vmgroup
func RemoveVMFromVMgroup(ip, name, vmName string) (string, error) {
	log.Printf("Removing VM [%s] from a vmgroup [%s] on esx [%s] \n", vmName, name, ip)
	return ssh.InvokeCommand(ip, admincli.RemoveVMFromVMgroup+name+" --vm-list="+vmName)
}

// ReplaceVMFromVMgroup - Replaces a vm from vmgroup
func ReplaceVMFromVMgroup(ip, name, vmName string) (string, error) {
	log.Printf("Replacing VM [%s] from a vmgroup [%s] on esx [%s] \n", vmName, name, ip)
	return ssh.InvokeCommand(ip, admincli.ReplaceVMFromVMgroup+name+" --vm-list="+vmName)
}

// AddCreateAccessForVMgroup - set allow-create access on the vmgroup
func AddCreateAccessForVMgroup(ip, name, datastore string) (string, error) {
	log.Printf("Enabling create access for vmgroup %s, datastore %s on esx [%s] \n", name, datastore, ip)
	return ssh.InvokeCommand(ip, admincli.SetAccessForVMgroup + name + " --allow-create True --datastore " + datastore)
}

// RemoveCreateAccessForVMgroup - set allow-create access on the vmgroup
func RemoveCreateAccessForVMgroup(ip, name, datastore string) (string, error) {
	log.Printf("Removing create access for vmgroup %s, datastore %s on esx [%s] \n", name, datastore, ip)
	return ssh.InvokeCommand(ip, admincli.SetAccessForVMgroup + name + " --allow-create False --datastore " + datastore)
}

// SetVolumeSizeForVMgroup - set max and total volume size for vmgroup
func SetVolumeSizeForVMgroup(ip, name, ds, msize, tsize string) (string, error) {
	log.Printf("Setting max %s and total %s for vmgroup %s, datastore %s on esx [%s] \n", msize, tsize, name, ds, ip)
	cmd := admincli.SetAccessForVMgroup + name + " --datastore " + ds + " --volume-maxsize=" + msize + " --volume-totalsize=" + tsize + " --allow-create True"
	return ssh.InvokeCommand(ip, cmd)
}

// ConfigInit - Initialize the (local) Single Node Config DB
func ConfigInit(ip string) (string, error) {
	log.Printf("Initializing the SingleNode Config DB on esx [%s] \n", ip)
	return ssh.InvokeCommand(ip, admincli.InitLocalConfigDb)
}

// ConfigRemove - Remove the (local) Single Node Config DB
func ConfigRemove(ip string) (string, error) {
	log.Printf("Removing the SingleNode Config DB on esx [%s] \n", ip)
	return ssh.InvokeCommand(ip, admincli.RemoveLocalConfigDb)
}
