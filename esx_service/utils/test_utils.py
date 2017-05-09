#!/usr/bin/env python
# Copyright 2016 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
'''
Placeholder for utils used in tests
'''

import sys
import logging
import random

import vmdk_ops
import error_code
import auth_api
import auth
import auth_data_const

from pyVmomi import vim

def generate_test_vm_name():
    """ This method gets unique name for vm e.g. vm_name = test-vm_19826"""
    random_id = random.randint(0, 65536)
    vm_name = 'test-vm_' + str(random_id)
    return vm_name


def create_vm(si, vm_name, datastore_name):
    """ Create a VM """
    content = si.RetrieveContent()
    datacenter = content.rootFolder.childEntity[0]
    vm_folder = datacenter.vmFolder
    hosts = datacenter.hostFolder.childEntity
    resource_pool = hosts[0].resourcePool
    logging.info("datacenter={0} vm_folder={1} hosts={2} resource_pool={3}".format(datacenter, vm_folder,
                                                                                   hosts, resource_pool))
    # bare minimum VM shell, no disks. Feel free to edit
    vmx_file = vim.vm.FileInfo(logDirectory=None,
                               snapshotDirectory=None,
                               suspendDirectory=None,
                               vmPathName='[' + datastore_name + '] ')

    config = vim.vm.ConfigSpec(
        name=vm_name,
        memoryMB=128,
        numCPUs=1,
        files=vmx_file,
        guestId='rhel5_64Guest',
        version='vmx-11'
    )

    task = vm_folder.CreateVM_Task(config=config, pool=resource_pool)
    vmdk_ops.wait_for_tasks(si, [task])

    logging.info("create_vm: vm_name=%s, datastore_name=%s", vm_name, datastore_name)
    vm = task.info.result
    if vm:
        logging.info("Found: VM %s", vm_name)
        if vm.runtime.powerState == vim.VirtualMachinePowerState.poweredOff:
            logging.info("Attempting to power on %s", vm_name)
            task = vm.PowerOnVM_Task()
            vmdk_ops.wait_for_tasks(si, [task])
    else:
        error_info = error_code.generate_error_info(error_code.ErrorCode.VM_NOT_FOUND, vm_name)
        logging.error("Cannot find vm %s", vm_name)
        return error_info, None

    return None, vm


def destroy_vm_object(si, vm):
    """ Destroy a VM object """
    logging.info("Trying to destroy VM %s", vm.config.name)
    task = vm.Destroy_Task()
    vmdk_ops.wait_for_tasks(si, [task])


def remove_vm(si, vm, destroy_vm=True):
    """ Remove a VM """
    if vm:
        logging.info("Found: VM %s", vm.config.name)
        if vm.runtime.powerState == vim.VirtualMachinePowerState.poweredOn:
            logging.info("Attempting to power off %s", vm.config.name)
            task = vm.PowerOffVM_Task()
            vmdk_ops.wait_for_tasks(si, [task])

        if destroy_vm:
            destroy_vm_object(si, vm)


def generate_volume_names(tenant, datastore_name, len):
    """ Returns a list of volume names
    e.g. volNames = ['tenant1_vol1@sharedVmfs-0', 'tenant1_vol2@sharedVmfs-0', 'tenant1_vol3@sharedVmfs-0']
    """
    volNames = []
    for x in range(len):
        volNames.append(tenant + "_vol" + str(x + 1) + "@" + datastore_name)
    return volNames


def checkIfVolumeExist(volume_names, result):
    """ checks if names of volume exists in the volumes present in result """
    if not result or not volume_names:
       return False
    for name in volume_names:
        for j in range(len(result)):
              if result[j]['Name'] == name:
                 break
              if j+1 == len(result):
                 logging.error("Cannot find volume %s", result[j]['Name'])
                 return False
    return True


def create_default_tenant_and_privileges(test_obj):
    """ Create default tenant and privilege if not exist"""

    # create DEFAULT tenant if needed
    error_info, tenant_uuid, tenant_name = auth.get_default_tenant()
    if not tenant_uuid:
        logging.debug("create_default_tenant_and_privileges: create DEFAULT tenant")
        error_info, tenant = auth_api._tenant_create(
                                        name=auth_data_const.DEFAULT_TENANT,
                                        default_datastore=auth_data_const.VM_DS,
                                        description=auth_data_const.DEFAULT_TENANT_DESCR,
                                        vm_list=[],
                                        privileges=[])

        if error_info:
            logging.warning(error_info.msg)
        test_obj.assertEqual(error_info, None)

    error_info, existing_privileges = auth_api._tenant_access_ls(auth_data_const.DEFAULT_TENANT)
    test_obj.assertEqual(error_info, None)

    # create access privilege to datastore "_ALL_DS" for _DEFAULT tenant if needed
    if not auth_api.privilege_exist(existing_privileges, auth_data_const.ALL_DS_URL):
        error_info = auth_api._tenant_access_add(name=auth_data_const.DEFAULT_TENANT,
                                                datastore=auth_data_const.ALL_DS,
                                                allow_create=True)

        if error_info:
            logging.warning(error_info.msg)
        test_obj.assertEqual(error_info, None)

    # create access privilege to datastore "_VM_DS" for _DEFAULT tenant if needed
    if not auth_api.privilege_exist(existing_privileges, auth_data_const.VM_DS_URL):
        error_info = auth_api._tenant_access_add(name=auth_data_const.DEFAULT_TENANT,
                                                datastore=auth_data_const.VM_DS,
                                                allow_create=True)

        if error_info:
            logging.warning(error_info.msg)
        test_obj.assertEqual(error_info, None)

def cleanup_tenant(name):
    error_info, vms = auth_api._tenant_vm_ls(name)
    if vms:
        vm_names = [vm_name for (_, vm_name) in vms]
        auth_api._tenant_vm_rm(name=name,
                               vm_list=vm_names)
    auth_api._tenant_rm(name=name,
                        remove_volumes=True)