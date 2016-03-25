#!/usr/bin/env python

'''
Tests for basic vmdk Operations

'''

import unittest
import sys
import logging
import glob
import os.path

import vmdk_ops
import volumeKVStore as kv

# Get dockvols full path
# WARNING: for many datastores with dockvols, this picks up the first
path = glob.glob("/vmfs/volumes/[a-z]*/dockvols")[0]

# TODO:
# (1) clean up TODO from vmci_svc.py in a separate check-in

class VmdkCreateRemoveTestCase(unittest.TestCase):
    """Unit test for VMDK Create and Remove ops"""

    volName = "vol_UnitTest_Create"
    badOpts = {u'policy': u'good', u'size': u'12unknown'}
    name = ""

    def setUp(self):
        self.name = vmdk_ops.getVmdkName(path, self.volName)

    def tearDown(self):
        self.vmdk = None

    def testCreateDelete(self):
        err = vmdk_ops.createVMDK(vmdkPath=self.name, volName=self.volName)
        self.assertEqual(err, None, err)
        self.assertEqual(os.path.isfile(self.name), True,
                    "VMDK {0} is missing after create.".format(self.name))
        err = vmdk_ops.removeVMDK(self.name)
        self.assertEqual(err, None, err)
        self.assertEqual(os.path.isfile(self.name), False,
                    "VMDK {0} is still present after delete.".format(self.name))


    def testBadOpts(self):
        err = vmdk_ops.createVMDK(vmdkPath=self.name, volName=self.volName,
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
            ["2gb",     "good",     True],
            ["14000pb", "good",     False ],
            ["bad size","good",     False],
            ["100mb", "impossible", True],
            ["100mb",  "good",      True],
            ]
        for unit in testInfo:
            # create a volume with requestes size/policy and check vs expected result
            err = vmdk_ops.createVMDK(vmdkPath=self.name, volName=self.volName,
                                      opts={u'policy': unit[1], u'size':unit[0]})
            self.assertEqual(err == None, unit[2] , err)

            # clean up should fail if the created should have failed.
            err = vmdk_ops.removeVMDK(self.name)
            self.assertEqual(err == None, unit[2], err)


# init log stream handle to use sys.output before running tests.
# this is needed if we want to keep using "print" in test
# otherwise the logger from vmdk_ops is just fine
def log_init():
    logger = logging.getLogger()
    logger.level = logging.DEBUG
    stream_handler = logging.StreamHandler(sys.stdout)
    logger.addHandler(stream_handler)

if __name__ == '__main__':
    #log_init()
    vmdk_ops.LogSetup("/var/log/vmware/docker-vmdk-plugin-pytest.log")
    kv.init()
    unittest.main()
    #suite = unittest.TestLoader().loadTestsFromTestCase(TestStringMethods)
    #unittest.TextTestRunner(verbosity=2).run(suite)
