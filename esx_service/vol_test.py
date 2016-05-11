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

# NOTE: this test needs rework and is not functional currently.
print("VOL_TEST kv_store skippped")
exit(0)
'''
Standalone tests for metadata volume operations
Create vm and try to attach/detach disks with metadata
This test is NOT a part of 'make testremote' or Drone CI pass
'''

import os
import atexit
import sys, getopt
import subprocess
import vmci_srv as vmci
import volume_kv as kv

# Default volumes dir
vmName = "testVM"
vols = ['vol1', 'vol2', 'vol3', 'vol4', 'vol5', 'vol6', 'vol7', 'vol8', 'vol9',
        'vol10']
volopts = "size:1gb"


def doCreate(volDir):
    print "Creating volumes"
    for vol in vols:
        volPath = os.path.join(volDir, "%s.vmdk" % vol)
        vmci.createVMDK(volPath, vol, None)

    print "Verifying volume metadata"
    for vol in vols:
        volPath = os.path.join(volDir, "%s.vmdk" % vol)
        volDict = kv.getAll(volPath)

        if not volDict:
            print "Failed to fetch volume meta-data for ", volPath
            continue

        print "Vol metadata 'status' - %s, 'volOpts' - %s" % (
            volDict['status'], volDict['volOpts'])
        if volDict['status'] != 'detached':
            print 'Found volume %s with status %s, expected' % (
                vol, volDict['status'], 'detached')

    return


def doAttach(volDir, vmName):
    print "Attaching volumes"
    for vol in vols:
        volPath = os.path.join(volDir, "%s.vmdk" % vol)
        vmci.attachVMDK(volPath, vmName)
    print "Verifying volume metadata"
    for vol in vols:
        volPath = os.path.join(volDir, "%s.vmdk" % vol)
        volDict = kv.getAll(volPath)

        if volDict['status'] != 'attached':
            print 'Found volume %s with status %s, expected' % (
                vol, volDict['status'], 'attached')

    return


def doDetach(volDir, vmName):
    print "Detaching volumes"
    for vol in vols:
        volPath = os.path.join(volDir, "%s.vmdk" % vol)
        vmci.detachVMDK(volPath, vmName)
    print "Verifying volume metadata"
    for vol in vols:
        volPath = os.path.join(volDir, "%s.vmdk" % vol)
        volDict = kv.getAll(volPath)

        if volDict['status'] != 'detached':
            print 'Found volume %s with status %s, expected' % (
                vol, volDict['status'], 'detached')

    return


def doVolDelete(volDir):
    print 'Removing volumes'
    for vol in vols:
        volPath = os.path.join(volDir, "%s.vmdk" % vol)
        vmci.removeVMDK(volPath)
    return


def cleanup(vmId):
    cmd = 'vim-cmd vmsvc/power.off %s' % vmId
    subprocess.call(cmd, shell=True)

    cmd = 'vim-cmd vmsvc/destroy %s' % vmId
    subprocess.call(cmd, shell=True)


def main(argv):
    if argv == []:
        print 'vol_tests.py -d <test dir>'
        sys.exit(2)

    try:
        opts, args = getopt.getopt(argv, "hd:")
    except getopt.GetoptError:
        print 'vol_tests.py -d <test dir>'
        sys.exit(2)

    for opt, arg in opts:
        if opt == '-h':
            print 'vol_tests.py -v <vm config path> -d <volumes dir>'
            sys.exit()
        elif opt in ("-d"):
            volDir = arg

    # Init logging
    logfile = "%s/test.log" % volDir
    vmci.LogSetup(logfile)

    # Init KV
    kv.init()

    cmd = 'vim-cmd vmsvc/createdummyvm %s %s' % (vmName, volDir)
    proc = subprocess.Popen(cmd,
                            stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE,
                            shell=True)

    s = proc.communicate()[0]

    vmId = s.rstrip()

    ret = proc.returncode

    atexit.register(cleanup, vmId)

    if ret != 0:
        print "Failed to power on VM, exiting", vmName
        sys.exit(0)

    # Start VM
    print "Starting VM %s with id %s ..." % (vmName, vmId)

    cmd = 'vim-cmd vmsvc/power.on %s' % vmId
    subprocess.call(cmd, shell=True)

    # Create volumes
    doCreate(volDir)

    # Attach/Re-attach volumes
    #doAttach(volDir, vmName)

    # Check volume meta-data
    #doVerifyVolMeta(volDir)

    # Detach volumes
    #doDetach(volDir, vmName)

    # Delete volumes
    doVolDelete(volDir)

# start the server
if __name__ == "__main__":
    main(sys.argv[1:])
