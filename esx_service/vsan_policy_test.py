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
import os, os.path
import vsan_policy
import vmdk_utils
import volume_kv
import vsan_info
import logging


class TestVsanPolicy(unittest.TestCase):
    """ Test VSAN Policy code """

    @unittest.skipIf(not vsan_info.get_vsan_datastore(),
                     "VSAN is not found - skipping vsan_info tests")

    def setUp(self):
        self.policy_path = os.path.join(vsan_info.get_vsan_dockvols_path(),
                                        'policies/test_policy')
        self.name = 'test_policy'
        self.content = ('(("proportionalCapacity" i50) '
                        '("hostFailuresToTolerate" i0))')

    def tearDown(self):
        try:
            os.remove(self.policy_path)
        except:
            pass

    def assertPoliciesEqual(self):
        with open(self.policy_path) as f:
            content = f.read()
        # Remove the added newline
        self.assertEqual(content[:-1], self.content)

    def test_create(self):
        self.assertEqual(None, vsan_policy.create(self.name, self.content))
        self.assertPoliciesEqual()

    def test_double_create_fails(self):
        self.assertEqual(None, vsan_policy.create(self.name, self.content))
        self.assertNotEqual(None, vsan_policy.create(self.name, self.content))
        self.assertPoliciesEqual()

    def test_create_delete(self):
        self.assertEqual(None, vsan_policy.create(self.name, self.content))
        self.assertPoliciesEqual()
        self.assertEqual(None, vsan_policy.delete(self.name))
        self.assertFalse(os.path.isfile(self.policy_path))

    def test_delete_nonexistent_policy_fails(self):
        self.assertNotEqual(None, vsan_policy.delete(self.name))
        logging.info("The test itself is expected to fail, and please ignore the errors printed above.")

    def test_create_list(self):
        self.assertEqual(None, vsan_policy.create(self.name, self.content))
        policies = vsan_policy.get_policies()
        self.assertTrue(self.content + '\n', policies[self.name])


if __name__ == '__main__':
    volume_kv.init()
    unittest.main()
