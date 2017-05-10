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
import time

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
import auth_data_const
import vmdk_utils
import random
import convert
import error_code
from error_code import ErrorCode
from error_code import error_code_to_message
from error_code import generate_error_info
from vmdkops_admin_sanity_test import ADMIN_CLI
import glob
import vmdkops_admin
import test_utils
# Max volumes count we can attach to a singe VM.
MAX_VOL_COUNT_FOR_ATTACH = 60

# Admin CLI to control config DB init
ADMIN_INIT_LOCAL_AUTH_DB = ADMIN_CLI + " config init --local"
ADMIN_RM_LOCAL_AUTH_DB = ADMIN_CLI + " config rm --local --confirm"

# backups to cleanup
CONFIG_DB_BAK_GLOB = "/etc/vmware/vmdkops/auth-db.bak_*"

# Seed for test configurations.
config = {
    # If True, test 60+ attaches (no detach) until if fails.
    "run_max_attach": False
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
    vm_name = test_utils.generate_test_vm_name()

    def setUp(self):
        self.name = vmdk_utils.get_vmdk_path(path, self.volName)
        self.policy_names = ['good', 'impossible', 'bad_string']
        self.orig_policy_content = ('(("proportionalCapacity" i0) '
                                     '("hostFailuresToTolerate" i0))')

        self.impossible_policy_content = ('(("proportionalCapacity" i0) '
                                     '("hostFailuresToTolerate" i3))')

        self.new_policy_content = '(("hostFailuresToTolerate" i0))'

        # Missing paranthesis at the end
        self.bad_string_policy_content = ('(("proportionalCapacity" i0) '
                                     '("hostFailuresToTolerate" i3)')

        vsan_policy.create('good', self.orig_policy_content)
        vsan_policy.create('impossible', self.impossible_policy_content)
        vsan_policy.create('bad_string', self.bad_string_policy_content)


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
        self.assertEqual(err, None, err)

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
    def testPolicy(self):
        # info for testPolicy
        testInfo = [
            #    size     policy   expected success?
            ["2000mb", "good", True, "zeroedthick"],
            ["14000pb", "good", False, "zeroedthick"],
            ["bad size", "good", False, "eagerzeroedthick"],
            ["100mb", "impossible", False, "eagerzeroedthick"],
            ["100mb", "good", True, "thin"],
            ["100mb", "bad_string", False, "eagerzeroedthick"],
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

            # clean up would succeed with #1084.
            err = vmdk_ops.removeVMDK(vmdk_path)
            self.assertEqual(err, None, err)

class VmdkCreateCloneRemoveTestCase(unittest.TestCase):
    vm_name = test_utils.generate_test_vm_name()
    vm_uuid = str(uuid.uuid4())
    volName = "vol_CloneTest"
    volName1 = "vol_CloneTest_1"
    volName2 = "vol_CloneTest_2"
    volName3 = "vol_CloneTest_3"
    vm_datastore = None
    vm_datastore_url = None

    def setUp(self):
        if not self.vm_datastore:
            datastore = vmdk_utils.get_datastores()[0]
            if not datastore:
                logging.error("Cannot find a valid datastore")
                self.assertFalse(True)
            self.vm_datastore = datastore[0]
            self.vm_datastore_url = datastore[1]

        path, err = vmdk_ops.get_vol_path(self.vm_datastore, auth_data_const.DEFAULT_TENANT)
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
                                  datastore_url=self.vm_datastore_url)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name1)
        self.assertEqual(err, None, err)

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
                                  datastore_url=self.vm_datastore_url)
        self.assertEqual(err, None, err)

        err = vmdk_ops.createVMDK(vmdk_path=self.name2,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName2,
                                  opts={volume_kv.CLONE_FROM: self.volName1},
                                  vm_uuid=self.vm_uuid,
                                  datastore_url=self.vm_datastore_url)
        self.assertEqual(err, None, err)

        err = vmdk_ops.createVMDK(vmdk_path=self.name3,
                                  vm_name=self.vm_name,
                                  vol_name=self.volName3,
                                  opts={volume_kv.CLONE_FROM: self.volName2},
                                  vm_uuid=self.vm_uuid,
                                  datastore_url=self.vm_datastore_url)
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
        sizes = ['2gb', '200tb', '200mb']
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

class VmdkAttachDetachTestCase(unittest.TestCase):
    """ Unit test for VMDK Attach and Detach ops """

    volNamePre = "vol_UnitTest_Attach"
    vm_name = test_utils.generate_test_vm_name()
    vm = None
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
        error, self.vm = test_utils.create_vm(si=si,
                                              vm_name=self.vm_name,
                                              datastore_name=self.datastore_name)
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
        test_utils.remove_vm(si, self.vm)

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
        # find test_vm
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

    vm_uuid = str(uuid.uuid4())
    vm_name = test_utils.generate_test_vm_name()
    tenant1 = None
    datastore_name = None
    datastore_url = None

    def setUp(self):
        """ Setup run before each test """
        logging.info("VMDKAuthorizeTest setUp path =%s", path)
        if (not self.datastore_name):
            datastores = vmdk_utils.get_datastores()
            if datastores:
                datastore = datastores[0]
                self.datastore_name = datastore[0]
                self.datastore_url = datastore[1]
                self.datastore_path = datastore[2]
                logging.debug("datastore_name=%s datastore_url=%s datastore_path=%s",
                              self.datastore_name, self.datastore_url, self.datastore_path)
            else:
                logging.error("Cannot find a valid datastore")
                self.assertFalse(True)

        self.auth_mgr = auth_data.AuthorizationDataManager()
        self.auth_mgr.connect()

        self.cleanup()

    def cleanup(self):
        logging.info("VMDKAuthorizeTest cleanup")
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
        vms = [(self.vm_uuid, self.vm_name)]
        privileges = []

        error_info, tenant1 = self.auth_mgr.create_tenant(name='vmdk_auth_test',
                                                          description='Tenant used to vmdk_auth_test',
                                                          vms=vms,
                                                          privileges=privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        # test CMD_CREATE without "create_volume" set
        privileges = [{'datastore_url': self.datastore_url,
                       'allow_create': 0,
                       'max_volume_size': 500,
                       'usage_quota': 1000}]

        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)
        opts={u'size': u'100MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, self.datastore_url, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, "No create privilege")

        # set "create_volume" privilege to true
        privileges = [{'datastore_url': self.datastore_url,
                       'allow_create': 1,
                       'max_volume_size': 500,
                       'usage_quota': 1000}]

        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, self.datastore_url, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, None)

        if not error_info:
            error_info = auth.add_volume_to_volumes_table(tenant1.id, self.datastore_url, "VmdkAuthorizeTestVol1", 100)
            self.assertEqual(error_info, None)

        opts={u'size': u'600MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, self.datastore_url, auth.CMD_CREATE, opts)
        # create a volume with 600MB which exceed the"max_volume_size", command should fail
        self.assertEqual(error_info, "Volume size exceeds the max volume size limit")

        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, self.datastore_url, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, None)

        if not error_info:
            error_info = auth.add_volume_to_volumes_table(tenant1.id, self.datastore_url, "VmdkAuthorizeTestVol2", 500)
            self.assertEqual(error_info, None)

        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, self.datastore_url, auth.CMD_CREATE, opts)
        self.assertEqual(error_info, "The total volume size exceeds the usage quota")

        # delete volume
        error_info, tenant_uuid, tenant_name = auth.authorize(self.vm_uuid, self.datastore_url, auth.CMD_REMOVE, opts)
        self.assertEqual(error_info, None)

        # remove the tenant
        error_info = self.auth_mgr.remove_tenant(tenant1.id, False)
        self.assertEqual(error_info, None)
        error_info = self.auth_mgr.remove_volumes_from_volumes_table(tenant1.id)
        self.assertEqual(error_info, None)

class VmdkTenantTestCase(unittest.TestCase):
    """ Unit test for VMDK ops for multi-tenancy """
    default_tenant_vol1_name = "default_tenant_vol1"
    default_tenant_vol2_name = "default_tenant_vol2"
    default_tenant_vol3_name = "default_tenant_vol3"
    default_tenant_vol4_name = "default_tenant_vol4"
    default_tenant_vols = [default_tenant_vol1_name, default_tenant_vol2_name,
                           default_tenant_vol3_name, default_tenant_vol4_name]

    # tenant1 info
    tenant1_name = "test_tenant1"
    vm1_name = test_utils.generate_test_vm_name()
    vm1 = None
    tenant1_vol1_name = 'tenant1_vol1'
    tenant1_vol2_name = 'tenant1_vol2'
    tenant1_vol3_name = 'tenant1_vol3'
    vm1_config_path = None
    tenant1_new_name = "new_test_tenant1"
    vm3_name = test_utils.generate_test_vm_name()
    vm3 = None

    # tenant2 info
    tenant2_name = "test_tenant2"
    vm2_name = test_utils.generate_test_vm_name()
    vm2 = None
    tenant2_vol1_name = 'tenant2_vol1'
    tenant2_vol2_name = 'tenant2_vol2'
    tenant2_vol3_name = 'tenant2_vol3'
    vm2_config_path = None

    datastore_name = None
    datastore_path = None
    datastore1_name = None
    datastore1_path = None

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
                if len(datastores) > 1:
                    datastore1 = datastores[1]
                    self.datastore1_name = datastore1[0]
                    self.datastoer1_path = datastore[2]
                    logging.debug("Found second datastore: datastore_name=%s datastore_path=%s",
                                  self.datastore1_name, self.datastore1_path)
            else:
                logging.error("Cannot find a valid datastore")
                self.assertFalse(True)

        self.cleanup()
        # get service_instance, and create VMs
        si = vmdk_ops.get_si()
        error, self.vm1 = test_utils.create_vm(si=si,
                                               vm_name=self.vm1_name,
                                               datastore_name=self.datastore_name)
        if error:
            self.assertFalse(True)

        self.vm1_config_path = vmdk_utils.get_vm_config_path(self.vm1_name)
        logging.info("VmdkTenantTestCase: create vm1 name=%s Done", self.vm1_name)

        error, self.vm2 = test_utils.create_vm(si=si,
                                    vm_name=self.vm2_name,
                                    datastore_name=self.datastore_name)
        if error:
            self.assertFalse(True)
        self.vm2_config_path = vmdk_utils.get_vm_config_path(self.vm2_name)
        logging.info("VmdkTenantTestCase: create vm2 name=%s Done", self.vm2_name)

        if self.datastore1_name:
            # create a volume from different datastore
            error, self.vm3 = test_utils.create_vm(si=si,
                                        vm_name=self.vm3_name,
                                        datastore_name=self.datastore1_name)
            if error:
                self.assertFalse(True)
            self.vm3_config_path = vmdk_utils.get_vm_config_path(self.vm3_name)
            logging.info("VmdkTenantTestCase: create vm3 name=%s Done", self.vm3_name)

        # create DEFAULT tenant and privilege if missing

        test_utils.create_default_tenant_and_privileges(self)

        # create tenant1 without adding any vms and privileges
        name = self.tenant1_name
        vm_list = None
        description = "Test tenant1"
        privileges = []

        error_info, tenant = auth_api._tenant_create(
                                                    name=name,
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description=description,
                                                    vm_list=vm_list,
                                                    privileges=privileges)
        self.assertEqual(None, error_info)

        # create tenant2 without adding any vms and privileges
        name = self.tenant2_name
        vm_list = None
        description = "Test tenant2"
        privileges = []

        error_info, tenant = auth_api._tenant_create(
                                                    name=name,
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description=description,
                                                    vm_list=vm_list,
                                                    privileges=privileges)
        self.assertEqual(None, error_info)

    def tearDown(self):
        """ Cleanup after each test """
        logging.info("VMDKTenantTest  tearDown path")
        self.cleanup()

    def cleanup(self):
        # cleanup existing volume under DEFAULT tenant
        logging.info("VMDKTenantTest cleanup")
        if self.datastore_path:
            default_tenant_path = os.path.join(self.datastore_path, auth_data_const.DEFAULT_TENANT)
            for vol in self.default_tenant_vols:
                vmdk_path = vmdk_utils.get_vmdk_path(default_tenant_path, vol)
                response = vmdk_ops.getVMDK(vmdk_path, vol, self.datastore_name)
                if not "Error" in response:
                    logging.debug("cleanup: remove volume %s", vmdk_path)
                    vmdk_ops.removeVMDK(vmdk_path)

        # cleanup existing tenants
        test_utils.cleanup_tenant(self.tenant1_name)
        test_utils.cleanup_tenant(self.tenant1_new_name)
        test_utils.cleanup_tenant(self.tenant2_name)

        # remove VM
        si = vmdk_ops.get_si()
        if self.vm1:
            test_utils.remove_vm(si, self.vm1)
        if self.vm2:
            test_utils.remove_vm(si, self.vm2)
        if self.vm3:
            test_utils.remove_vm(si, self.vm3)

    def test_vmdkops_on_default_tenant_vm(self):
        """ Test vmdk life cycle on a VM which belongs to DEFAULT tenant """
        # This test test the following cases:
        # 1. DEFAULT tenant, privilege to datastore "_ALL_DS", and privilege to datastore "_VM_DS""
        # are present, vmdk_ops from VM which is not owned by any tenant, vmdk_ops should succeed, and the volumes
        # will be created in the _VM_DS
        # 2. change the default_datastore for DEFAULT tenant,  volume create with short name should be created in the
        # default datastore instead of VM datastore
        # 3. Only privilege to  "_VM_DS" present, "create volume" should succed
        # 4. REMOVE DEFAULT tenant, "create volume" should fail

        # run create, attach, detach, remove command when DEFAULT tenant and  privileges to "_ALL_DS" and "_VM_DS" are present
        # This is the case after user fresh install
        logging.info("test_vmdkops_on_default_tenant_vm")
        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)

        # run create command
        opts={u'size': u'100MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        # test attach a volume
        opts={}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_ATTACH, self.default_tenant_vol1_name, opts)
        self.assertFalse("Error" in result)

        # test detach a volume
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_DETACH, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        # run remove command
        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        # create a volume with default size
        # default_datastore for _DEFAULT tenant is set to "VM_DS", a volume will be
        # tried to create on the vm_datastore, which is self.datastore_name
        opts={u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        # Only run this test if second datastore exists
        if self.datastore1_name:
            # Change the "default_datastore" of _DEFAULT tenant to second datastore "self.datastore1_name"
            # Then the full access privilege will be created to this new "default_datastore"
            error_info = auth_api._tenant_update(name=auth_data_const.DEFAULT_TENANT,
                                                 default_datastore=self.datastore1_name,
                                                 )
            # create a volume with default size
            # default_datastore is set for _DEFAULT tenant, a volume will be
            # tried to create on the default_datastore, which is "self.datastore1_name"
            opts={u'fstype': u'ext4'}
            error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.default_tenant_vol2_name, opts)
            self.assertEqual(None, error_info)

            # list volumes
            opts = {}
            result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)

            # there should be two volumes "default_tenant1_vol1" and "default_tenant_vol2"
            self.assertEqual(len(result), 2)
            self.assertEqual(self.default_tenant_vol1_name + "@" + self.datastore_name, result[0]['Name'])
            self.assertEqual(self.default_tenant_vol2_name + "@" + self.datastore1_name, result[1]['Name'])

            # remove privilege to datastore "self.datastore1_name"
            error_info = auth_api._tenant_access_rm(name=auth_data_const.DEFAULT_TENANT,
                                                    datastore=self.datastore1_name)
            # datastore "self.datastore1_name" is still the "default_datastore" for _DEFAULT tenant, cannot remove
            self.assertNotEqual(None, error_info)

            # change the "default_datastore" for _DEFAULT tenant to "_ALL_DS", which should fail
            # "_ALL_DS" cannot be set as "default_datastore"
            error_info = auth_api._tenant_update(name=auth_data_const.DEFAULT_TENANT,
                                                 default_datastore=auth_data_const.ALL_DS)
            self.assertNotEqual(None, error_info)

            # set the "default_datastore" for _DEFAULT tenant to "_VM_DS", which should succeed
            error_info = auth_api._tenant_update(name=auth_data_const.DEFAULT_TENANT,
                                                 default_datastore=auth_data_const.VM_DS)
            self.assertEqual(None, error_info)

            # remove the access privilege to "self.datastore1_name"
            error_info = auth_api._tenant_access_rm(auth_data_const.DEFAULT_TENANT, self.datastore1_name)
            self.assertEqual(None, error_info)

        # remove "_ALL_DS"" privileges, and run create command, which should succeed since we still have
        # access privilege to "_VM_DS"
        error_info = auth_api._tenant_access_rm(auth_data_const.DEFAULT_TENANT, auth_data_const.ALL_DS)
        self.assertEqual(None, error_info)

        opts={u'size': u'100MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.default_tenant_vol3_name, opts)
        self.assertEqual(None, error_info)

        # try to create volume on a different datastore, which should fail
        if self.datastore1_name:
            full_vol_name = self.default_tenant_vol4_name + "@" + self.datastore1_name
            opts={u'size': u'100MB', u'fstype': u'ext4'}
            error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, full_vol_name, opts)
            self.assertNotEqual(None, error_info)

        # remove "_VM_DS" privilege, which should fail since "_VM_DS"
        # is the "default_datastore" of _DEFAULT tenant
        # cannot remove it
        error_info = auth_api._tenant_access_rm(auth_data_const.DEFAULT_TENANT, auth_data_const.VM_DS)
        self.assertNotEqual(None, error_info)

        # remove DEFAULT tenant, and run create command, which should fail
        error_info = auth_api._tenant_rm(auth_data_const.DEFAULT_TENANT, False)
        self.assertEqual(None, error_info)

        opts={u'size': u'100MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.default_tenant_vol4_name, opts)
        self.assertNotEqual(None, error_info)


    def test_vmdkops_on_tenant_vm(self):
        """ Test vmdk life cycle on a VM which belongs to a tenant """
        # 1. named tenant "tenant1" is created with default_datastore set to "VM_DS", test volume
        # create with short name and full name.  With short name, volume is created in the default_datastore
        # which is the vm_datastore. Also test volume remove/attach/detach/ls.
        # 2. change the default_datastore to from "VM_DS" to a real datastore, and set the "volume_maxsize" and
        # "usage_quota" to test volume create with different sizes
        # 3. rename the "tenant1" to "new_tenant1", after that, test basic volume create/attach/detach/remove/ls
        logging.info("test_vmdkops_on_tenant_vm")
        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)
        # add vm to tenant
        error_info = auth_api._tenant_vm_add(
                                         name=self.tenant1_name,
                                         vm_list=[self.vm1_name])
        self.assertEqual(None, error_info)

        # run create command, which should succeed
        # "default_datastore" is set to "_VM_DS" for "tenant1"
        # the volume will be created in the datastore where the vm lives in
        # vm lives in self.datastore_name
        # the default volume size is 100MB
        opts={u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        error_info, privileges = auth_api._tenant_access_ls(self.tenant1_name)
        _, rows = vmdkops_admin.generate_tenant_access_ls_rows(privileges, self.tenant1_name)
        print(rows)
        # try to create the volume with full name at self.datastore_name
        # should succeed even only have privilege to "_VM_DS", since
        # the VM is created on self.datastore_name
        full_vol_name = self.tenant1_vol2_name + "@" + self.datastore_name
        opts={u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, full_vol_name, opts)
        self.assertEqual(None, error_info)

        # remove that volume with short name
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.tenant1_vol2_name, opts)
        self.assertEqual(None, error_info)

        # list volumes
        opts = {}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)

        # listVMDK return vol name with the format like vol@datastore
        # "result"" should have one volume:tenant1_vol1
        # the volume is created at the datastore where the VM lives in
        self.assertEqual(1, len(result))
        self.assertEqual("tenant1_vol1@"+self.datastore_name, result[0]['Name'])

        # test attach a volume
        opts={}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_ATTACH, self.tenant1_vol1_name, opts)
        self.assertFalse("Error" in result)

        # test detach a volume
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_DETACH, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        # update access privileges to "_VM_DS"
        # try to change the "usage_quota" which should fail
        # since change "usage_quota" to "_VM_DS" and "_ALL_DS" is not allowed
        volume_maxsize_in_MB = convert.convert_to_MB("500MB")
        volume_totalsize_in_MB = convert.convert_to_MB("1GB")
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=auth_data_const.VM_DS,
                                                 allow_create=True,
                                                 volume_maxsize_in_MB=volume_maxsize_in_MB,
                                                 volume_totalsize_in_MB=volume_totalsize_in_MB)
        self.assertNotEqual(None, error_info)

        # change "default_datastore" for tenant1 to self.datastore_name
        # a full access privilege to self.datastore will be created after this update
        error_info = auth_api._tenant_update(name=self.tenant1_name,
                                             default_datastore=self.datastore_name)
        self.assertEqual(None, error_info)

        # update the access privilege to self.datastore_name
        # to set the "max_vol_size" and "usage_quota"
        volume_maxsize_in_MB = convert.convert_to_MB("500MB")
        volume_totalsize_in_MB = convert.convert_to_MB("1GB")
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=self.datastore_name,
                                                 allow_create=True,
                                                 volume_maxsize_in_MB=volume_maxsize_in_MB,
                                                 volume_totalsize_in_MB=volume_totalsize_in_MB)
        self.assertEqual(None, error_info)

        # create a volume with 600MB which exceed the volume_maxsize
        opts={u'size': u'600MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol2_name, opts)
        self.assertEqual({u'Error': 'Volume size exceeds the max volume size limit'}, error_info)

        # create a volume with 500MB
        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol2_name, opts)
        self.assertEqual(None, error_info)

        # create another volume with 500MB, and total_storeage used by this tenant will exceed volume_totalsize
        opts={u'size': u'500mb', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol3_name, opts)
        self.assertEqual({u'Error': 'The total volume size exceeds the usage quota'}, error_info)

        # set allow_create to False
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=self.datastore_name,
                                                 allow_create=False)
        self.assertEqual(None, error_info)

        # try to delete the first volume, which should fail
        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.tenant1_vol1_name, opts)
        self.assertEqual({u'Error': 'No delete privilege'}, error_info)

        # set allow_create to True
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=self.datastore_name,
                                                 allow_create="True")
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

        # listVMDK return vol name with the format like vol@datastore
        # "result"" should have two volumes :tenant1_vol2@default_datastore, and  tenant1_vol3@datastore
        # now the "default_datastore" for tenant1 is set to "_VM_DS"
        # by default, those volume will be created on the datastore where the VM lives
        # VM lives in self.datastore_name
        self.assertEqual(2, len(result))
        volume_names = ["tenant1_vol2@" + self.datastore_name, "tenant1_vol3@" + self.datastore_name]

        self.assertTrue(test_utils.checkIfVolumeExist(volume_names,result))
        # rename the tenant
        error_info = auth_api._tenant_update(name=self.tenant1_name,
                                             new_name=self.tenant1_new_name)
        self.assertEqual(None, error_info)

        # set the usage quota to 2GB
        volume_totalsize_in_MB = convert.convert_to_MB('2GB')
        error_info = auth_api._tenant_access_set(name=self.tenant1_new_name,
                                                 datastore=self.datastore_name,
                                                 volume_totalsize_in_MB=volume_totalsize_in_MB)
        self.assertEqual(None, error_info)

        # test the vmdkops after rename

        # list volumes
        opts = {}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)

        # listVMDK return vol name with the format like vol@datastore
        # result should have two volumes :tenant1_vol2, and  tenant1_vol3
        self.assertEqual(2, len(result))
        volume_names = ["tenant1_vol2@" + self.datastore_name, "tenant1_vol3@" + self.datastore_name]
        self.assertTrue(test_utils.checkIfVolumeExist(volume_names,result))
        # create a volume(tenant1_vol1_name) with default size
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

        # try to delete the volume again
        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)


    def test_vmdkops_on_different_tenants(self):
        """ Test vmdkops on VMs which belong to different tenant """
        # 1. Two tenants are created with "default_datastore" set to "_VM_DS"
        # 2. For each tenant, update the "default_datastore" to a real datastore (self.datastore), and try to update
        # the privilege to this datastore, test volume with different sizes, run volume ls to make sure each tenant
        # can only see the volumes belong to that tenant
        logging.info("test_vmdkops_on_different_tenants")
        # add vm1 to tenant1
        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)
        error_info = auth_api._tenant_vm_add(
                                         name=self.tenant1_name,
                                         vm_list=[self.vm1_name])
        self.assertEqual(None, error_info)

        # After setup(), the "default_datastore" of tenant1 has been set to "_VM_DS"
        # A full access privilege to datastore "_VM_DS" has been created for tenant1
        # change "default_datastore" for tenant1 to self.datastore_name
        # a full access privilege to self.datastore will be created after this update
        error_info = auth_api._tenant_update(name=self.tenant1_name,
                                             default_datastore=self.datastore_name)
        self.assertEqual(None, error_info)

        # update the access privilege to self.datastore_name
        # to set the "max_vol_size" and "usage_quota"
        volume_maxsize_in_MB = convert.convert_to_MB("500MB")
        volume_totalsize_in_MB = convert.convert_to_MB("2GB")
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=self.datastore_name,
                                                 allow_create=True,
                                                 volume_maxsize_in_MB=volume_maxsize_in_MB,
                                                 volume_totalsize_in_MB=volume_totalsize_in_MB)
        self.assertEqual(None, error_info)

        error_info = auth_api._tenant_access_rm(name=self.tenant1_name,
                                                datastore=auth_data_const.VM_DS)
        self.assertEqual(None, error_info)

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
                                         name=self.tenant2_name,
                                         vm_list=[self.vm2_name])
        self.assertEqual(None, error_info)


         # After setup(), the "default_datastore" of tenant1 has been set to "_VM_DS"
        # A full access privilege to datastore "_VM_DS" has been created for tenant1
        # change "default_datastore" for tenant1 to self.datastore_name
        # a full access privilege to self.datastore will be created after this update
        error_info = auth_api._tenant_update(name=self.tenant2_name,
                                             default_datastore=self.datastore_name)
        self.assertEqual(None, error_info)

        # update the access privilege to self.datastore_name
        # to set the "max_vol_size" and "usage_quota"
        volume_maxsize_in_MB = convert.convert_to_MB("500MB")
        volume_totalsize_in_MB = convert.convert_to_MB("2GB")
        error_info = auth_api._tenant_access_set(name=self.tenant2_name,
                                                 datastore=self.datastore_name,
                                                 allow_create=True,
                                                 volume_maxsize_in_MB=volume_maxsize_in_MB,
                                                 volume_totalsize_in_MB=volume_totalsize_in_MB)
        self.assertEqual(None, error_info)

        error_info = auth_api._tenant_access_rm(name=self.tenant2_name,
                                                datastore=auth_data_const.VM_DS)
        self.assertEqual(None, error_info)

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
        # listVMDK return vol name with the format like vol@datastore
        self.assertEqual(3, len(result))

        volume_names = test_utils.generate_volume_names("tenant1", self.datastore_name, 3)
        self.assertTrue(test_utils.checkIfVolumeExist(volume_names, result))
        # list volumes for tenant2
        opts = {}
        result = vmdk_ops.executeRequest(vm2_uuid, self.vm2_name, self.vm2_config_path, 'list', None, opts)
        # listVMDK return vol name with the format like vol@datastore
        self.assertEqual(3, len(result))
        volume_names = test_utils.generate_volume_names("tenant2", self.datastore_name, 3)
        self.assertTrue(test_utils.checkIfVolumeExist(volume_names,result))


    @unittest.skipIf(len(vmdk_utils.get_datastores()) < 2,
                     "second datastore is not found - skipping this test")
    def test_vmdkops_for_default_datastore(self):
        # 1. Named tenant "tenant1" is created with "default_datastore" set to "VM_DS"
        # 2. Test volume create with short name succeeds with "VM_DS"
        # 3. Modify the privilege "VM_DS" to disallow create, then volume create with short name failed
        # 4. set the "default_datastore" to a real datastore, make sure volume create succeeds and volume is
        # created on the "default_datastore"
        # 5. modify the "default_datastore" to a different real datastore, make sure volume create succeeds and
        # volue is created on the new "default_datastore"

        """ Test vmdkops for default_datastore """
        logging.info("test_vmdkops_for_default_datastore")
        # add vm1 to tenant1, vm1 is created on self.datastore_name
        vm1_uuid =  vmdk_utils.get_vm_uuid_by_name(self.vm1_name)
        error_info = auth_api._tenant_vm_add(
                                         name=self.tenant1_name,
                                         vm_list=[self.vm1_name])
        self.assertEqual(None, error_info)

        # After setup(), the "default_datastore" of tenant1 has been set to "_VM_DS"
        # A full access privilege to datastore "_VM_DS" has been created for tenant1
        # try to create a volume, which should succeed
        # a volume will be tried to create on the default_datastore, which is "_VM_DS"
        opts =  {u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        # Modify the access privilege to "_VM_DS" to disallow create
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=auth_data_const.VM_DS,
                                                 allow_create=False)

        self.assertEqual(None, error_info)
        # create second volume with default size
        # a volume will be tried to create on the default_datastore, which is "_VM_DS"
        # create should fail since on default_datastore, since access privilege to "_VM_DS" does
        # not have "allow_create"" set to True
        opts =  {u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol2_name, opts)
        self.assertNotEqual(None, error_info)

        # delete the access privilege to "_VM_DS"
        # which should fail since the "default_datastore" for tenant1 is still set to "_VM_DS"
        error_info = auth_api._tenant_access_rm(name=self.tenant1_name,
                                                datastore=auth_data_const.VM_DS)
        self.assertNotEqual(None, error_info)

        # update the "default_datastore" for tenant1 to self.datastore_name
        # a full access privileg to self.datastore_name will be created
        error_into = auth_api._tenant_update(name=self.tenant1_name,
                                             default_datastore=self.datastore_name)

        # try to create the volume again, which should succeed, since
        # we create an access privilege to "self.datastore_name" which allowed to create
        opts =  {u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol2_name, opts)
        self.assertEqual(None, error_info)

        # change the "default_datastore" for tenant1 to "self.datastore1_name"
        # it also create full access privilege to "self.datastore1_name"
        # now the volume will be create on "self.datastore1_name"
        error_info = auth_api._tenant_update(name=self.tenant1_name,
                                             default_datastore=self.datastore1_name)
        opts =  {u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol3_name, opts)
        self.assertEqual(None, error_info)

        # list volumes
        opts = {}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)

        # there should be three volumes
        # "tenant1_vol1"" and "tenant1_vol2" which are created on "self.datastore_name""
        # "tenant1_vol3" which is created on "self.datastore1_name"
        self.assertEqual(len(result), 3)
        self.assertEqual("tenant1_vol1@"+self.datastore_name, result[0]['Name'])
        self.assertEqual("tenant1_vol2@"+self.datastore_name, result[1]['Name'])
        self.assertEqual("tenant1_vol3@"+self.datastore1_name, result[2]['Name'])


    @unittest.skipIf(len(vmdk_utils.get_datastores()) < 2,
                     "second datastore is not found - skipping this test")
    def test_vmdkops_for_vm_ds(self):
        # Only privilege to "_VM_DS" present, test volume create and ls
        # Change the privilege to "_VM_DS" to disallow create, test volume create failed
        # Change the privilege to "_VM_DS" to allow create, and set the "max_volsize", make
        # sure can only create volume which size is smaller than "max_volsize", and test volume remove

        logging.info("test_vmdkops_for_vm_ds")
        # add vm1 to tenant1, vm1 is created on self.datastore_name
        vm1_uuid =  vmdk_utils.get_vm_uuid_by_name(self.vm1_name)
        error_info = auth_api._tenant_vm_add(
                                         name=self.tenant1_name,
                                         vm_list=[self.vm1_name])
        self.assertEqual(None, error_info)

        # add vm3 to tenant1, vm3 is created on self.datastore1_name
        vm3_uuid =  vmdk_utils.get_vm_uuid_by_name(self.vm3_name)
        error_info = auth_api._tenant_vm_add(
                                         name=self.tenant1_name,
                                         vm_list=[self.vm3_name])
        #print error_info.msg
        self.assertEqual(None, error_info)

        # After setup(), the "default_datastore" of tenant1 has been set to "_VM_DS"
        # A full access privilege to datastore "_VM_DS" has been created for tenant1
        # try to create a volume from vm1, which should succeed
        # a volume will be tried to create on the default_datastore, which is "_VM_DS"(self.datastore_name)
        opts =  {u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        # try to create a volume from vm3, which should succeed
        # a volume will be tried to create on the default_datastore, which is "_VM_DS"(self.datastore1_name)
        opts =  {u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm3_uuid, self.vm3_name, self.vm3_config_path, auth.CMD_CREATE, self.tenant1_vol2_name, opts)
        self.assertEqual(None, error_info)

        # list volumes
        opts = {}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)

        # there should be three volumes
        # "tenant1_vol1" will be created on "self.datastore_name"
        # "tenant1_vol2" which is created on "self.datastore1_name"
        self.assertEqual(len(result), 2)
        self.assertEqual("tenant1_vol1@"+self.datastore_name, result[0]['Name'])
        self.assertEqual("tenant1_vol2@"+self.datastore1_name, result[1]['Name'])

        # Remove those two volumes
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_REMOVE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        error_info = vmdk_ops.executeRequest(vm3_uuid, self.vm3_name, self.vm3_config_path, auth.CMD_REMOVE, self.tenant1_vol2_name, opts)
        self.assertEqual(None, error_info)

        # Modify the privilege to "_VM_DS" to disallow create
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=auth_data_const.VM_DS,
                                                 allow_create=False)
        self.assertEqual(None, error_info)

        # try to create volume, which should fail
        # since privilege to "_VM_DS" does not allow_create
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertNotEqual(None, error_info)

        # Modify the privilege to "_VM_DS" to allow create and set
        # the vol_maxsize
        volume_maxsize_in_MB = convert.convert_to_MB("500MB")
        error_info = auth_api._tenant_access_set(name=self.tenant1_name,
                                                 datastore=auth_data_const.VM_DS,
                                                 allow_create=True,
                                                 volume_maxsize_in_MB=volume_maxsize_in_MB)
        self.assertEqual(None, error_info)

        # Try to create the volume with size 600MB, which should fail
        opts={u'size': u'600MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertNotEqual(None, error_info)

        # Try to create the volume with size 500MB, which should succeed
        opts={u'size': u'500MB', u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, self.tenant1_vol1_name, opts)
        self.assertEqual(None, error_info)

        # Try to create a volume at self.datastore_name which also the vm_datastore
        # Even if we don't have any privilege to "self.datastore_name"
        # create should succeed since we have privilege to "_VM_DS"
        full_vol_name = self.tenant1_vol2_name + "@" + self.datastore_name
        opts={u'fstype': u'ext4'}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, auth.CMD_CREATE, full_vol_name, opts)
        self.assertEqual(None, error_info)

        # list volumes
        opts = {}
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path, 'list', None, opts)

        # there should be three volumes
        # "tenant1_vol1" will be created on "self.datastore_name"
        # "tenant1_vol2" which is created on "self.datastore_name"
        self.assertEqual(len(result), 2)
        self.assertEqual("tenant1_vol1@"+self.datastore_name, result[0]['Name'])
        self.assertEqual("tenant1_vol2@"+self.datastore_name, result[1]['Name'])


class VmdkTenantPolicyUsageTestCase(unittest.TestCase):
    """ Unit test for VMDK ops for multi-tenancy """
    logging.info("Running VmdkTenantPolicyUsageTestCase")
    default_tenant_vol1_name = "default_tenant_vol_1"
    default_tenant_vol2_name = "default_tenant_vol_2"

    default_tenant_vols = [default_tenant_vol1_name, default_tenant_vol2_name]

    vm1_name = test_utils.generate_test_vm_name()
    vm1 = None
    vm1_config_path = None

    datastore_name = None
    datastore_path = None


    @unittest.skipIf(not vsan_info.get_vsan_datastore(),
                     "VSAN is not found - skipping policy usage tests")
    def setUp(self):
        """ Setup run before each test """
        logging.info("VmdkTenantPolicyUsageTestCase setUp path =%s", path)

        if (not self.datastore_name):
            self.datastore_name = vsan_info.get_vsan_datastore().info.name
            self.datastore_path = vsan_info.get_vsan_datastore().info.url

            if not self.datastore_name:
                logging.error("Cannot find a vsan datastore")
                self.assertFalse(True)

        self.cleanup()
        # get service_instance, and create VMs
        si = vmdk_ops.get_si()
        error, self.vm1 = test_utils.create_vm(si=si,
                                               vm_name=self.vm1_name,
                                               datastore_name=self.datastore_name)
        if error:
            self.assertFalse(True)

        self.vm1_config_path = vmdk_utils.get_vm_config_path(self.vm1_name)
        logging.info("VmdkTenantPolicyUsageTestCase: create vm1 name=%s Done", self.vm1_name)

        test_utils.create_default_tenant_and_privileges(self)

    def tearDown(self):
        """ Cleanup after each test """
        logging.info("VmdkTenantPolicyUsageTestCase  tearDown path")
        self.cleanup()

    def cleanup(self):
        # cleanup existing volume under DEFAULT tenant
        logging.info("VmdkTenantPolicyUsageTestCase cleanup")
        if self.datastore_path:
            default_tenant_path = os.path.join(self.datastore_path, auth_data_const.DEFAULT_TENANT)
            for vol in self.default_tenant_vols:
                vmdk_path = vmdk_utils.get_vmdk_path(default_tenant_path, vol)
                response = vmdk_ops.getVMDK(vmdk_path, vol, self.datastore_name)
                if not "Error" in response:
                    logging.debug("cleanup: remove volume %s", vmdk_path)
                    vmdk_ops.removeVMDK(vmdk_path)

        # remove VM
        si = vmdk_ops.get_si()
        test_utils.remove_vm(si, self.vm1)

    @unittest.skipIf(not vsan_info.get_vsan_datastore(),
                    "VSAN is not found - skipping vsan_info tests")
    def testPolicyUpdateBackupDelete(self):
        self.orig_policy_content = ('(("proportionalCapacity" i0) '
                                     '("hostFailuresToTolerate" i0))')

        self.name = 'good'
        vsan_policy.create(self.name, self.orig_policy_content)

        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)

        # run create command
        opts={'vsan-policy-name': 'good', u'fstype': u'ext4'}

        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                auth.CMD_CREATE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        # delete should fail because policy is in use
        self.assertNotEqual(None, vsan_policy.delete(self.name))

        # Setting an identical policy returns an error msg
        self.assertNotEqual(None, vsan_policy.update('good',
                                    self.orig_policy_content))

        backup_policy_file = vsan_policy.backup_policy_filename(self.name)
        #Ensure there is no backup policy file
        self.assertFalse(os.path.isfile(backup_policy_file))

        # Fail to update because of a bad policy, and ensure there is no backup
        self.assertNotEqual(None, vsan_policy.update('good', 'blah'))
        self.assertFalse(os.path.isfile(backup_policy_file))

        # try to delete the volume again, which should succeed
        # run remove command
        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                auth.CMD_REMOVE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        self.assertEqual(None, vsan_policy.delete(self.name))

    @unittest.skipIf(not vsan_info.get_vsan_datastore(),
                    "VSAN is not found - skipping vsan_info tests")
    def testPolicyUsageCount(self):
        self.orig_policy_content = ('(("proportionalCapacity" i0) '
                                     '("hostFailuresToTolerate" i0))')

        self.name = 'good'
        vsan_policy.create(self.name, self.orig_policy_content)

        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)

        # run create command
        opts={'vsan-policy-name': 'good', u'fstype': u'ext4'}

        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                auth.CMD_CREATE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                auth.CMD_CREATE, self.default_tenant_vol2_name, opts)
        self.assertEqual(None, error_info)

        # Checking the usage count. Should be equal to number of volumes created
        associated_vmdk_count = 0
        for v in vsan_policy.list_volumes_and_policies():
            if v['policy'] == "good":
                associated_vmdk_count = associated_vmdk_count + 1

        self.assertEqual(associated_vmdk_count, 2,
                        "volumes-policy association listing failed")

        # try to delete the volume again, which should succeed
        opts = {}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                auth.CMD_REMOVE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                auth.CMD_REMOVE, self.default_tenant_vol2_name, opts)
        self.assertEqual(None, error_info)

        # Checking the usage count. Should be equal to 0 as the volumes have been deleted
        associated_vmdk_count = 0
        for v in vsan_policy.list_volumes_and_policies():
            if v['policy'] == "good":
                associated_vmdk_count = associated_vmdk_count + 1

        self.assertEqual(associated_vmdk_count, 0,
                        "volumes-policy association listing failed")

        vsan_policy.delete(self.name)

class VMListenerTest(unittest.TestCase):
    """ Unit test for VM listener. """
    logging.info("Running VMListenerTest")
    default_tenant_vol1_name = "stale_volume"

    vm1_name = test_utils.generate_test_vm_name()
    vm1 = None
    vm1_config_path = None

    datastore_name = None
    datastore_path = None

    def setUp(self):
        """ Setup run before each test """
        logging.info("VMListenerTest setUp path =%s", path)

        if not self.datastore_name:
            datastores = vmdk_utils.get_datastores()
            # Use the first datastore out of list of datastore to
            # create VMs and volumes
            datastore = datastores[0]
            if not datastore:
                logging.error("Cannot find a valid datastore")
                self.assertFalse(True)
            self.datastore_name = datastore[0]
            self.datastore_path = datastore[2]

        self.cleanup()
        # get service_instance, and create VMs
        si = vmdk_ops.get_si()
        error, self.vm1 = test_utils.create_vm(si=si,
                                               vm_name=self.vm1_name,
                                               datastore_name=self.datastore_name)
        if error:
            self.assertFalse(True)

        self.vm1_config_path = vmdk_utils.get_vm_config_path(self.vm1_name)
        logging.info("VMListenerTest: create vm1 name=%s Done", self.vm1_name)

        # create DEFAULT tenant DEFAULT privilege if missing
        test_utils.create_default_tenant_and_privileges(self)

    def tearDown(self):
        """ Cleanup after each test """
        logging.info("VMListenerTest tearDown path")
        self.cleanup()

    def cleanup(self):
        # cleanup existing volume under DEFAULT tenant
        logging.info("VMListenerTest cleanup")
        if self.datastore_path:
            default_tenant_path = os.path.join(self.datastore_path, auth_data_const.DEFAULT_TENANT)
            vol = self.default_tenant_vol1_name
            vmdk_path = vmdk_utils.get_vmdk_path(default_tenant_path, vol)
            response = vmdk_ops.getVMDK(vmdk_path, vol, self.datastore_name)
            if not "Error" in response:
                logging.debug("cleanup: remove volume %s", vmdk_path)
                vmdk_ops.removeVMDK(vmdk_path)

    def test_stale_volume_status_update(self):
        """
        This test case creates a volume and attaches it to the VM.
        The VM is then powerd off. At the power off, the VM listener thread
        gets the event and updates the status of the attached volume to detached
        """
        vm1_uuid = vmdk_utils.get_vm_uuid_by_name(self.vm1_name)

        opts={}
        error_info = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                auth.CMD_CREATE, self.default_tenant_vol1_name, opts)
        self.assertEqual(None, error_info)

        # test attach a volume
        result = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                         auth.CMD_ATTACH, self.default_tenant_vol1_name, opts)
        self.assertFalse("Error" in result)

        si = vmdk_ops.get_si()
        test_utils.remove_vm(si, self.vm1, False)

        # VM listener thread waits for update through property collector and updates the status
        # This happens in background so we need to give some time for this sanitization to complete
        time.sleep(5)

        # status of the volume should be updated to "detached"
        response = vmdk_ops.executeRequest(vm1_uuid, self.vm1_name, self.vm1_config_path,
                                         auth.CMD_GET, self.default_tenant_vol1_name, opts)
        self.assertEqual(response[volume_kv.STATUS], volume_kv.DETACHED)

        test_utils.destroy_vm_object(si, self.vm1)

def setUpModule():
    # Let's make sure we are testing a local DB
    os.system(ADMIN_RM_LOCAL_AUTH_DB)
    ret = os.system(ADMIN_INIT_LOCAL_AUTH_DB)
    if ret != 0:
        raise Exception("Failed to initialize local Config DB")

def tearDownModule():
    # clean up Config DB backups
    for f in glob.glob(CONFIG_DB_BAK_GLOB):
        os.remove(f)
    ret = os.system(ADMIN_RM_LOCAL_AUTH_DB)
    if ret != 0:
        raise Exception("Failed to remove local Config DB")

if __name__ == '__main__':
    # configure the log, find the dir and run the tests
    log_config.configure()
    volume_kv.init()

    # Calculate the path, use the first datastore in datastores
    datastores = vmdk_utils.get_datastores()
    path = datastores[0][2]

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
