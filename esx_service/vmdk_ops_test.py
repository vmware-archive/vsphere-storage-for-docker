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
Tests for basic vmdk Operations

'''

import unittest
import sys
import logging
import glob
import os
import os.path

import vmdk_ops
import log_config
import volume_kv
import vsan_policy
import vsan_info
import vmdk_utils
from pyVim import connect
from pyVmomi import vim
import uuid
import auth
import auth_data
import auth_api
import error_code
import vmdk_utils

# Max volumes count we can attach to a singe VM.
MAX_VOL_COUNT_FOR_ATTACH = 60

# Seed for test configurations.
config = {
    # If True, test 60+ attaches (no detach) until if fails.
    "run_max_attach": True
    }

# will do creation/deletion in this folder:
global path

class VolumeNamingTestCase(unittest.TestCase):
    """Unit test for operations with volume names (volume@datastore)"""

    def test_name_parse(self):
        """checks name parsing and error checks
        'volume[@datastore]' -> volume and datastore"""
        testInfo = [
            #  [ full_name. expected_vol_name, expected_datastore_name,  expected_success? ]
            ["MyVolume123_a_.vol@vsanDatastore_11", "MyVolume123_a_.vol", "vsanDatastore_11", True],
            ["a1@x",                            "a1",                  "x",             True],
            ["a1",                              "a1",                  None,            True],
            ["1",                                "1",                 None,             True],
            ["strange-volume-10@vsan:Datastore",  "strange-volume-10",  "vsan:Datastore",     True],
            ["dashes-and stuff !@datastore ok",  "dashes-and stuff !",  "datastore ok",       True],
            ["no-snaps-please-000001@datastore", None,                 None,            False],
            ["TooLong0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789", None, None, False],
            ["Just100Chars0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567",
                           "Just100Chars0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567", None, True],
            ["Volume.123@dots.dot",              "Volume.123",        "dots.dot",       True],
            ["simple_volume",                    "simple_volume",      None,            True],
        ]
        for unit in testInfo:
            full_name, expected_vol_name, expected_ds_name, expected_result = unit
            try:
                vol, ds = vmdk_ops.parse_vol_name(full_name)
                self.assertTrue(expected_result,
                                "Expected volume name parsing to fail for '{0}'".format(full_name))
                self.assertEqual(vol, expected_vol_name, "Vol name mismatch '{0}' expected '{1}'" \
                                                         .format(vol, expected_vol_name))
                self.assertEqual(vol, expected_vol_name, "Datastore name: '{0}' expected: '{1}'" \
                                                         .format(ds, expected_ds_name))
            except vmdk_ops.ValidationError as ex:
                self.assertFalse(expected_result, "Expected vol name parsing to succeed for '{0}'"
                                 .format(full_name))


class VmdkCreateRemoveTestCase(unittest.TestCase):
    """Unit test for VMDK Create and Remove ops"""

    volName = "vol_UnitTest_Create"
    badOpts = {u'policy': u'good', volume_kv.SIZE: u'12unknown', volume_kv.DISK_ALLOCATION_FORMAT: u'5disk'}
    invalid_access_choice = {volume_kv.ACCESS: u'only-read'}
    invalid_access_opt = {u'acess': u'read-write'}
    valid_access_opt = {volume_kv.ACCESS: 'read-only'}
    invalid_attach_as_choice = {volume_kv.ATTACH_AS: u'persisten'}
    invalid_attach_as_opt = {u'atach-as': u'persistent'}
    valid_attach_as_opt_1 = {volume_kv.ATTACH_AS: u'persistent'}
    valid_attach_as_opt_2 = {volume_kv.ATTACH_AS: u'independent_persistent'}
    name = ""
    vm_name = 'test-vm'

    def setUp(self):
        self.name = vmdk_utils.get_vmdk_path(path, self.volName)
        self.policy_names = ['good', 'impossible']
        self.orig_policy_content = ('(("proportionalCapacity" i0) '
                                     '("hostFailuresToTolerate" i0))')
        self.new_policy_content = '(("hostFailuresToTolerate" i0))'
        for n in self.policy_names:
            vsan_policy.create(n, self.orig_policy_content)

    def tearDown(self):
        vmdk_ops.removeVMDK(self.name)
                            
        self.vmdk = None
        for n in self.policy_names:
            vsan_policy.delete(n)

    def testCreateDelete(self):
        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName)
        self.assertEqual(err, None, err)
        self.assertEqual(
            os.path.isfile(self.name), True,
            "VMDK {0} is missing after create.".format(self.name))
        err = vmdk_ops.removeVMDK(self.name)
        self.assertEqual(err, None, err)
        self.assertEqual(
            os.path.isfile(self.name), False,
            "VMDK {0} is still present after delete.".format(self.name))

    def testBadOpts(self):
        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.badOpts)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name)
        self.assertNotEqual(err, None, err)

    def testAccessOpts(self):
        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.invalid_access_choice)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.invalid_access_opt)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.valid_access_opt)
        self.assertEqual(err, None, err)
        err = vmdk_ops.removeVMDK(self.name)
        self.assertEqual(err, None, err)

    def testAttachAsOpts(self):
        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.invalid_attach_as_choice)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.invalid_attach_as_opt)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.valid_attach_as_opt_1)
        self.assertEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name)
        self.assertEqual(err, None, err)

        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=self.name,
                                  vol_name=self.volName,
                                  opts=self.valid_attach_as_opt_2)
        self.assertEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name)
        self.assertEqual(err, None, err)


    @unittest.skipIf(not vsan_info.get_vsan_datastore(),
                    "VSAN is not found - skipping vsan_info tests")
    def testPolicyUpdate(self):
        path = vsan_info.get_vsan_dockvols_path()
        vmdk_path = vmdk_utils.get_vmdk_path(path, self.volName)
        err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                  vmdk_path=vmdk_path,
                                  vol_name=self.volName,
                                  opts={'vsan-policy-name': 'good'})
        self.assertEqual(err, None, err)
        self.assertEqual(None, vsan_policy.update('good',
                                                  self.new_policy_content))
        # Setting an identical policy returns an error msg
        self.assertNotEqual(None, vsan_policy.update('good',
                                                     self.new_policy_content))

        backup_policy_file = vsan_policy.backup_policy_filename(self.name)
        #Ensure there is no backup policy file
        self.assertFalse(os.path.isfile(backup_policy_file))

        # Fail to update because of a bad policy, and ensure there is no backup
        self.assertNotEqual(None, vsan_policy.update('good', 'blah'))
        self.assertFalse(os.path.isfile(backup_policy_file))


    @unittest.skipIf(not vsan_info.get_vsan_datastore(),
                    "VSAN is not found - skipping vsan_info tests")
    def testPolicy(self):
        # info for testPolicy
        testInfo = [
            #    size     policy   expected success?
            ["2000kb", "good", True, "zeroedthick"],
            ["14000pb", "good", False, "zeroedthick"],
            ["bad size", "good", False, "eagerzeroedthick"],
            ["100mb", "impossible", True, "eagerzeroedthick"],
            ["100mb", "good", True, "thin"],
        ]
        path = vsan_info.get_vsan_dockvols_path()
        i = 0
        for unit in testInfo:
            vol_name = '{0}{1}'.format(self.volName, i)
            vmdk_path = vmdk_utils.get_vmdk_path(path,vol_name)
            i = i+1
            # create a volume with requests size/policy and check vs expected result
            err = vmdk_ops.createVMDK(vm_name=self.vm_name,
                                      vmdk_path=vmdk_path,
                                      vol_name=vol_name,
                                      opts={volume_kv.VSAN_POLICY_NAME: unit[1],
                                            volume_kv.SIZE: unit[0],
                                            volume_kv.DISK_ALLOCATION_FORMAT: unit[3]})
            self.assertEqual(err == None, unit[2], err)

            # clean up should fail if the created should have failed.
            err = vmdk_ops.removeVMDK(vmdk_path)
            self.assertEqual(err == None, unit[2], err)

class VmdkCreateCloneRemoveTestCase(unittest.TestCase):
    vm_name = 'test-vm'
    vm_uuid = str(uuid.uuid4())
    volName = "vol_CloneTest"
    volName1 = "vol_CloneTest_1"
    volName2 = "vol_CloneTest_2"
    volName3 = "vol_CloneTest_3"
    vm_datastore = None

    def setUp(self):
        if not self.vm_datastore:
            datastore = vmdk_utils.get_datastores()[0]
            if not datastore:
                logging.error("Cannot find a valid datastore")
                self.assertFalse(True)
            self.vm_datastore = datastore[0]

        path, err = vmdk_ops.get_vol_path(self.vm_datastore)
        self.assertEqual(err, None, err)

        self.name = vmdk_utils.get_vmdk_path(path, self.volName)
        self.name1 = vmdk_utils.get_vmdk_path(path, self.volName1)
        self.name2 = vmdk_utils.get_vmdk_path(path, self.volName2)
        self.name3 = vmdk_utils.get_vmdk_path(path, self.volName3)
        self.badOpts = {volume_kv.CLONE_FROM: self.volName, volume_kv.FILESYSTEM_TYPE: u'ext4',
                        volume_kv.SIZE: volume_kv.DEFAULT_DISK_SIZE}

    def testBadOpts(self):
        err = vmdk_ops.createVMDK(vmdk_path=self.name,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName)
        self.assertEqual(err, None, err)

        err = vmdk_ops.createVMDK(vmdk_path=self.name1,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName1,
                                  opts=self.badOpts,
                                  vm_uuid=self.vm_uuid,
                                  datastore=self.vm_datastore)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name1)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name)
        self.assertEqual(err, None, err)


    def testCreateCloneDelete(self):
        err = vmdk_ops.createVMDK(vmdk_path=self.name,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName)
        self.assertEqual(err, None, err)

        err = vmdk_ops.createVMDK(vmdk_path=self.name1,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName1,
                                  opts={volume_kv.CLONE_FROM: self.volName},
                                  vm_uuid=self.vm_uuid,
                                  datastore=self.vm_datastore)
        self.assertEqual(err, None, err)

        err = vmdk_ops.createVMDK(vmdk_path=self.name2,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName2,
                                  opts={volume_kv.CLONE_FROM: self.volName1},
                                  vm_uuid=self.vm_uuid,
                                  datastore=self.vm_datastore)
        self.assertEqual(err, None, err)

        err = vmdk_ops.createVMDK(vmdk_path=self.name3,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName3,
                                  opts={volume_kv.CLONE_FROM: self.volName2},
                                  vm_uuid=self.vm_uuid,
                                  datastore=self.vm_datastore)
        self.assertEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name)
        self.assertEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name1)
        self.assertEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name2)
        self.assertEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name3)
        self.assertEqual(err, None, err)

class ValidationTestCase(unittest.TestCase):
    """ Test validation of -o options on create """

    @unittest.skipIf(not vsan_info.get_vsan_datastore(),
                     "VSAN is not found - skipping vsan_info tests")

    def setUp(self):
        """ Create a bunch of policies """
        self.policy_names = ['name1', 'name2', 'name3']
        self.policy_content = ('(("proportionalCapacity" i50) '
                               '("hostFailuresToTolerate" i0))')
        self.path = vsan_info.get_vsan_datastore().info.url
        for n in self.policy_names:
            result = vsan_policy.create(n, self.policy_content)
            self.assertEqual(None, result,
                              "failed creating policy %s (%s)" % (n, result))

    def tearDown(self):
        for n in self.policy_names:
            try:
                vsan_policy.delete(n)
            except:
                pass

    def test_success(self):
        sizes = ['2gb', '200tb', '200mb', '5kb']
        sizes.extend([s.upper() for s in sizes])

        for s in sizes:
            for p in self.policy_names:
                for d in volume_kv.VALID_ALLOCATION_FORMATS:
                # An exception should not be raised
                    vmdk_ops.validate_opts({volume_kv.SIZE: s, volume_kv.VSAN_POLICY_NAME: p, volume_kv.DISK_ALLOCATION_FORMAT : d},
                                       self.path)
                    vmdk_ops.validate_opts({volume_kv.SIZE: s}, self.path)
                    vmdk_ops.validate_opts({volume_kv.VSAN_POLICY_NAME: p}, self.path)
                    vmdk_ops.validate_opts({volume_kv.DISK_ALLOCATION_FORMAT: d}, self.path)

    def test_failure(self):
        bad = [{volume_kv.SIZE: '2'}, {volume_kv.VSAN_POLICY_NAME: 'bad-policy'},
        {volume_kv.DISK_ALLOCATION_FORMAT: 'thiN'}, {volume_kv.SIZE: 'mb'}, {'bad-option': '4'}, {'bad-option': 'what',
                                                             volume_kv.SIZE: '4mb'}]
        for opts in bad:
            with self.assertRaises(vmdk_ops.ValidationError):
                vmdk_ops.validate_opts(opts, self.path)

def create_vm(si, vm_name, datastore):
        """ Create a VM """
        content = si.RetrieveContent()
        datacenter = content.rootFolder.childEntity[0]
        vm_folder = datacenter.vmFolder
        hosts = datacenter.hostFolder.childEntity
        resource_pool = hosts[0].resourcePool
        logging.info("datacenter={0} vm_folder={1} hosts={2} resource_pool={3}".format(datacenter, vm_folder,
                                                                                       hosts, resource_pool))

        datastore_path = '[' + datastore + '] ' + vm_name

        # bare minimum VM shell, no disks. Feel free to edit
        vmx_file = vim.vm.FileInfo(logDirectory=None,
                                   snapshotDirectory=None,
                                   suspendDirectory=None,
                                   vmPathName=datastore_path)


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

        vm = vmdk_utils.find_vm_by_name(vm_name)
        if vm:
            logging.debug("Found: VM %s", vm_name)
            if vm.runtime.powerState == vim.VirtualMachinePowerState.poweredOff:
                logging.debug("Attempting to power on %s", vm_name)
                task = vm.PowerOnVM_Task()
                vmdk_ops.wait_for_tasks(si, [task])
        else:
            error_info = error_code.VM_NOT_FOUND.format(vm_name)
            logging.error("Cannot find the test-vm")
            return error_info
        
        return None
               
def remove_vm(si, vm_name):
    """ Remove a VM """
    vm = vmdk_utils.find_vm_by_name(vm_name)
    if vm:
        logging.debug("Found: VM %s", vm_name)
        #logging.debug("The current powerState is  : %s", format(vm[0].runtime.powerState)))
        if vm.runtime.powerState == vim.VirtualMachinePowerState.poweredOn:
            logging.debug("Attempting to power off %s", vm_name)
            task = vm.PowerOffVM_Task()
            vmdk_ops.wait_for_tasks(si, [task])
        
        logging.debug("Trying to destroy VM %s", vm_name)    
        task = vm.Destroy_Task()
        vmdk_ops.wait_for_tasks(si, [task])

class VmdkAttachDetachTestCase(unittest.TestCase):
    """ Unit test for VMDK Attach and Detach ops """

    volNamePre = "vol_UnitTest_Attach"
    vm_name = 'test-vm'
    if config["run_max_attach"]:
        max_vol_count = MAX_VOL_COUNT_FOR_ATTACH
    else:
        max_vol_count = 1
    datastore_path = None
    datastore_name = None

    def setUp(self):
        """ Setup run before each test """
        logging.debug("VMDKAttachDetachTest setUp path =%s", path)
        self.cleanup()

        if (not self.datastore_name):
            datastores = vmdk_utils.get_datastores()
            datastore = datastores[0]
            if (not datastore):
                logging.error("Cannot find a valid datastore")
                self.assertFalse(True)
            self.datastore_name = datastore[0]
            self.datastore_path = datastore[2]
            logging.debug("datastore_name=%s datastore_path=%s", self.datastore_name,
                                                                 self.datastore_path)   
        
        # get service_instance, and create a VM
        si = vmdk_ops.get_si()
        error = create_vm(si=si, 
                          vm_name=self.vm_name, 
                          datastore=self.datastore_name)
        if error:
            self.assertFalse(True)

        # create max_vol_count+1 VMDK files
        for id in range(1, self.max_vol_count + 2):
            volName = 'VmdkAttachDetachTestVol' + str(id)
            fullpath = os.path.join(self.datastore_path, volName + '.vmdk')
            self.assertEqual(None,
                                vmdk_ops.createVMDK(vm_name=self.vm_name,
                                                    vmdk_path=fullpath,
                                                    vol_name=volName))
            
    def tearDown(self):
        """ Cleanup after each test """
        logging.debug("VMDKAttachDetachTest tearDown path")
        self.cleanup()
    
    
    
    def cleanup(self):
        # remove VM
        si = vmdk_ops.get_si()
        remove_vm(si, self.vm_name)

        for v in self.get_testvols():
            self.assertEqual(
                None,
                vmdk_ops.removeVMDK(os.path.join(v['path'], v['filename'])))

    def get_testvols(self):
        return [x
                for x in vmdk_utils.get_volumes(None)
                if x['filename'].startswith('VmdkAttachDetachTestVol')]

    
    def testAttachDetach(self):
        logging.info("Start VMDKAttachDetachTest")
        si = vmdk_ops.get_si()
        #find test_vm
        vm = [d for d in si.content.rootFolder.childEntity[0].vmFolder.childEntity 
              if d.config.name == self.vm_name]
        self.assertNotEqual(None, vm)

        # attach max_vol_count disks
        for id in range(1, self.max_vol_count+1):
            volName = 'VmdkAttachDetachTestVol' + str(id)
            fullpath = os.path.join(self.datastore_path, volName + '.vmdk')
            ret = vmdk_ops.disk_attach(vmdk_path=fullpath,
                                       vm=vm[0])
            logging.info("Returned '%s'", ret)
            self.assertFalse("Error" in ret)

        if config["run_max_attach"]:
            # attach one more disk, which should fail    
            volName = 'VmdkAttachDetachTestVol' + str(self.max_vol_count+1)
            fullpath = os.path.join(self.datastore_path, volName + '.vmdk')
            ret = vmdk_ops.disk_attach(vmdk_path=fullpath, vm=vm[0])
            self.assertTrue("Error" in ret)

        # detach all the attached disks
        for id in range(1, self.max_vol_count + 1):
            volName = 'VmdkAttachDetachTestVol' + str(id)
            fullpath = os.path.join(self.datastore_path, volName + '.vmdk')
            ret = vmdk_ops.disk_detach(vmdk_path=fullpath,
                                       vm=vm[0])
            self.assertTrue(ret is None)

class VmdkAuthorizeTestCase(unittest.TestCase):
    """ Unit test for VMDK Authorization """
    
    vm_name = 'test-vm'
    vm_uuid = str(uuid.uuid4())
    tenant1 = None
        
    def setUp(self):
        """ Setup run before each test """
        logging.info("VMDKAuthorizeTest setUp path =%s", path)
        self.auth_mgr = auth_data.AuthorizationDataManager()
        self.auth_mgr.connect()

        self.cleanup()     
    
    def cleanup(self):
        error_info, exist_tenant = self.auth_mgr.get_tenant('vmdk_auth_test')
        if exist_tenant:
             error_info = self.auth_mgr.remove_tenant(exist_tenant.id, False)
             self.assertEqual(error_info, None)
             error_info = self.auth_mgr.remove_volumes_from_volumes_table(exist_tenant.id)
             self.assertEqual(error_info, None)

    def tearDown(self):
        logging.info("VMDKAuthorizeTest tearDown path =%s", path)
        self.cleanup()
       
    def test_vmdkop_authorize(self):
        """ Test vmdkop authorize """
        vm_ds = 'datastore1'
        vms = [(self.vm_uuid, self.vm_name)]
        privileges = []
        default_datastore='default_ds'
        default_privileges = {'datastore': default_datastore,
                              'create_volume': 0,
                              'delete_volume': 0,
                              'mount_volume': 0,
                              'max_volume_size': 0,
                              'usage_quota': 0}
                
        error_info, tenant1 = self.auth_mgr.create_tenant('vmdk_auth_test', 'Tenant used to vmdk_auth_test', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        # test CMD_CREATE without "create_volume" set
        privileges = [{'datastore': vm_ds,
                       'create_volume': 0,
                       'delete_volume': 0,
                       'mount_volume': 1,
                       'max_volume_size': 500,
                       'usage_quota': 1000}]
        
        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)
        opts={u'size': u'100MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, vm_ds, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, "No create privilege" )

        # set "create_volume" privilege to true 
        privileges = [{'datastore': vm_ds,
                       'create_volume': 1,
                       'delete_volume': 0,
                       'mount_volume': 1,
                       'max_volume_size': 500,
                       'usage_quota': 1000}]

        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, vm_ds, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, None)

        if not error_info:
            error_info = auth.add_volume_to_volumes_table(tenant1.id, vm_ds, "VmdkAuthorizeTestVol1", 100)
            self.assertEqual(error_info, None)

        opts={u'size': u'600MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, vm_ds, auth.CMD_CREATE, opts)
        # create a volume with 600MB which exceed the"max_volume_size", command should fail
        self.assertEqual(error_info, "volume size exceeds the max volume size limit") 

        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, vm_ds, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, None)

        if not error_info:
            error_info = auth.add_volume_to_volumes_table(tenant1.id, vm_ds, "VmdkAuthorizeTestVol2", 500)
            self.assertEqual(error_info, None)
        
        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, vm_ds, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, "The total volume size exceeds the usage quota")

        # delete volume
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, vm_ds, auth.CMD_REMOVE, opts)
        self.assertEqual(error_info, "No delete privilege")

        privileges = [{'datastore': vm_ds,
                       'create_volume': 1,
                       'delete_volume': 1,
                       'mount_volume': 1,
                       'max_volume_size': 500,
                       'usage_quota': 1000}]

        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)

        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, vm_ds, auth.CMD_REMOVE, opts)
        self.assertEqual(error_info, None)

        # remove the tenant
        error_info = self.auth_mgr.remove_tenant(tenant1.id, False)
        self.assertEqual(error_info, None)
        error_info = self.auth_mgr.remove_volumes_from_volumes_table(tenant1.id)
        self.assertEqual(error_info, None)

class VmdkTenantTestCase(unittest.TestCase):
    """ Unit test for VMDK ops for multi-tenancy """
    non_tenant_vol_name = "non_tenant_vol1"

    # tenant1 info
    tenant1_name = "test_tenant1"
    vm1_name = 'test_vm1'
    tenant1_vol1_name = 'tenant1_vol1'
    tenant1_vol2_name = 'tenant1_vol2'
    tenant1_vol3_name = 'tenant1_vol3'
    vm1_config_path = None

    # tenant2 info
    tenant2_name = "test_tenant2"
    vm2_name = 'test_vm2'
    tenant2_vol1_name = 'tenant2_vol1'
    tenant2_vol2_name = 'tenant2_vol2'
    tenant2_vol3_name = 'tenant2_vol3'
    vm2_config_path = None
    
    datastore_name = None
    datastore_path = None
    

    def setUp(self):
        """ Setup run before each test """
        logging.info("VMDKTenantTest setUp path =%s", path)
        
        if (not self.datastore_name):
            datastores = vmdk_utils.get_datastores()
            if datastores:
                datastore = datastores[0]
                self.datastore_name = datastore[0]
                self.datastore_path = datastore[2]
                logging.debug("datastore_name=%s datastore_path=%s", self.datastore_name,
                                                                     self.datastore_path)
            else:
                logging.error("Cannot find a valid datastore")
                self.assertFalse(True)

        self.cleanup()
        # get service_instance, and create VMs
        si = vmdk_ops.get_si()
        error = create_vm(si=si, 
                          vm_name=self.vm1_name, 
                          datastore=self.datastore_name)
        if error:
            self.assertFalse(True)

        self.vm1_config_path = vmdk_utils.get_vm_config_path(self.vm1_name)

        error = create_vm(si=si, 
                          vm_name=self.vm2_name, 
                          datastore=self.datastore_name)
        if error:
            self.assertFalse(True)
        self.vm2_config_path = vmdk_utils.get_vm_config_path(self.vm2_name)
        
        # create tenant1 without adding any vms and privileges
        name = self.tenant1_name
        vm_list = None 
        description = "Test tenant1" 
        default_datastore ="default_ds" 
        default_privileges =  {}
        privileges = []
  
        error_info, tenant = auth_api._tenant_create(
                                                    name = name, 
                                                    description = description, 
                                                    default_datastore = default_datastore, 
                                                    default_privileges = default_privileges, 
                                                    vm_list = vm_list, 
                                                    privileges = privileges)
        self.assertEqual(None, error_info)

        # create tenant2 without adding any vms and privileges
        name = self.tenant2_name
        vm_list = None 
        description = "Test tenant2" 
        default_datastore ="default_ds" 
        default_privileges =  {}
        privileges = []
  
        error_info, tenant = auth_api._tenant_create(
                                                    name = name, 
                                                    description = description, 
                                                    default_datastore = default_datastore, 
                                                    default_privileges = default_privileges, 
                                                    vm_list = vm_list, 
                                                    privileges = privileges)
        self.assertEqual(None, error_info)      
          
    def tearDown(self):
        """ Cleanup after each test """
        logging.info("VMDKTenantTest  tearDown path")
        self.cleanup()      
            
    def cleanup(self):
        # cleanup existing non-tenant volume
        if not self.datastore_path:
            vmdk_path = vmdk_utils.get_vmdk_path(self.datastore_path, self.non_tenant_vol_name)
            if vmdk_ops.getVMDK(vmdk_path, self.non_tenant_vol_name, self.datastore_name):
                vmdk_ops.removeVMDK(vmdk_path)

        # cleanup existing tenant
        error_info = auth_api._tenant_rm(
                                         name = self.tenant1_name, 
                                         remove_volumes = True) 
                                                
        error_info = auth_api._tenant_rm(
                                         name = self.tenant2_name, 
                                         remove_volumes = True) 
                                                
        # remove VM
        si = vmdk_ops.get_si()
        remove_vm(si, self.vm1_name)
        remove_vm(si, self.vm2_name)
       
    def test_create_and_remove_volume_on_non_tenant_vm(self):
        """ Test volume create remove on a VM which does not belong to any tenant """
        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)

        # run create command
        opts={u'size': u'100MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.non_tenant_vol_name, opts)
        self.assertEqual(None, error_info)

        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.non_tenant_vol_name, opts)
        self.assertEqual(None, error_info)
    
    def test_vmdkops_on_tenant_vm(self):
        """ Test vmdk life cycle on a VM which belongs to a tenant """
        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)
        # add vm to tenant
        error_info = auth_api._tenant_vm_add(
                                         name = self.tenant1_name, 
                                         vm_list = [self.vm1_name]) 
        self.assertEqual(None, error_info)

        # run create command
        opts={u'size': u'100MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual({u'Error': 'No create privilege'}, error_info)

        # set access privileges, create a volume with 100MB
        rights = ["create", "mount"]
        volume_maxsize = "500MB"
        volume_totalsize = "1GB"
        error_info = auth_api._tenant_access_add(name = self.tenant1_name, 
                                                 datastore = self.datastore_name, 
                                                 rights = rights, 
                                                 volume_maxsize = volume_maxsize, 
                                                 volume_totalsize = volume_totalsize)
        # create a volume with default size
        opts={u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        # test attach a volume
        opts={}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_ATTACH, self.tenant1_vol1_name, opts)
        self.assertFalse("Error" in result)
     
        # test detach a volume
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_DETACH, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        # create a volume with 600MB which exceed the volume_maxsize
        opts={u'size': u'600MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual({u'Error': 'volume size exceeds the max volume size limit'}, error_info)

        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol2_name, opts)
        self.assertEqual(None, error_info)

        # create a volume with 500MB, and total_storeage used by this tenant will exceed volume_totalsize
        opts={u'size': u'500mb', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol3_name, opts)
        self.assertEqual({u'Error': 'The total volume size exceeds the usage quota'}, error_info)

        # try to delete the first volume, which should fail
        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.tenant1_vol1_name, opts)
        self.assertEqual({u'Error': 'No delete privilege'}, error_info)

        # add delete privilege 
        add_rights = ["delete"]
        error_info = auth_api._tenant_access_set(name = self.tenant1_name, 
                                                 datastore = self.datastore_name, 
                                                 add_rights = add_rights,
                                                 rm_rights = None,
                                                 volume_maxsize = None, 
                                                 volume_totalsize = None)
        self.assertEqual(None, error_info)
        
        # try to delete the volume again, which should succeed
        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        # create the volume again, which should succeed
        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol3_name, opts)
        self.assertEqual(None, error_info)

        # list volumes
        opts = {}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)
       
        # result should have two volumes :tenant1_vol2, and  tenant1_vol3
        self.assertEqual(2, len(result))
        self.assertEqual("tenant1_vol2", result[0]['Name'])
        self.assertEqual("tenant1_vol3", result[1]['Name'])
    
    def test_vmdkops_on_different_tenants(self):
        """ Test vmdkops on VMs which belong to different tenant """
        # add vm1 to tenant1
        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)
        error_info = auth_api._tenant_vm_add(
                                         name = self.tenant1_name, 
                                         vm_list = [self.vm1_name]) 
        self.assertEqual(None, error_info)

        # set access privileges, create a volume with 100MB
        rights = ["all"]
        volume_maxsize = "500MB"
        volume_totalsize = "2GB"
        error_info = auth_api._tenant_access_add(name = self.tenant1_name, 
                                                 datastore = self.datastore_name, 
                                                 rights = rights, 
                                                 volume_maxsize = volume_maxsize, 
                                                 volume_totalsize = volume_totalsize)
        # create a volume with default size
        opts={u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)
        
        # create a volume with 500MB
        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol2_name, opts)
        self.assertEqual(None, error_info)

        # create a volume with 500MB
        opts={u'size': u'500mb', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol3_name, opts)
        self.assertEqual(None, error_info)

        # add vm2 to tenant2
        vm2_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm2_name)
        error_info = auth_api._tenant_vm_add(
                                         name = self.tenant2_name, 
                                         vm_list = [self.vm2_name]) 
        self.assertEqual(None, error_info)

        # set access privileges, create a volume with 100MB
        rights = ["all"]
        volume_maxsize = "500MB"
        volume_totalsize = "2GB"
        error_info = auth_api._tenant_access_add(name = self.tenant2_name, 
                                                 datastore = self.datastore_name, 
                                                 rights = rights, 
                                                 volume_maxsize = volume_maxsize, 
                                                 volume_totalsize = volume_totalsize)
        # create a volume with default size
        opts={u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm2_uuid, self.vm2_name, self.vm2_config_path, auth.CMD_CREATE, self.tenant2_vol1_name, opts)
        self.assertEqual(None, error_info)
        
        # create a volume with 500MB
        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm2_uuid, self.vm2_name, self.vm2_config_path, auth.CMD_CREATE, self.tenant2_vol2_name, opts)
        self.assertEqual(None, error_info)

        # create a volume with 500MB
        opts={u'size': u'500mb', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm2_uuid, self.vm2_name, self.vm2_config_path, auth.CMD_CREATE, self.tenant2_vol3_name, opts)
        self.assertEqual(None, error_info)
        
        # list volumes for tenant1
        opts = {}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)
       
        self.assertEqual(3, len(result))
        self.assertEqual("tenant1_vol1", result[0]['Name'])
        self.assertEqual("tenant1_vol2", result[1]['Name'])
        self.assertEqual("tenant1_vol3", result[2]['Name'])

        # list volumes for tenant2
        opts = {}
        result = vmdk_ops.executeRequest(vm2_uuid, self.vm2_name, self.vm2_config_path, 'list', None, opts)
       
        self.assertEqual(3, len(result))
        self.assertEqual("tenant2_vol1", result[0]['Name'])
        self.assertEqual("tenant2_vol2", result[1]['Name'])
        self.assertEqual("tenant2_vol3", result[2]['Name'])

    
    

if __name__ == '__main__':
    # configure the log, find the dir and run the tests
    log_config.configure()
    volume_kv.init()

    # Calculate the path
    paths = glob.glob("/vmfs/volumes/[a-zA-Z]*/dockvols")
    logging.info("Found datastores: %s", paths)
    if paths:
        # WARNING: for many datastores with dockvols, this picks up the first
        path = paths[0]
    else:
        # create dir in a datastore (just pick first datastore if needed)
        path = glob.glob("/vmfs/volumes/[a-zA-Z]*")[0] + "/dockvols"
        logging.debug("Directory does not exist - creating %s", path)
        os.makedirs(path)

    def clean_path(path):
        if not path:
            logging.info("Directory clean up - empty dir passed")
            return

        logging.info("Directory clean up - removing  %s", path)
        try:
            # TODO: need to use osfs-rmdir on VSAN. For now jus yell if it failed
            os.removedirs(path)
        except Exception as e:
            logging.warning("Directory clean up failed  -  %s, err: %s", path, e)

    logging.info("Running tests. Directory used: %s", path)
    try:
        unittest.main()
    except:
        clean_path(path)
         # If the unittest failed, re-raise the error
        raise

    clean_path(path)
