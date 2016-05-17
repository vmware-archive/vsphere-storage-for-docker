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

# Ensure that actually running /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py

import subprocess
import unittest
import os

admin_cli = '/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py'

class TestVmdkopsAdminSanity(unittest.TestCase):
    """ Test output from running vmdkops_admin.py """

    def setUp(self):
        os.chdir('/')
        self.devnull = open('/dev/null', 'w')

    def tearDown(self):
        self.devnull.close()

    def test_ls(self):
        cmd = '{0} ls'.format(admin_cli)
        # Don't print errors about stty using a bad ioctl (we aren't attached to
        # a tty here)
        output = subprocess.check_output(cmd, shell=True, stderr=self.devnull)
        lines = output.split('\n')
        header1, header2 = lines[0].split()
        self.assertEqual(header1, 'Volume')
        self.assertEqual(header2, 'Datastore')
        for string in lines[1].split():
            self.assertTrue(all_dashes(string))

    def test_policy_ls(self):
        cmd = '{0} policy ls'.format(admin_cli)
        # Don't print errors about stty using a bad ioctl (we aren't attached to
        # a tty here)
        output = subprocess.check_output(cmd, shell=True, stderr=self.devnull)
        lines = output.split('\n')
        headers = lines[0].split()
        self.assertEqual(['Policy', 'Name', 'Policy', 'Content', 'Active'],
                         headers)
        for string in lines[1].split():
            self.assertTrue(all_dashes(string))

def all_dashes(string):
    return all(map(lambda char: char == '-', string))

if __name__ == '__main__':
    unittest.main()
