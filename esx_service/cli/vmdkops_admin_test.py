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
import vmdk_ops_test
import auth_api

# Number of expected columns in ADMIN_CLI ls
EXPECTED_COLUMN_COUNT = 12

# Number of expected columns in "tenant ls"
TENANT_LS_EXPECTED_COLUMN_COUNT = 5

def str_to_unicode(str):
    """ convert str to unicode """
    return str.decode('utf-8')

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
        args = self.parser.parse_args(['ls'])
        self.assertEqual(args.func, vmdkops_admin.ls)
        self.assertEqual(args.c, None)

    def test_parse_ls_dash_c(self):
        args = self.parser.parse_args(
            'ls -c created-by,created'.split())
        self.assertEqual(args.func, vmdkops_admin.ls)
        self.assertEqual(args.c, ['created-by', 'created'])

    def test_parse_ls_dash_c_invalid_argument(self):
        self.assert_parse_error('ls -c personality')

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
        args = self.parser.parse_args('policy rm testPolicy'.split())
        self.assertEqual(args.func, vmdkops_admin.policy_rm)
        self.assertEqual(args.name, 'testPolicy')

    def test_policy_rm_no_args_fails(self):
        self.assert_parse_error('policy rm')

    def test_policy_ls(self):
        args = self.parser.parse_args('policy ls'.split())
        self.assertEqual(args.func, vmdkops_admin.policy_ls)

    def test_policy_ls_badargs(self):
        self.assert_parse_error('policy ls --name=yo')
        
    def test_tenant_create(self):
        args = self.parser.parse_args('tenant create --name=tenant1 --vm-list vm1,vm2'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_create)
        self.assertEqual(args.name, 'tenant1')
        self.assertEqual(args.vm_list, ['vm1', 'vm2'])

    def test_tenant_create_missing_option_fails(self):
        self.assert_parse_error('tenant create')

    def test_tenant_rm(self):
        args = self.parser.parse_args('tenant rm --name=tenant1 --remove-volumes'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_rm)
        self.assertEqual(args.name, 'tenant1')
        self.assertEqual(args.remove_volumes, True)

    def test_tenant_rm_without_arg_remove_volumes(self):
        args = self.parser.parse_args('tenant rm --name=tenant1'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_rm)
        self.assertEqual(args.name, 'tenant1')
        # If arg "remove_volumes" is not specified in the CLI, then args.remove_volumes
        # will be None
        self.assertEqual(args.remove_volumes, False)
        

    def test_tenant_rm_missing_name(self):
        self.assert_parse_error('tenant rm')

    def test_tenant_ls(self):
        args = self.parser.parse_args('tenant ls'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_ls)
    
    def test_tenant_vm_add(self):
        args = self.parser.parse_args('tenant vm add --name=tenant1 --vm-list vm1,vm2'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_vm_add)
        self.assertEqual(args.name, 'tenant1')
        self.assertEqual(args.vm_list, ['vm1', 'vm2'])
    
    def test_tenant_vm_add_missing_option_fails(self):
        self.assert_parse_error('tenant vm add')
        self.assert_parse_error('tenant vm add --name=tenant1')

    def test_tenant_vm_rm(self):
        args = self.parser.parse_args('tenant vm rm --name=tenant1 --vm-list vm1,vm2'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_vm_rm)
        self.assertEqual(args.name, 'tenant1')
        self.assertEqual(args.vm_list, ['vm1', 'vm2'])
    
    def test_tenant_vm_rm_missing_option_fails(self):
        self.assert_parse_error('tenant vm add')
        self.assert_parse_error('tenant vm add --name=tenant1')
    
    def test_tenant_vm_ls(self):
        args = self.parser.parse_args('tenant vm ls --name=tenant1'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_vm_ls)
        self.assertEqual(args.name, 'tenant1')
    
    def test_tenant_vm_ls_missing_option_fails(self):
        self.assert_parse_error('tenant vm ls')
    
    def test_tenant_access_add(self):
        args = self.parser.parse_args('tenant access add --name=tenant1 --datastore=datastore1 --default-datastore --allow-create --volume-maxsize=500MB --volume-totalsize=1GB'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_add)
        self.assertEqual(args.name, 'tenant1')
        self.assertEqual(args.datastore, 'datastore1')
        self.assertEqual(args.allow_create, True)
        self.assertEqual(args.default_datastore, True)
        self.assertEqual(args.volume_maxsize, '500MB')
        self.assertEqual(args.volume_totalsize, '1GB')
    
    def test_tenant_access_add_missing_option_fails(self):
        self.assert_parse_error('tenant access add')
        self.assert_parse_error('tenant access add --name=tenant1')
    
    def test_tenant_access_add_invalid_option_fails(self):
        self.assert_parse_error('tenant access add --name=tenant1 --datastore=datastore1 --rights=create mount')
            
    def test_tenant_accss_set(self):
        args = self.parser.parse_args('tenant access set --name=tenant1 --datastore=datastore1 --allow-create --volume-maxsize=500MB --volume-totalsize=1GB'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_set)
        self.assertEqual(args.name, 'tenant1')
        self.assertEqual(args.datastore, 'datastore1')
        self.assertEqual(args.allow_create, True)
        self.assertEqual(args.volume_maxsize, '500MB')
        self.assertEqual(args.volume_totalsize, '1GB')
        
    def test_tenant_access_set_missing_option_fails(self):
        self.assert_parse_error('tenant access set')
        self.assert_parse_error('tenant access set --name=tenant1')
    
    def test_tenant_access_set_invalid_option_fails(self):
        self.assert_parse_error('tenant access set --name=tenant1 --datastore=datastore1 --rights=crete,mount')
    
    def test_tenant_access_rm(self):
        args = self.parser.parse_args('tenant access rm --name=tenant1 --datastore=datastore1'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_rm)
        self.assertEqual(args.name, 'tenant1')
        self.assertEqual(args.datastore, 'datastore1')

    def test_tenant_access_rm_missing_option_fails(self): 
        self.assert_parse_error('tenant access rm')
        self.assert_parse_error('tenant access rm --name=tenant1')
    
    def test_tenant_access_ls(self):
        args = self.parser.parse_args('tenant access ls --name=tenant1'.split())
        self.assertEqual(args.func, vmdkops_admin.tenant_access_ls)
        self.assertEqual(args.name, 'tenant1')
    
    def test_tenant_access_ls_missing_option_fails(self):
        self.assert_parse_error('tenant access ls')

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
        args = self.parser.parse_args('set --volume=vol_name@datastore --options="access=read-only"'.split())
        self.assertEqual(args.func, vmdkops_admin.set_vol_opts)
        self.assertEqual(args.volume, 'vol_name@datastore')
        self.assertEqual(args.options, '"access=read-only"')

        args = self.parser.parse_args('set --volume=vol_name@datastore --options="attach-as=persistent"'.split())
        self.assertEqual(args.func, vmdkops_admin.set_vol_opts)
        self.assertEqual(args.volume, 'vol_name@datastore')
        self.assertEqual(args.options, '"attach-as=persistent"')

        args = self.parser.parse_args('set --volume=vol_name@datastore --options="attach-as=independent_persistent"'.split())
        self.assertEqual(args.func, vmdkops_admin.set_vol_opts)
        self.assertEqual(args.volume, 'vol_name@datastore')
        self.assertEqual(args.options, '"attach-as=independent_persistent"')

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
        for (datastore, url_name, path) in vmdk_utils.get_datastores():
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
        for (datastore, url_name, path) in vmdk_utils.get_datastores():
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
            set_ok = vmdk_ops.set_vol_opts(vol_arg, attach_as_arg)
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
            set_ok = vmdk_ops.set_vol_opts(vol_arg, access_arg)
            self.assertTrue(set_ok)

            metadata = vmdkops_admin.get_metadata(os.path.join(v['path'], v[
                'filename']))
            self.assertNotEqual(None, metadata)

            curr_access = vmdkops_admin.get_access(metadata)
            self.assertEqual(access_opt, curr_access)

class TestStatus(unittest.TestCase):
    """ Test status functionality """
    def test_status(self):
        self.assertEqual(vmdkops_admin.status(None), None)

class TestTenant(unittest.TestCase):
    """ Test tenant functionality """
    # tenant1 info
    tenant1_name = "test_tenant1"
    vm1_name = 'test_vm1'
    vm1 = None
    vm2_name = 'test_vm2'
    vm2 = None
    tenant1_new_name = "new_test_tenant1"
    datastore_name = None
    datastore1_name = None

    def setUp(self):
        """ Setup run before each test """
        
        if (not self.datastore_name):
            datastores = vmdk_utils.get_datastores()
            if datastores:
                datastore = datastores[0]
                self.datastore_name = datastore[0]
                self.datastore_path = datastore[2]
            
                if len(datastores) >= 1:
                    datastore1 = datastores[1]
                    self.datastore1_name = datastore1[0]
                    self.datastoer1_path = datastore[2]
      
            else:
            
                self.assertFalse(True)

        self.cleanup()
        # get service_instance, and create VMs
        si = vmdk_ops.get_si()
        error, self.vm1 = vmdk_ops_test.create_vm(si=si, 
                                    vm_name=self.vm1_name, 
                                    datastore=self.datastore_name)
        if error:
            self.assertFalse(True)

        self.vm1_config_path = vmdk_utils.get_vm_config_path(self.vm1_name)

        error, self.vm2 = vmdk_ops_test.create_vm(si=si, 
                                    vm_name=self.vm2_name, 
                                    datastore=self.datastore_name)
        if error:
            self.assertFalse(True)
        self.vm2_config_path = vmdk_utils.get_vm_config_path(self.vm2_name)
          
    def tearDown(self):
        """ Cleanup after each test """
        self.cleanup()      
            
    def cleanup(self):
        # cleanup existing tenant
        error_info = auth_api._tenant_rm(
                                         name=self.tenant1_name, 
                                         remove_volumes=True)
                                         
        error_info = auth_api._tenant_rm(
                                         name=self.tenant1_new_name, 
                                         remove_volumes=True)                                          
        
        # remove VM
        si = vmdk_ops.get_si()
        vmdk_ops_test.remove_vm(si, self.vm1)
        vmdk_ops_test.remove_vm(si, self.vm2)
    
    def test_tenant(self):
        # create tenant1 without adding any vms and privileges
        name = self.tenant1_name
        vm_list = [self.vm1_name]
        description = "Test tenant1" 
        privileges = []
  
        error_info, tenant = auth_api._tenant_create(
                                                    name=name, 
                                                    description=description,  
                                                    vm_list=vm_list, 
                                                    privileges=privileges)
        self.assertEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)
        self.assertEqual(len(header), TENANT_LS_EXPECTED_COLUMN_COUNT)
        
        # Two tenants in the list, "_DEFAULT" and "test_tenant1"
        # row[0] is for "_DEFAULT" tenant, and row[1] is for "test_tenant1"
        self.assertEqual(len(rows), 2)

        # There are 5 columns for each row, the name of the columns are 
        # "Uuid", "Name", "Description", "Default_datastore", "VM_list"
        # Sample output of row[1]:
        # [u'9e1be0ce-3d58-40f6-a335-d6e267e34baa', u'test_tenant1', u'Test tenant1', '', 'test_vm1']
        expected_output = [str_to_unicode(self.tenant1_name),
                           str_to_unicode("Test tenant1"),
                           "",
                           generate_vm_name_str(vm_list)]
        actual_output = [rows[1][1],
                         rows[1][2],
                         rows[1][3],
                         rows[1][4]]
        
        self.assertEqual(expected_output, actual_output)

        # tenant update to update description default_datastore
        error_info = auth_api._tenant_update(
                                             name=name, 
                                             description="This is test tenant1",  
                                             default_datastore=self.datastore_name)
        self.assertEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)

        expected_output = [str_to_unicode(self.tenant1_name),
                           str_to_unicode("This is test tenant1"),
                           self.datastore_name,
                           generate_vm_name_str(vm_list)]
        actual_output = [rows[1][1],
                         rows[1][2],
                         rows[1][3],
                         rows[1][4]]

        self.assertEqual(expected_output, actual_output)

        # tenant update to rename the tenant
        error_info  = auth_api._tenant_update(
                                              name=name, 
                                              new_name=self.tenant1_new_name)
        self.assertEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)

        expected_output = [str_to_unicode(self.tenant1_new_name),
                           str_to_unicode("This is test tenant1"),
                           self.datastore_name,
                           generate_vm_name_str(vm_list)]
        actual_output = [rows[1][1],
                         rows[1][2],
                         rows[1][3],
                         rows[1][4]]
        

        # tenant rm to remove the tenant
        error_info = auth_api._tenant_rm(
                                         name=self.tenant1_new_name, 
                                         remove_volumes=True)
        self.assertEqual(None, error_info)

        error_info, tenant_list = auth_api._tenant_ls()
        self.assertEqual(None, error_info)

        header = vmdkops_admin.tenant_ls_headers()
        rows = vmdkops_admin.generate_tenant_ls_rows(tenant_list)
       
        # right now, should only have 1 tenant, which is "_DEFAULT" tenant
        self.assertEqual(len(rows), 1)


if __name__ == '__main__':
    kv.init()
    unittest.main()
