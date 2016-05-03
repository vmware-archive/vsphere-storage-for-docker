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
import volumeKVStore as kv
import vmdkops_admin

class TestParsing(unittest.TestCase):
    """ Test command line arg parsing for all commands """

    def setUp(self):
      self.parser = vmdkops_admin.create_parser()

    def test_parse_ls_no_options(self):
        args = self.parser.parse_args(['ls'])
        self.assertEqual(args.func, vmdkops_admin.ls)
        self.assertEqual(args.l, False)
        self.assertEqual(args.c, None)

    def test_parse_ls_dash_l(self):
        args = self.parser.parse_args('ls -l'.split())
        self.assertEqual(args.func, vmdkops_admin.ls)
        self.assertEqual(args.l, True)
        self.assertEqual(args.c, None)

    def test_parse_ls_dash_c(self):
        args = self.parser.parse_args('ls -c created-by,created,last-attached'.split())
        self.assertEqual(args.func, vmdkops_admin.ls)
        self.assertEqual(args.l, False)
        self.assertEqual(args.c, ['created-by', 'created', 'last-attached'])

    def test_parse_ls_dash_c_invalid_argument(self):
        self.assert_parse_error('ls -c personality')

    def test_policy_no_args_fails(self):
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

    def test_role_create(self):
        cmd = 'role create --name=carl --volume-maxsize=2TB ' + \
              '--matches-vm test*,qa* --rights=create,mount'
        args = self.parser.parse_args(cmd.split())
        self.assertEqual(args.func, vmdkops_admin.role_create)
        self.assertEqual(args.name, 'carl')
        self.assertEqual(args.volume_maxsize, '2TB')
        self.assertEqual(args.matches_vm, ['test*', 'qa*'])
        self.assertEqual(args.rights, ['create', 'mount'])

    def test_role_create_missing_option_fails(self):
        cmd = 'role create --name=carl --volume-maxsize=2TB --matches-vm=test*,qa*'
        self.assert_parse_error(cmd)

    def test_role_rm(self):
        args = self.parser.parse_args('role rm myRole'.split())
        self.assertEqual(args.func, vmdkops_admin.role_rm)
        self.assertEqual(args.name, 'myRole')

    def test_role_rm_missing_name(self):
        self.assert_parse_error('role rm')

    def test_role_ls(self):
        args = self.parser.parse_args('role ls'.split())
        self.assertEqual(args.func, vmdkops_admin.role_ls)

    def test_role_set(self):
        cmds = [
            'role set --name=carl --volume-maxsize=4TB',
            'role set --name=carl --rights create,mount',
            'role set --name=carl --matches-vm marketing*',
            'role set --name=carl --volume-maxsize=2GB --rights create,mount,delete'
            ]
        for cmd in cmds:
            args = self.parser.parse_args(cmd.split())
            self.assertEqual(args.func, vmdkops_admin.role_set)
            self.assertEqual(args.name, 'carl')

    def test_role_set_missing_name_fails(self):
        self.assert_parse_error('role set --volume-maxsize=4TB')

    def test_role_get(self):
        args = self.parser.parse_args('role get testVm'.split())
        self.assertEqual(args.func, vmdkops_admin.role_get)
        self.assertEqual(args.vm_name, 'testVm')

    def test_status(self):
        args = self.parser.parse_args(['status'])
        self.assertEqual(args.func, vmdkops_admin.status)

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
        for (datastore, path) in vmdkops_admin.get_datastores():
            if not self.mkdir(path):
                continue
            for id in range(5):
                volName = 'testvol'+str(id)
                fullpath = os.path.join(path, volName+'.vmdk')
                self.assertEqual(None, vmdk_ops.createVMDK(vmdkPath=fullpath, volName=volName))
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
            self.assertEqual(None, vmdk_ops.removeVMDK(os.path.join(v['path'], v['filename'])))

    def get_testvols(self):
        return [x for x in vmdkops_admin.get_volumes() if x['filename'].startswith('testvol')]

    def test_ls_helpers(self):
        volumes = self.get_testvols()
        self.assertEqual(len(volumes), self.vol_count)
        for v in volumes:
            metadata = vmdkops_admin.get_metadata(os.path.join(v['path'], v['filename']))
            self.assertNotEqual(None, metadata)

    def test_ls_no_args(self):
          volumes = vmdkops_admin.get_volumes()
          (header, data) = vmdkops_admin.ls_no_args()
          self.assertEqual(2, len(header))
          self.assertEqual(len(volumes), len(data))
          for i in range(len(volumes)):
              self.assertEqual(volumes[i]['filename'], data[i][0]+'.vmdk')

if __name__ == '__main__':
    kv.init()
    unittest.main()
