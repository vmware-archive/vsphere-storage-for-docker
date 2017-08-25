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

import unittest
import sys
import os
import shutil
import uuid
import glob
import vmdk_ops
import vmdk_utils
import volume_kv as kv
import vmdkops_admin
import random
import auth_api
import log_config
import logging
import convert
import error_code
import auth_data_const
import auth_data
import test_utils

# Number of expected columns in ADMIN_CLI ls
EXPECTED_COLUMN_COUNT = 13

# Number of expected columns in "tenant ls"
TENANT_LS_EXPECTED_COLUMN_COUNT = 5

# Number of expected columns in "tenant vm ls"
TENANT_VM_LS_EXPECTED_COLUMN_COUNT = 2

# Number of expected columns in "tenant vm ls"
TENANT_ACCESS_LS_EXPECTED_COLUMN_COUNT = 4

VMGROUP = 'vmgroup'

def convert_to_str(unicode_or_str):
    python_version = sys.version_info.major
    if python_version >= 3:
        return unicode_or_str
    else:
        # convert the input from unicode to str
        return unicode_or_str.encode('utf-8')

def generate_vm_name_str(vms):
    """ Generate a str with concatenation of given list of vm name """
    # vms is a list of vm_name
    # example: vms=["vm1", "vm2"]
    # the return value is a string like this "vm1,vm2""
    res = ""
    for vm in vms:
        # vm[0] is vm_uuid, vm has format (vm_uuid)
        res = res + vm
        res = res + ","

    if res:
        res = res[:-1]

    return res

class TestParsing(unittest.TestCase):
    """ Test command line arg parsing for all commands """

    def setUp(self):
        self.parser = vmdkops_admin.create_parser()

    def test_parse_ls_no_options(self):
        args = self.parser.parse_args('volume ls'.split())
        self.assertEqual(args.func, vmdkops_admin.ls)
        self.assertEqual(args.c, None)

    def test_parse_ls_dash_c(self):
        args = self.parser.parse_args(
            'volume ls -c created-by,created'.split())
        self.assertEqual(args.func, vmdkops_admin.ls)
        self.assertEqual(args.c, ['created-by', 'created'])

    def test_parse_ls_dash_c_invalid_argument(self):
        self.assert_parse_error('volume ls -c personality')

    def test_policy_no_args_fails(self):
        # Py2 argsparse throws in this case, Py3 peacefully shows help
        if sys.version_info[0] == 2:
            self.assert_parse_error('policy')

    def test_policy_create_no_args_fails(self):
        self.assert_parse_error('policy create')

    def test_policy_create(self):
        content = 'some policy content'
        cmd = 'policy create --name=testPolicy --content'.split()
        cmd.append("some policy content")
        args = self.parser.parse_args(cmd)
        self.assertEqual(args.content, 'some policy content')
        self.assertEqual(args.func, vmdkops_admin.policy_create)
        self.assertEqual(args.name, 'testPolicy')

    def test_policy_rm(self):
        args = self.parser.parse_args('policy rm --name=testPolicy'.split())
        self.assertEqual(args.func, vmdkops_admin.policy_rm)
        self.assertEqual(args.name, 'testPolicy')

    def test_policy_rm_no_args_fails(self):
        self.assert_parse_error('policy rm')

    def test_policy_ls(self):
        args = self.parser.parse_args('policy ls'.split())
        self.assertEqual(args.func, vmdkops_admin.policy_ls)

    def test_policy_ls_badargs(self):
        self.assert_parse_error('policy ls --name=yo')

    # NOTE: "tenant" is renamed to "vmgroup", but we only change it in command line
    # all the function name remain unchanged

    def test_tenant_create(self):
        args = self.parser.parse_args((VMGROUP + ' create --name=vmgroup1 --default-datastore=datastore1 --vm-list vm1,vm2').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_create)
        self.assertEqual(args.default_datastore, 'datastore1')
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.vm_list, ['vm1', 'vm2'])

    def test_tenant_create_vm_ds(self):
        args = self.parser.parse_args((VMGROUP + ' create --name=vmgroup1 --default-datastore=_VM_DS --vm-list vm1,vm2').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_create)
        self.assertEqual(args.default_datastore, '_VM_DS')
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.vm_list, ['vm1', 'vm2'])

    def test_tenant_create_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' create')
        # does not set default_datastore
        self.assert_parse_error(VMGROUP + ' create --name=vmgroup1')

    def test_tenant_update(self):
        args = self.parser.parse_args((VMGROUP + ' update --name=vmgroup1 --default-datastore=datastore1 --new-name=new-vmgroup1 --description=new_desc').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_update)
        self.assertEqual(args.default_datastore, 'datastore1')
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.new_name, 'new-vmgroup1')
        self.assertEqual(args.description, 'new_desc')

    def test_tenant_update_missing_name(self):
        self.assert_parse_error(VMGROUP + ' update')

    def test_tenant_rm(self):
        args = self.parser.parse_args((VMGROUP + ' rm --name=vmgroup1 --remove-volumes').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_rm)
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.remove_volumes, True)

    def test_tenant_rm_without_arg_remove_volumes(self):
        args = self.parser.parse_args((VMGROUP + ' rm --name=vmgroup1').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_rm)
        self.assertEqual(args.name, 'vmgroup1')
        # If arg "remove_volumes" is not specified in the CLI, then args.remove_volumes
        # will be None
        self.assertEqual(args.remove_volumes, False)


    def test_tenant_rm_missing_name(self):
        self.assert_parse_error(VMGROUP + ' rm')

    def test_tenant_ls(self):
        args = self.parser.parse_args((VMGROUP + ' ls').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_ls)

    def test_tenant_vm_add(self):
        args = self.parser.parse_args((VMGROUP + ' vm add --name=vmgroup1 --vm-list vm1,vm2').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_vm_add)
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.vm_list, ['vm1', 'vm2'])

    def test_tenant_vm_add_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' vm add')
        self.assert_parse_error(VMGROUP + ' vm add --name=vmgroup1')

    def test_tenant_vm_rm(self):
        args = self.parser.parse_args((VMGROUP + ' vm rm --name=vmgroup1 --vm-list vm1,vm2').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_vm_rm)
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.vm_list, ['vm1', 'vm2'])

    def test_tenant_vm_rm_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' vm add')
        self.assert_parse_error(VMGROUP + ' vm add --name=vmgroup1')

    def test_tenant_vm_ls(self):
        args = self.parser.parse_args((VMGROUP + ' vm ls --name=vmgroup1').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_vm_ls)
        self.assertEqual(args.name, 'vmgroup1')

    def test_tenant_vm_ls_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' vm ls')

    def test_tenant_access_add(self):
        args = self.parser.parse_args((VMGROUP + ' access add --name=vmgroup1 --datastore=datastore1 --allow-create --volume-maxsize=500MB --volume-totalsize=1GB').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_add)
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.datastore, 'datastore1')
        self.assertEqual(args.allow_create, True)
        self.assertEqual(args.volume_maxsize, '500MB')
        self.assertEqual(args.volume_totalsize, '1GB')

    def test_tenant_access_add_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' access add')
        self.assert_parse_error(VMGROUP + ' access add --name=vmgroup1')
        self.assert_parse_error(VMGROUP + ' access add --name=vmgroup1 --default-datastore')

    def test_tenant_access_add_invalid_option_fails(self):
        self.assert_parse_error(VMGROUP + ' access add --name=vmgroup1 --datastore=datastore1 --rights=create mount')

    def test_tenant_access_set(self):
        args = self.parser.parse_args((VMGROUP + ' access set --name=vmgroup1 --datastore=datastore1 --allow-create=True --volume-maxsize=500MB --volume-totalsize=1GB').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_set)
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.datastore, 'datastore1')
        self.assertEqual(args.allow_create, "True")
        self.assertEqual(args.volume_maxsize, '500MB')
        self.assertEqual(args.volume_totalsize, '1GB')

    def test_tenant_accss_set_not_set_allow_create(self):
        args = self.parser.parse_args((VMGROUP + ' access set --name=vmgroup1 --datastore=datastore1 --volume-maxsize=500MB --volume-totalsize=1GB').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_set)
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.datastore, 'datastore1')
        self.assertEqual(args.allow_create, None)
        self.assertEqual(args.volume_maxsize, '500MB')
        self.assertEqual(args.volume_totalsize, '1GB')

    def test_tenant_access_set_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' access set')
        self.assert_parse_error(VMGROUP + ' access set --name=vmgroup1')

    def test_tenant_access_set_invalid_option_fails(self):
        self.assert_parse_error(VMGROUP + ' access set --name=vmgroup1 --datastore=datastore1 --rights=crete,mount')

    def test_tenant_access_rm(self):
        args = self.parser.parse_args((VMGROUP + ' access rm --name=vmgroup1 --datastore=datastore1').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_rm)
        self.assertEqual(args.name, 'vmgroup1')
        self.assertEqual(args.datastore, 'datastore1')

    def test_tenant_access_rm_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' access rm')
        self.assert_parse_error(VMGROUP + ' access rm --name=vmgroup1')

    def test_tenant_access_ls(self):
        args = self.parser.parse_args((VMGROUP + ' access ls --name=vmgroup1').split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_ls)
        self.assertEqual(args.name, 'vmgroup1')

    def test_tenant_access_ls_missing_option_fails(self):
        self.assert_parse_error(VMGROUP + ' access ls')

    def test_status(self):
        args = self.parser.parse_args(['status'])
        self.assertEqual(args.func, vmdkops_admin.status)

    def test_set_no_args(self):
        self.assert_parse_error('set')

    def test_set_no_volname(self):
        self.assert_parse_error('set --options="access=read-only"')

    def test_set_invalid_options(self):
        self.assert_parse_error('set --options="size=10gb"')
        self.assert_parse_error('set --options="acces=read-write"')
        self.assert_parse_error('set --options="attach-as=persisten"')

    def test_set_no_options(self):
        self.assert_parse_error('set --volume=volume_name')

    def test_set(self):
        args = self.parser.parse_args('volume set --volume=vol_name@datastore --vmgroup=vmgroup1 --options="access=read-only"'.split())
        self.assertEqual(args.func, vmdkops_admin.set_vol_opts)
        self.assertEqual(args.volume, 'vol_name@datastore')
        self.assertEqual(args.vmgroup, 'vmgroup1')
        self.assertEqual(args.options, '"access=read-only"')

        args = self.parser.parse_args('volume set --volume=vol_name@datastore --vmgroup=vmgroup1 --options="attach-as=persistent"'.split())
        self.assertEqual(args.func, vmdkops_admin.set_vol_opts)
        self.assertEqual(args.volume, 'vol_name@datastore')
        self.assertEqual(args.vmgroup, 'vmgroup1')
        self.assertEqual(args.options, '"attach-as=persistent"')

        args = self.parser.parse_args('volume set --volume=vol_name@datastore --vmgroup=vmgroup1 --options="attach-as=independent_persistent"'.split())
        self.assertEqual(args.func, vmdkops_admin.set_vol_opts)
        self.assertEqual(args.volume, 'vol_name@datastore')
        self.assertEqual(args.vmgroup, 'vmgroup1')
        self.assertEqual(args.options, '"attach-as=independent_persistent"')

    def test_config_parse(self):
        '''Validate that the parser accepts the known config commands'''
        valid_commands = [
            'config init --datastore=store1',
            'config init --datastore=store1 --force',
            'config rm --local --confirm',
            'config rm --no-backup --confirm --local',
            'config mv --to=datastore'
        ]
        for cmd in valid_commands:
            args = self.parser.parse_args(cmd.split())


    def test_config_parse_fail(self):
        '''Expected failures in config command parse'''
        invalid_commands = ['status --all',
                            'config init --confirm',
                            'config rm --no-backup --datastore=DS1',
                            'config mv --force']
        for cmd in invalid_commands:
            self.assert_parse_error(cmd)


    # Usage is always printed on a parse error. It's swallowed to prevent clutter.
    def assert_parse_error(self, command):
        with open('/dev/null', 'w') as f:
            sys.stdout = f
            sys.stderr = f
            with self.assertRaises(SystemExit):
                args = self.parser.parse_args(command.split())
            sys.stdout = sys.__stdout__
            sys.stderr = sys.__stderr__


class TestLs(unittest.TestCase):
    """ Test ls functionality """

    def setUp(self):
        """ Setup run before each test """
        self.vol_count = 0
        self.cleanup()
        for (datastore, url, path) in vmdk_utils.get_datastores():
            if not self.mkdir(path):
                continue
            for id in range(5):
                volName = 'testvol' + str(id)
                fullpath = os.path.join(path, volName + '.vmdk')
                self.assertEqual(None,
                                 vmdk_ops.createVMDK(vm_name='test-vm',
                                                     vmdk_path=fullpath,
                                                     vol_name=volName))
                self.vol_count += 1

    def tearDown(self):
        """ Cleanup after each test """
        self.cleanup()

    def mkdir(self, path):
        """ Create a directory if it doesn't exist. Returns pathname or None. """
        if not os.path.isdir(path):
            try:
                os.mkdir(path)
            except OSError as e:
                return None
        return path

    def cleanup(self):
        for v in self.get_testvols():
            self.assertEqual(
                None,
                vmdk_ops.removeVMDK(os.path.join(v['path'], v['filename'])))

    def get_testvols(self):
        return [x
                for x in vmdk_utils.get_volumes(None)
                if x['filename'].startswith('testvol')]

    def test_ls_helpers(self):
        volumes = self.get_testvols()
        self.assertEqual(len(volumes), self.vol_count)
        for v in volumes:
            metadata = vmdkops_admin.get_metadata(os.path.join(v['path'], v[
                'filename']))
            self.assertNotEqual(None, metadata)

    def test_ls_no_args(self):
        volumes = vmdk_utils.get_volumes(None)
        header = vmdkops_admin.all_ls_headers()
        rows = vmdkops_admin.generate_ls_rows(None)
        self.assertEqual(EXPECTED_COLUMN_COUNT, len(header))
        self.assertEqual(len(volumes), len(rows))
        for i in range(len(volumes)):
            self.assertEqual(volumes[i]['filename'], rows[i][0] + '.vmdk')

class TestSet(unittest.TestCase):
    """ Test set functionality """

    def setUp(self):
        """ Setup run before each test """
        self.vol_count = 0
        self.cleanup()
        for (datastore, url, path) in vmdk_utils.get_datastores():
            if not self.mkdir(path):
                continue
            for id in range(5):
                volName = 'testvol' + str(id)
                fullpath = os.path.join(path, volName + '.vmdk')
                self.assertEqual(None,
                                 vmdk_ops.createVMDK(vm_name='test-vm',
                                                     vmdk_path=fullpath,
                                                     vol_name=volName))
                self.vol_count += 1

    def tearDown(self):
        """ Cleanup after each test """
        self.cleanup()

    def mkdir(self, path):
        """ Create a directory if it doesn't exist. Returns pathname or None. """
        if not os.path.isdir(path):
            try:
                os.mkdir(path)
            except OSError as e:
                return None
        return path

    def cleanup(self):
        for v in self.get_testvols():
            self.assertEqual(
                None,
                vmdk_ops.removeVMDK(os.path.join(v['path'], v['filename'])))

    def get_testvols(self):
        return [x
                for x in vmdk_utils.get_volumes(None)
                if x['filename'].startswith('testvol')]

    def test_set_attach_as(self):
        volumes = self.get_testvols()
        self.assertEqual(len(volumes), self.vol_count)
        for v in volumes:
            attach_as_opt = random.choice(kv.ATTACH_AS_TYPES)
            # generate string like "testvol0@datastore1"
            vol_arg = '@'.join([v['filename'].replace('.vmdk', ''), v['datastore']])

            attach_as_arg = 'attach-as={}'.format(attach_as_opt)
            set_ok = vmdk_ops.set_vol_opts(name=vol_arg,
                                           tenant_name=None,
                                           options=attach_as_arg)
            self.assertTrue(set_ok)

            metadata = vmdkops_admin.get_metadata(os.path.join(v['path'], v[
                'filename']))
            self.assertNotEqual(None, metadata)

            curr_attach_as = vmdkops_admin.get_attach_as(metadata)
            self.assertEqual(attach_as_opt, curr_attach_as)

    def test_set_access(self):
        volumes = self.get_testvols()
        self.assertEqual(len(volumes), self.vol_count)
        for v in volumes:
            access_opt = random.choice(kv.ACCESS_TYPES)
            # generate string like "testvol0@datastore1"
            vol_arg = '@'.join([v['filename'].replace('.vmdk', ''), v['datastore']])
            access_arg = 'access={}'.format(access_opt)
            set_ok = vmdk_ops.set_vol_opts(name=vol_arg,
                                           tenant_name=None,
                                           options=access_arg)
            self.assertTrue(set_ok)

            metadata = vmdkops_admin.get_metadata(os.path.join(v['path'], v[
                'filename']))
            self.assertNotEqual(None, metadata)

            curr_access = vmdkops_admin.get_access(metadata)
            self.assertEqual(access_opt, curr_access)

class TestStatus(unittest.TestCase):
    """ Test status functionality """
    def test_status(self):
        args = vmdkops_admin.create_parser().parse_args("status".split())
        self.assertEqual(vmdkops_admin.status(args), None)

class TestTenant(unittest.TestCase):
    """
        Test tenant functionality
    """

    # NOTE:We rename "tenant" to "vmgroup", but we do not plan to
    # change the name used in the following test
    # only the command itself will be changed from "tenantxxx" to "vmgroup xxx"

    # The following tests are covered:
    # 1. tenant command
    # Test tenant create, tenant ls and tenant rm
    # test tenant create with "default_datastore" set to an invalid datastore name, "_VM_DS" and "_ALL_DS"
    # tenant update to update tenant description and default_datastore
    # tenant update to rename a tenant
    # 2. tenant vm command
    # tenant vm add , tenant vm rm and tenant vm ls
    # 3. tenant access command
    # tenant access create, tenant access ls and tenant access rm
    # tenant access set command to update allow_create, volume_maxsize and volume_totalsize
    # tenant access rm command to remove a privilege to default_datastore failed as expect
    # Test are positive, no negative testing is done here.

    # tenant1 info
    tenant1_name = "test_tenant1"
    random_id = random.randint(0, 65536)
    vm1_name = 'test_vm1_' + str(random_id)
    vm1 = None
    random_id = random.randint(0, 65536)
    vm2_name = 'test_vm2_' + str(random_id)
    vm2 = None
    tenant1_new_name = "new_test_tenant1"
    datastore_name = None
    datastore1_name = None
    bad_datastore_name = "__BAD_DS"

    def setUp(self):
        """ Setup run before each test """

        if (not self.datastore_name):
            datastores = vmdk_utils.get_datastores()
            if datastores:
                datastore = datastores[0]
                self.datastore_name = datastore[0]
                self.datastore_path = datastore[2]

                if len(datastores) > 1:
                    datastore1 = datastores[1]
                    self.datastore1_name = datastore1[0]
                    self.datastoer1_path = datastore[2]

            else:
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

        logging.info("TestTenant: create vm1 name=%s Done", self.vm1_name)

        error, self.vm2 = test_utils.create_vm(si=si,
                                    vm_name=self.vm2_name,
                                    datastore_name=self.datastore_name)
        if error:
            self.assertFalse(True)
        self.vm2_config_path = vmdk_utils.get_vm_config_path(self.vm2_name)

        logging.info("TestTenant: create vm2 name=%s Done", self.vm2_name)


    def tearDown(self):
        """ Cleanup after each test """
        self.cleanup()

    def cleanup(self):
        # cleanup existing tenants
        test_utils.cleanup_tenant(self.tenant1_name)
        test_utils.cleanup_tenant(self.tenant1_new_name)

        # remove VM
        si = vmdk_ops.get_si()
        test_utils.remove_vm(si, self.vm1)
        test_utils.remove_vm(si, self.vm2)

    def test_tenant(self):
        """ Test AdminCLI command for tenant management """
        # create tenant1
        vm_list = [self.vm1_name]

        # create tenant and set "default_datastore" to an invalide datastore_name
        # should fail since the "default_datastore" is not valid
        error_info, tenant = auth_api._tenant_create(
                                                    name=self.tenant1_name,
                                                    default_datastore=self.bad_datastore_name,
                                                    description="Test tenant1",
                                                    vm_list=vm_list,
                                                    privileges=[])
        self.assertNotEqual(None, error_info)

        # try to create the tenant and set "default_datastore" to "_ALL_DS"
        # should fail since "_ALL_DS" cannot be set as "default_datastore"
        error_info, tenant = auth_api._tenant_create(
                                                    name=self.tenant1_name,
                                                    default_datastore=self.bad_datastore_name,
                                                    description="Test tenant1",
                                                    vm_list=vm_list,
                                                    privileges=[])
        self.assertNotEqual(None, error_info)

        # try to create the tenant and set "default_datastore" to "_VM_DS"
        # should succeed
        error_info, tenant = auth_api._tenant_create(
                                                    name=self.tenant1_name,
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description="Test tenant1",
                                                    vm_list=vm_list,
                                                    privileges=[])
        self.assertEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        _, rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)
        self.assertEqual(len(header), TENANT_LS_EXPECTED_COLUMN_COUNT)

        # Two tenants in the list, "_DEFAULT" and "test_tenant1"
        # rows[0] is for "_DEFAULT" tenant, and rows[1] is for "test_tenant1"
        self.assertEqual(len(rows), 2)

        # There are 5 columns for each row, the name of the columns are
        # "Uuid", "Name", "Description", "Default_datastore", "VM_list"
        # Sample output of rows[1]:
        # [u'9e1be0ce-3d58-40f6-a335-d6e267e34baa', u'test_tenant1', u'Test tenant1', '', 'test_vm1']
        expected_output = [self.tenant1_name,
                           "Test tenant1",
                           auth_data_const.VM_DS,
                           generate_vm_name_str(vm_list)]
        actual_output = [convert_to_str(rows[1][1]),
                         convert_to_str(rows[1][2]),
                         convert_to_str(rows[1][3]),
                         convert_to_str(rows[1][4])
                        ]

        self.assertEqual(expected_output, actual_output)

        # tenant update to update description and default_datastore
        # update default_datastore to self.datastore_name, which should succeed
        error_info = auth_api._tenant_update(
                                             name=self.tenant1_name,
                                             description="This is test tenant1",
                                             default_datastore=self.datastore_name)
        self.assertEqual(None, error_info)


        # update default_datastore to "_ALL_DS", which should fail
        error_info = auth_api._tenant_update(
                                             name=self.tenant1_name,
                                             default_datastore=auth_data_const.ALL_DS)
        self.assertNotEqual(None, error_info)

        # list the access privilege for tenant1
        # now should have two access privileges
        # to datastore "_VM_DS" and self.datastore
        error_info, privileges = auth_api._tenant_access_ls(self.tenant1_name)
        self.assertEqual(None, error_info)
        _, rows = vmdkops_admin.generate_tenant_access_ls_rows(privileges, self.tenant1_name)

        self.assertEqual(len(rows), 2)
        if rows[0][0] == self.datastore_name:
            expected_output = [
                               [self.datastore_name, 'True', 'Unset', 'Unset'],
                               [auth_data_const.VM_DS, 'True', 'Unset', 'Unset']
                              ]
        else:
            expected_output = [
                               [auth_data_const.VM_DS, 'True', 'Unset', 'Unset'],
                               [self.datastore_name, 'True', 'Unset', 'Unset']
                              ]

        actual_output = [
                         [rows[0][0], rows[0][1], rows[0][2], rows[0][3]],
                         [rows[1][0], rows[1][1], rows[1][2], rows[1][3]]
                        ]

        self.assertEqual(expected_output, actual_output)

        # remove access privilege to "_VM_DS"
        error_info = auth_api._tenant_access_rm(name=self.tenant1_name,
                                                datastore=auth_data_const.VM_DS)
        self.assertEqual(None, error_info)

        # now, should only have one access privilege
        error_info, privileges = auth_api._tenant_access_ls(self.tenant1_name)
        self.assertEqual(None, error_info)
        _, rows = vmdkops_admin.generate_tenant_access_ls_rows(privileges, self.tenant1_name)
        self.assertEqual(len(rows), 1)

        expected_output = [self.datastore_name, 'True', 'Unset', 'Unset']
        actual_output = [rows[0][0], rows[0][1], rows[0][2], rows[0][3]]
        self.assertEqual(expected_output, actual_output)

        # update default vmgroup description
        error_info = auth_api._tenant_update(name=auth_data_const.DEFAULT_TENANT,
                                             description="This is the default vmgroup")

        self.assertEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        _, rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)

        expected_output = [self.tenant1_name,
                           "This is test tenant1",
                           self.datastore_name,
                           generate_vm_name_str(vm_list)]
        actual_output = [convert_to_str(rows[1][1]),
                         convert_to_str(rows[1][2]),
                         convert_to_str(rows[1][3]),
                         convert_to_str(rows[1][4])
                        ]

        self.assertEqual(expected_output, actual_output)

        # tenant update to rename the tenant
        error_info = auth_api._tenant_update(
            name=self.tenant1_name,
            new_name=self.tenant1_new_name)
        self.assertEqual(None, error_info)

        # verify default vmgroup can't be renamed
        error_info  = auth_api._tenant_update(name=auth_data_const.DEFAULT_TENANT,
                                              new_name=self.tenant1_new_name)
        self.assertNotEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        _, rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)

        expected_output = [self.tenant1_new_name,
                           "This is test tenant1",
                           self.datastore_name,
                           generate_vm_name_str(vm_list)]
        actual_output = [convert_to_str(rows[1][1]),
                         convert_to_str(rows[1][2]),
                         convert_to_str(rows[1][3]),
                         convert_to_str(rows[1][4])
                         ]


        # tenant rm to remove the tenant
        error_info = auth_api._tenant_vm_rm(name=self.tenant1_new_name,
                                            vm_list=vm_list)
        self.assertEqual(None, error_info)

        error_info = test_utils.cleanup_tenant(self.tenant1_new_name)
        self.assertEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        _, rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)

        # right now, should only have 1 tenant, which is "_DEFAULT" tenant
        self.assertEqual(len(rows), 1)

    def test_tenant_vm(self):
        """ Test AdminCLI command for tenant vm management """
        logging.debug("test_tenant_vm")
        # create tenant1 without adding any vms and privilege
        error_info, tenant = auth_api._tenant_create(
                                                    name=self.tenant1_name,
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description="Test tenant1",
                                                    vm_list=[],
                                                    privileges=[])
        self.assertEqual(None, error_info)

        error_info, vms = auth_api._tenant_vm_ls(self.tenant1_name)
        self.assertEqual(None, error_info)

        headers = vmdkops_admin.tenant_vm_ls_headers()
        rows = vmdkops_admin.generate_tenant_vm_ls_rows(vms)

        self.assertEqual(len(headers), TENANT_VM_LS_EXPECTED_COLUMN_COUNT)
        expected_output = []
        actual_output = rows
        self.assertEqual(expected_output, actual_output)

        # Trying to create tenant with duplicate vm names
        error_info, tenant_dup = auth_api._tenant_create(
                                                    name="tenant_add_dup_vms",
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description="Tenant with duplicate VMs",
                                                    vm_list=[self.vm1_name, self.vm1_name],
                                                    privileges=[])

        self.assertEqual(error_code.ErrorCode.VM_DUPLICATE, error_info.code)

        # tenant vm add to add two VMs to the tenant
        error_info = auth_api._tenant_vm_add(
                                             name=self.tenant1_name,
                                             vm_list=[self.vm1_name, self.vm2_name])
        self.assertEqual(None, error_info)

        error_info, vms = auth_api._tenant_vm_ls(self.tenant1_name)
        self.assertEqual(None, error_info)

        # create tenant2 with vm1 a part of it. Should fail as VM can be a part
        # of just one tenant
        error_info, tenant2 = auth_api._tenant_create(
                                                    name="Test_tenant2",
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description="Test_tenant2",
                                                    vm_list=[self.vm1_name],
                                                    privileges=[])
        self.assertEqual(error_code.ErrorCode.VM_IN_ANOTHER_TENANT, error_info.code)

        # create tenant3 and then try to add vm1 to it which is a part of
        # another tenant. Should fail as VM can be a part of just one tenant
        error_info, tenant3 = auth_api._tenant_create(
                                                    name="Test_tenant3",
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description="Test_tenant3",
                                                    vm_list=[],
                                                    privileges=[])
        self.assertEqual(None, error_info)

        error_info = auth_api._tenant_vm_add(
                                              name=tenant3.name,
                                              vm_list=[self.vm1_name])

        self.assertEqual(error_code.ErrorCode.VM_IN_ANOTHER_TENANT, error_info.code)

        # Replace should fail since vm1 is already a part of tenant1
        error_info = auth_api._tenant_vm_replace(
                                              name=tenant3.name,
                                              vm_list=[self.vm1_name])
        self.assertEqual(error_code.ErrorCode.VM_IN_ANOTHER_TENANT, error_info.code)

        # remove the tenant3
        error_info = test_utils.cleanup_tenant(name=tenant3.name)
        self.assertEqual(None, error_info)

        # There are 2 columns for each row, the name of the columns are
        # "Uuid", "Name"
        # Sample output of a row:
        # [u'564d2b7d-187c-eaaf-60bc-e015b5cdc3eb', 'test_vm1']
        rows = vmdkops_admin.generate_tenant_vm_ls_rows(vms)
        # Two vms are associated with this tenant
        self.assertEqual(len(rows), 2)

        expected_output = [self.vm1_name, self.vm2_name]
        actual_output = [rows[0][1],
                         rows[1][1]]
        self.assertEqual(expected_output, actual_output)

        # tenant vm rm to remove one VM from the tenant
        error_info = auth_api._tenant_vm_rm(
                                             name=self.tenant1_name,
                                             vm_list=[self.vm2_name])
        self.assertEqual(None, error_info)

        error_info, vms = auth_api._tenant_vm_ls(self.tenant1_name)
        self.assertEqual(None, error_info)

        rows = vmdkops_admin.generate_tenant_vm_ls_rows(vms)

        # tenant should only have one VM now
        self.assertEqual(len(rows), 1)

        expected_output = [self.vm1_name]
        actual_output = [rows[0][1]]
        self.assertEqual(expected_output, actual_output)

    def test_tenant_access(self):
        """ Test AdminCLI command for tenant access management """

        # create tenant1 without adding any vms and privileges
        # the "default_datastore" will be set to "_VM_DS"
        # a full access privilege will be created for tenant1
        error_info, tenant = auth_api._tenant_create(
                                                    name=self.tenant1_name,
                                                    default_datastore=auth_data_const.VM_DS,
                                                    description="Test tenant1",
                                                    vm_list=[self.vm1_name],
                                                    privileges=[])
        self.assertEqual(None, error_info)

        # now, should only have privilege to ""_VM_DS
        error_info, privileges = auth_api._tenant_access_ls(self.tenant1_name)
        _, rows = vmdkops_admin.generate_tenant_access_ls_rows(privileges, self.tenant1_name)
        self.assertEqual(1, len(rows))
        expected_output = [auth_data_const.VM_DS,
                            "True",
                            "Unset",
                            "Unset"]
        actual_output = [rows[0][0],
                         rows[0][1],
                         rows[0][2],
                         rows[0][3]]
        self.assertEqual(expected_output, actual_output)

        # remove the privilege to "_VM_DS", which should fail
        # since "_VM_DS still the "default_datastore" for tenant1
        error_info = auth_api._tenant_access_rm(name=self.tenant1_name,
                                                datastore=auth_data_const.VM_DS
                                                )
        self.assertNotEqual(None, error_info)

        # add a access privilege for tenant to self.datastore_name
        # allow_create = False
        # max_volume size = 600MB
        # total_volume size = 1GB
        volume_maxsize_in_MB = convert.convert_to_MB("600MB")
        volume_totalsize_in_MB = convert.convert_to_MB("1GB")
        error_info = auth_api._tenant_access_add(name=self.tenant1_name,
                                                 datastore=self.datastore_name,
                                                 allow_create=False,
                                                 volume_maxsize_in_MB=volume_maxsize_in_MB,
                                                 volume_totalsize_in_MB=volume_totalsize_in_MB
                                             )
        self.assertEqual(None, error_info)

        # update the "default_datastore" to self.datastore_name
        error_info = auth_api._tenant_update(name=self.tenant1_name,
                                             default_datastore=self.datastore_name)
        self.assertEqual(None, error_info)

        # try to remove the privilege to "_VM_DS" again, which should not fail
        # since the "default_datastore" for tenant1 is set to self.datastore_name
        error_info = auth_api._tenant_access_rm(name=self.tenant1_name,
                                                datastore=auth_data_const.VM_DS
                                                )
        self.assertEqual(None, error_info)

        error_info, privileges = auth_api._tenant_access_ls(self.tenant1_name)
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_access_ls_headers()
        # There are 4 columns for each row, the name of the columns are
        # "Datastore", "Allow_create", "Max_volume_size", "Total_size"
        # Sample output of a row:
        # ['datastore1', 'False', '600.00MB', '1.00GB']
        _, rows = vmdkops_admin.generate_tenant_access_ls_rows(privileges, self.tenant1_name)
        self.assertEqual(len(header), TENANT_ACCESS_LS_EXPECTED_COLUMN_COUNT)

        # tenant aceess privilege should only have a row now
        self.assertEqual(len(rows), 1)

        expected_output = [self.datastore_name,
                           "False",
                           "600.00MB",
                           "1.00GB"]
        actual_output = [rows[0][0],
                         rows[0][1],
                         rows[0][2],
                         rows[0][3]]
        self.assertEqual(expected_output, actual_output)

        # update the access privileges
        # change allow_create to True
        # change max_volume size to 1000MB
        # change total_volume size to 2GB
        volume_maxsize_in_MB = convert.convert_to_MB("1000MB")
        volume_totalsize_in_MB = convert.convert_to_MB("3GB")

        self.parser = vmdkops_admin.create_parser()

        privilege_test_info = [
            ["False", "False"],
            ["FALSE", "False"],
            ["false", "False"],
            ["True", "True"],
            ["TRUE", "True"],
            ["true", "True"],
        ]

        for val in privilege_test_info:
            command = ("vmgroup access set --name={0} ".format(self.tenant1_name))
            command += ("--datastore={0} ".format(self.datastore_name))
            command += ("--allow-create={0} ".format(val[0]))
            command += ("--volume-maxsize=500MB --volume-totalsize=1GB")

            args = self.parser.parse_args(command.split())
            error_info = vmdkops_admin.tenant_access_set(args)
            self.assertEqual(None, error_info)

            error_info, privileges = auth_api._tenant_access_ls(self.tenant1_name)
            self.assertEqual(None, error_info)

            _, rows = vmdkops_admin.generate_tenant_access_ls_rows(privileges, self.tenant1_name)
            self.assertEqual(len(rows), 1)

            expected_output = [self.datastore_name,
                            val[1],
                            "500.00MB",
                            "1.00GB"]
            actual_output = [rows[0][0],
                            rows[0][1],
                            rows[0][2],
                            rows[0][3]]
            self.assertEqual(expected_output, actual_output)

        print("[Negative test case]: Expected invalid values for allow-create option")
        for val in ["INVALID", ""]:
            command = ("vmgroup access set --name={0} ".format(self.tenant1_name))
            command += ("--datastore={0} ".format(self.datastore_name))
            command += ("--allow-create={0} ".format(val))
            command += ("--volume-maxsize=500MB --volume-totalsize=1GB")

            args = self.parser.parse_args(command.split())
            error_info = vmdkops_admin.tenant_access_set(args)
            expected_message = "ERROR:Invalid value {0} for allow-create option".format(val)
            self.assertEqual(expected_message, error_info)

        if self.datastore1_name:
            # second datastore is available, can test tenant update with  --default_datastore
            error_info, tenant_list = auth_api._tenant_ls()
            self.assertEqual(None, error_info)

            # Two tenants in the list, "_DEFAULT" and "test_tenant1"
            # rows[0] is for "_DEFAULT" tenant, and rows[1] is for "test_tenant1"

            # There are 5 columns for each row, the name of the columns are
            # "Uuid", "Name", "Description", "Default_datastore", "VM_list"
            # Sample output of one row:
            # [u'9e1be0ce-3d58-40f6-a335-d6e267e34baa', u'test_tenant1', u'Test tenant1', '', 'test_vm1']
            _, rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)

            # get "default_datastore" from the output
            actual_output = rows[1][3]
            expected_output = self.datastore_name
            self.assertEqual(expected_output, actual_output)

            # add access privilege to self.datastore1_name
            volume_maxsize_in_MB = convert.convert_to_MB("600MB")
            volume_totalsize_in_MB = convert.convert_to_MB("1GB")
            error_info = auth_api._tenant_access_add(name=self.tenant1_name,
                                                     datastore=self.datastore1_name,
                                                     allow_create=False,
                                                     volume_maxsize_in_MB=volume_maxsize_in_MB,
                                                     volume_totalsize_in_MB=volume_totalsize_in_MB)
            self.assertEqual(None, error_info)

            # update the "default_datastore"
            error_info = auth_api._tenant_update(name=self.tenant1_name,
                                                 default_datastore=self.datastore1_name)
            self.assertEqual(None, error_info)

            error_info, tenant_list = auth_api._tenant_ls()
            self.assertEqual(None, error_info)

            _, rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)
            # get "default_datastore" from the output
            actual_output = rows[1][3]
            expected_output = self.datastore1_name
            self.assertEqual(expected_output, actual_output)

            # switch the "default_datastore" to self.datastore_name
            error_info = auth_api._tenant_update(name=self.tenant1_name,
                                                 default_datastore=self.datastore_name)
            self.assertEqual(error_info, None)
            # remove the privilege to self.datastore1_name
            error_info = auth_api._tenant_access_rm(name=self.tenant1_name,
                                                    datastore=self.datastore1_name)
            self.assertEqual(error_info, None)

        # remove access privileges, which should fail
        # since the "default_datastore" is set to self.datastore_name
        # cannot remove the privilege
        error_info = auth_api._tenant_access_rm(name=self.tenant1_name,
        datastore=self.datastore_name)

        self.assertNotEqual(None, error_info)

        error_info, privileges = auth_api._tenant_access_ls(self.tenant1_name)
        self.assertEqual(None, error_info)

        # now, only have a privilege to self.datastore_name
        _, rows = vmdkops_admin.generate_tenant_access_ls_rows(privileges, self.tenant1_name)
        self.assertEqual(1, len(rows))
        expected_output = [self.datastore_name,
                            "True",
                            "500.00MB",
                            "1.00GB"]
        actual_output = [rows[0][0],
                         rows[0][1],
                         rows[0][2],
                         rows[0][3]]
        self.assertEqual(expected_output, actual_output)

    def test_tenant_vm_for_default_tenant(self):
        """ Test AdminCLI vmgroup vm management for _DEFAULT vmgroup """
        logging.debug("Test vm add for _DEFAULT vmgroup")
        # Test "vm add" for _DEFAULT tenant, which should fail
        error_info = auth_api._tenant_vm_add(name=auth_data_const.DEFAULT_TENANT,
                                             vm_list=[self.vm1_name])
        self.assertEqual(error_code.ErrorCode.FEATURE_NOT_SUPPORTED, error_info.code)

        # Test "vm rm" for _DEFAULT tenant, which should fail
        error_info = auth_api._tenant_vm_add(name=auth_data_const.DEFAULT_TENANT,
                                             vm_list=[self.vm1_name])
        self.assertEqual(error_code.ErrorCode.FEATURE_NOT_SUPPORTED, error_info.code)

        # Test "vm add" for _DEFAULT tenant, which should fail
        error_info = auth_api._tenant_vm_add(name=auth_data_const.DEFAULT_TENANT,
                                             vm_list=[self.vm1_name])
        self.assertEqual(error_code.ErrorCode.FEATURE_NOT_SUPPORTED, error_info.code)


class TestConfig(unittest.TestCase):
    """ Test 'config' functionality """

    def __init__(self, *args, **kwargs):
        super(TestConfig, self).__init__(*args, **kwargs)
        self.parser = vmdkops_admin.create_parser()

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def test_config(self):
        '''Config testing'''

        # TBD: Init config and check status - should be NotConfigured
        args = self.parser.parse_args('config rm --local --confirm'.split())
        self.assertEqual(vmdkops_admin.config_rm(args), None)

        args = self.parser.parse_args('config init --local'.split())
        self.assertEqual(vmdkops_admin.config_init(args), None)
        # init
        # check status - should be MultiNode
        # init - should fail
        # init -f should succeed

if __name__ == '__main__':
    kv.init()
    log_config.configure()
    unittest.main()
