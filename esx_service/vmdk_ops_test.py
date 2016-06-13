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
import os, os.path

import vmdk_ops
import log_config
import volume_kv
import vsan_policy
import vsan_info
import vmdk_utils

# will do creation/deletion in this folder:
global path


class VmdkCreateRemoveTestCase(unittest.TestCase):
    """Unit test for VMDK Create and Remove ops"""

    volName = "vol_UnitTest_Create"
    badOpts = {u'policy': u'good', volume_kv.SIZE: u'12unknown', volume_kv.DISK_ALLOCATION_FORMAT: u'5disk'}
    name = ""
    vm_name = 'test-vm'

    def setUp(self):
        self.name = vmdk_ops.getVmdkName(path, self.volName)
        self.policy_names = ['good', 'impossible']
        policy_content = ('(("proportionalCapacity" i50) '
                          '("hostFailuresToTolerate" i0))')
        for n in self.policy_names:
            vsan_policy.create(n, policy_content)

    def tearDown(self):
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
        logging.info(err)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name)
        logging.info(err)
        self.assertNotEqual(err, None, err)

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
        path = vmdk_utils.get_vsan_dockvols_path()
        i = 0
        for unit in testInfo:
            vol_name = '{0}{1}'.format(self.volName, i)
            vmdk_path = vmdk_ops.getVmdkName(path,vol_name)
            i = i+1
            # create a volume with requestes size/policy and check vs expected result
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
            self.assertEquals(None, result,
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
        
        diskformats = ["zeroedthick", "thin", "eagerzeroedthick"]
        diskformats.extend([diskformat.upper() for diskformat in diskformats])

        for s in sizes:
            for p in self.policy_names:
                for d in diskformats:
                # An exception should not be raised
                    vmdk_ops.validate_opts({volume_kv.SIZE: s, volume_kv.VSAN_POLICY_NAME: p, volume_kv.DISK_ALLOCATION_FORMAT : d},
                                       self.path)
                    vmdk_ops.validate_opts({volume_kv.SIZE: s}, self.path)
                    vmdk_ops.validate_opts({volume_kv.VSAN_POLICY_NAME: p}, self.path)
                    vmdk_ops.validate_opts({volume_kv.DISK_ALLOCATION_FORMAT: d}, self.path)

    def test_failure(self):
        bad = [{volume_kv.SIZE: '2'}, {volume_kv.VSAN_POLICY_NAME: 'bad-policy'}, 
        {volume_kv.DISK_ALLOCATION_FORMAT: 'bad-format'}, {volume_kv.SIZE: 'mb'}, {'bad-option': '4'}, {'bad-option': 'what',
                                                             volume_kv.SIZE: '4mb'}]
        for opts in bad:
            with self.assertRaises(vmdk_ops.ValidationError):
                vmdk_ops.validate_opts(opts, self.path)


if __name__ == '__main__':
    log_config.configure()
    volume_kv.init()

    # Calculate the path
    paths = glob.glob("/vmfs/volumes/[a-zA-Z]*/dockvols")
    if paths:
        # WARNING: for many datastores with dockvols, this picks up the first
        path = paths[0]
    else:
        # create dir in a datastore (just pick first datastore if needed)
        path = glob.glob("/vmfs/volumes/[a-zA-Z]*")[0] + "/dockvols"
        logging.debug("Directory does not exist - creating %s", path)
        os.makedirs(path)

    logging.info("Directory used in test - %s", path)

    try:
        unittest.main()
    except:
        pass
    finally:
        if not paths:
            logging.debug("Directory clean up - removing  %s", path)
            os.removedirs(path)
    #suite = unittest.TestLoader().loadTestsFromTestCase(TestStringMethods)
    #unittest.TextTestRunner(verbosity=2).run(suite)
