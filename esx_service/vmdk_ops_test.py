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
import volume_kv as kv
import vsan_policy

# will do creation/deletion in this folder:
global path


class VmdkCreateRemoveTestCase(unittest.TestCase):
    """Unit test for VMDK Create and Remove ops"""

    volName = "vol_UnitTest_Create"
    badOpts = {u'policy': u'good', u'size': u'12unknown'}
    name = ""

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
        err = vmdk_ops.createVMDK(vmdkPath=self.name, volName=self.volName)
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
        err = vmdk_ops.createVMDK(vmdkPath=self.name,
                                  volName=self.volName,
                                  opts=self.badOpts)
        logging.info(err)
        self.assertNotEqual(err, None, err)

        err = vmdk_ops.removeVMDK(self.name)
        logging.info(err)
        self.assertNotEqual(err, None, err)

    def testPolicy(self):
        # info for testPolicy
        testInfo = [
            #    size     policy   expected success?
            ["2gb", "good", True],
            ["14000pb", "good", False],
            ["bad size", "good", False],
            ["100mb", "impossible", True],
            ["100mb", "good", True],
        ]
        for unit in testInfo:
            # create a volume with requestes size/policy and check vs expected result
            err = vmdk_ops.createVMDK(vmdkPath=self.name,
                                      volName=self.volName,
                                      opts={'vsan-policy-name': unit[1],
                                            'size': unit[0]})
            self.assertEqual(err == None, unit[2], err)

            # clean up should fail if the created should have failed.
            err = vmdk_ops.removeVMDK(self.name)
            self.assertEqual(err == None, unit[2], err)


class ValidationTestCase(unittest.TestCase):
    """ Test validation of -o options on create """

    def setUp(self):
        """ Create a bunch of policies """
        self.policy_names = ['name1', 'name2', 'name3']
        self.policy_content = ('(("proportionalCapacity" i50) '
                               '("hostFailuresToTolerate" i0))')
        for n in self.policy_names:
            vsan_policy.create(n, self.policy_content)

    def tearDown(self):
        for n in self.policy_names:
            vsan_policy.delete(n)

    def test_success(self):
        sizes = ['2gb', '200tb', '200mb', '5kb']
        sizes.extend([s.upper() for s in sizes])
        for s in sizes:
            for p in self.policy_names:
                # An exception should not be raised
                vmdk_ops.validate_opts({'size': s, 'vsan-policy-name': p})
                vmdk_ops.validate_opts({'size': s})
                vmdk_ops.validate_opts({'vsan-policy-name': p})

    def test_failure(self):
        bad = [{'size': '2'}, {'vsan-policy-name': 'bad-policy'},
               {'size': 'mb'}, {'bad-option': '4'}, {'bad-option': 'what',
                                                     'size': '4mb'}]
        for opts in bad:
            with self.assertRaises(vmdk_ops.ValidationError):
                vmdk_ops.validate_opts(opts)


if __name__ == '__main__':
    log_config.configure()
    kv.init()

    # Calculate the path
    paths = glob.glob("/vmfs/volumes/[a-z]*/dockvols")
    if paths:
        # WARNING: for many datastores with dockvols, this picks up the first
        path = paths[0]
    else:
        # create dir in a datastore (just pick first datastore if needed)
        path = glob.glob("/vmfs/volumes/[a-z]*")[0] + "/dockvols"
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
