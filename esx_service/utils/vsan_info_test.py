#!/usr/bin/env python
#
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
# limitations under the License.#
#
# tests for vsan_info.py

import unittest
import log_config
import volume_kv as kv
import vmdk_ops
import os.path
import pyVim.connect

import vsan_info


@unittest.skipIf(not vsan_info.get_vsan_datastore(),
                "VSAN is not found - skipping vsan_info tests")
class TestVsanInfo(unittest.TestCase):
    """ Test VSAN Info API """

    VM_NAME = "test-vm"
    VSAN_DS = "/vmfs/volumes/vsandatastore"
    TEST_DIR = os.path.join(VSAN_DS, "vsan_info_test")
    TEST_VOL = "test_policy_vol"
    VMDK_PATH = os.path.join(TEST_DIR, TEST_VOL + ".vmdk")
    NON_VSAN_VMDK = "/vmfs/volumes/datastore/eek/other.vmdk"

    # make sure dir is there, failure "already exits" is OK here
    vmdk_ops.RunCommand("/usr/lib/vmware/osfs/bin/osfs-mkdir " + TEST_DIR)

    def __init__(self, *args, **kwargs):
        super(TestVsanInfo, self).__init__(*args, **kwargs)
        self.si = None

    def setUp(self):
        """create a vmdk before each test (method) in this class"""
        si = vmdk_ops.get_si()
        # create VMDK
        err = vmdk_ops.createVMDK(vmdk_path=self.VMDK_PATH,
                                  vm_name=self.VM_NAME,
                                  vol_name="test_policy_vol")
        self.assertEqual(err, None, err)

    def tearDown(self):
        """clean up after each test (method) in this class"""
        err = vmdk_ops.removeVMDK(self.VMDK_PATH)
        self.assertEqual(err, None, err)
        pyVim.connect.Disconnect(self.si)

    def test_ds(self):
        self.assertNotEqual(vsan_info.get_vsan_datastore(), None,
                            "Failed to find VSAN datastore")
        self.assertTrue(
            vsan_info.is_on_vsan(self.VMDK_PATH),
            "is_on_vsan can't find file %s" % self.VMDK_PATH)
        self.assertFalse(
            vsan_info.is_on_vsan(self.NON_VSAN_VMDK),
            "is_on_vsan is mistaken about the file %s" % self.NON_VSAN_VMDK)

    def test_policy(self):
        # check it's on VSAN
        self.assertTrue(
            vsan_info.is_on_vsan(self.VMDK_PATH),
            "is_on_vsan can't find file %s" % self.VMDK_PATH)
        # set policy
        policy_string = \
            '(("hostFailuresToTolerate" i0) ("forceProvisioning" i1))'
        # same policy content with different space/tabs:
        same_policy = \
            ' ((  "hostFailuresToTolerate"    \ti0) ("forceProvisioning" i1))'
        # different content:
        notsame_policy = \
            '(("hostFailuresToTolerate" i0) ("forceProvisioning" i0))'
        err = vsan_info.set_policy(self.VMDK_PATH, policy_string)
        self.assertEqual(err, None, "failed to set")
        # get policy and check it
        p = vsan_info.get_policy(self.VMDK_PATH)
        self.assertTrue(
            vsan_info.same_policy(self.VMDK_PATH, p),
            "failed to compare with get_policy")
        self.assertTrue(
            vsan_info.same_policy(self.VMDK_PATH, policy_string),
            "failed to compare with original policy")
        self.assertTrue(
            vsan_info.same_policy(self.VMDK_PATH, same_policy),
            "failed to compare with same policy, different tabs")
        self.assertFalse(
            vsan_info.same_policy(self.VMDK_PATH, notsame_policy),
            "failed to compare with different policy")


if __name__ == "__main__":
    log_config.configure()
    kv.init()
    unittest.main()
