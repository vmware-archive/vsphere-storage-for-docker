#!/usr/bin/env python

'''
TOP TODO

0. extract attach/detach code and make sure it work fine without anything else
1. add WaitForTask in Python  and drop sleep, Also add time track for reconfigure
2. Add scsi_resync to plugin.go. Check how mount is done in flocker and in VIC
3. Go over code, and collect TODO in one place (add TODO file)
4. add simple test script (or sketch it)
5. Look at cleaning up makefile and adding godeps
'''


#
# ESX-side service  handling VMDK creat/attach requests from VMCI clients
#
# The requests (create/delete/attach/detach) are JSON formatted.
#
# All operations are using requester VM (docker host) datastore and
# "Name" in request refers to vmdk basename
# VMDK name is formed as [vmdatastore] dvol/"Name".vmdk
#
# Commands ("cmd" in request):
#		"create" - create a VMDK in "[vmdatastore] dvol"
#		"remove" - remove a VMDK. We assume it's not open, and fail if it is
#		"list"   - [future, need docker support] enumerate related VMDK
#		"attach" - attach a VMDK to the requesting VM
#		"detach" - detach a VMDK from the requesting VM (assuming it's unmounted)
#
#
# known issues:
# - need interrupt handler to restart listen() on control-c
# - call VirtualDiskManager directly instead of cmd line
# - todo/tbd below

'''
TODO
==
Drop command line and use VDM for disk manipulation:
spec=vim.FileBackedVirtualDiskSpec(capacityKB=1024, profile = None ,adapterType='lsiLogic', diskType = 'thin')
vdm = si.content.virtualDiskManager
vdm.CreateVirtualDisk_Task(name="[datastore] eek/a1.vmdk", datadatacenter=dc, spec = spec)
vm = findVmByName()... ; vm.config.datastoreUrl[0].name is datastore
see https://opengrok.eng.vmware.com/source/xref/vmcore-main.perforce.1666/bora/vmkernel/tests/vsan/vsansparse_sanity.py#314
or even better https://gist.github.com/hartsock/d8b9c56cd7f779c92a78 (fails if exists)
for examples
==

make sure backend name is properly calculated -
OR replace it all with in-guest formatting (but then we'd need to locate the proper blockdevice()

With the current approach, new device is found for mount with blkid -L <volume-name>, which is easier


==
REPLACE prints with logging
===
check size format and handle policy on -o flags
==
make sure the NEW disk is really formatted if if it does not have -flat.vmdk
==
findChild seems to generate a task - too slow. Find out where and drop
==
Pass error as a code in the VMCI package, rather than a hack with server.c:vmci_reply

===

 *   TBD: authorization/access config:
 *   - location configuration. And generally, configuration... host profile/adv.conf?
 *   - a local version, talking over Unix Socket, to debug all I can on Linux - with logic and fake responses (part of build unit test)
 *   - an ESX version, talking over http , to debug logic on a random ESX - with full vigor API usage
 *   - an ESX version, with vSocket (C I suppose) connection, to finalize debugging on actual guest/host story

'''

from ctypes import *
import json
import os
import subprocess
import atexit
import time

from vmware import vsi

import pyVim
from pyVim.connect import Connect, Disconnect
from pyVim.invt import GetVmFolder, FindChild
from pyVim import vmconfig

from pyVmomi import VmomiSupport, vim, vmodl

# defaults
DockVolsDir = "dockvols"   # place in the same (with Docker VM) datastore
MaxDescrSize = 100000  # we assume files smaller that that to be descriptor files
DefaultDiskSize = "100mb"
BinLoc = "/usr/lib/vmware/vmdkops/bin/"

# Run executable on ESX as needed for vmkfstools invocation (until normal disk create is written)
# Returns the integer return value and the stdout str on success and integer return value and
# the stderr str on error
def RunCommand(cmd):
   """RunCommand

   Runs command specified by user

   @param command to execute
   """
   print ("Running cmd %s" % cmd)

   p = subprocess.Popen(cmd,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE,
                        shell=True)
   o, e = p.communicate()
   s = p.returncode

   print "Return:", s, o

   if s != 0:
       return (s, e)

   return (s, o)

# returns error, or None for OK
# opts is  dictionary of {option: value}.
# for now we care about size and (maybe) policy
def createVMDK(vmdkPath, volName, opts):
	print "*** createVMDK: " + vmdkPath
	if os.path.isfile(vmdkPath):
		return "File %s already exists" % vmdkPath

	if not opts or not "size" in opts:
		size = DefaultDiskSize
		print "SETTING DEFAULT SIZE to " , size
	else:
		size = str(opts["size"])
		#TODO: check it is compliant format, and correct if possible
		print "SETTING  SIZE to " , size

        cmd = "/sbin/vmkfstools -d thin -c {0} {1}".format(size, vmdkPath)
        rc, out = RunCommand(cmd)

        if rc != 0:
            if removeVMDK(vmdkPath) == None:
                return {u'Error': "Failed to create %s. %s" % (vmdkPath, out)}
            else:
                return {u'Error': "Unable to create %s and unable to delete volume. Please delete it manually." % vmdkPath}

        return formatVmdk(vmdkPath, volName)

def formatVmdk(vmdkPath, volName):
        # now format it as ext4:
        # WARNING  this won't work on VVOL/VSAN
        # TODO: need to get backing
        # block device differently (in vsan via objtool create)
        backing = vmdkPath.replace(".vmdk", "-flat.vmdk")
        cmd = "{0}/mkfs.ext4   -F -L {1} -q {2}".format(BinLoc, volName, backing)
        rc, out = RunCommand(cmd)

        if rc != 0:
            if removeVMDK(vmdkPath) == None:
                return {u'Error': "Failed to format %s. %s" % (vmdkPath, out)}
            else:
                return {u'Error': "Unable to format %s and unable to delete volume. Please delete it manually." % vmdkPath}

	return None

#returns error, or None for OK
def removeVMDK(vmdkPath):
	print "*** removeVMDK: " + vmdkPath
        cmd = "/sbin/vmkfstools -U {0}".format(vmdkPath)
        rc, out = RunCommand(cmd)
        if rc != 0:
            return {u'Error': "Failed to remove %s. %s" % (vmdkPath, out)}

	return None

# returns a list of volume names (note: may be an empty list)
def listVMDK(path):
	vmdks = [x for x in os.listdir(path) if  ".vmdk" in x and
			os.stat(os.path.join(path, x)).st_size < MaxDescrSize]
        return [{u'Name': x.replace(".vmdk", ""), u'Attributes': {}} for x in vmdks]

# Find VM , reconnect if needed. throws on error
def findVmByName(vmName):
	vm = None
	try:
		vm = FindChild(GetVmFolder(), vmName)
	except AttributeError as ex:
		connectLocal() 					#  retry
		vm = FindChild(GetVmFolder(), vmName)

	if not vm:
		raise Exception("VM" + vmName + "not found")

	return vm

#returns error, or None for OK
def attachVMDK(vmdkPath, vmName):
	vm = findVmByName(vmName)
	print "*** attachVMDK: " + vmdkPath + " to"   + vmName + " uuid=", vm.config.uuid
	disk_attach(vmdkPath, vm)
	return None

#returns error, or None for OK
def detachVMDK(vmdkPath, vmName):
	vm = findVmByName(vmName)
	print "*** detachVMDK: " + vmdkPath + " from "  + vmName + " VM uuid=", vm.config.uuid
	disk_detach(vmdkPath, vm)
	return None

# check existence (and creates if needed) the path
# NOTE / TBD: for vsan we may need to use osfs_mkdir instead of regular os.mkdir
def checkPath(path):
	try:
		os.mkdir(path)
		print path, "created"
	except OSError:
		pass

# gets the requests, calculates path for volumes, and calls the relevant handler
def executeRequest(vmName, vmId, configPath, cmd, volName, opts):
	print "{0} ({1}) -> '{2} {3} ({4}".format(vmName, vmId, cmd, volName, opts)

	# get /vmfs/volumes/<volid> path on ESX:
	path = os.path.join("/".join(configPath.split("/")[0:4]),  DockVolsDir)
	checkPath(path)

	# form full name as <path-to-volumes>/<volname>.vmdk
	vmdkPath = os.path.join(path, "%s.vmdk" % volName)

	if cmd == "create":
		return createVMDK(vmdkPath, volName, opts)
	elif cmd == "remove":
		return removeVMDK(vmdkPath)
	elif cmd == "list":
		return listVMDK(path)
	elif cmd == "attach":
		return attachVMDK(vmdkPath, vmName)
	elif cmd == "detach":
		return detachVMDK(vmdkPath, vmName)
	else:
		return "Unknown command:" + cmd



def connectLocal():
	'''
	connect and do stuff on local machine
	'''

	# Connect to localhost as dcui
	# User "dcui" is a local Admin that does not lose permissions
	# when the host is in lockdown mode.
	si = pyVim.connect.Connect(host='localhost', user='dcui')
	if not si:
		raise SystemExit("Failed to connect to localhost as 'dcui'.")

	atexit.register(pyVim.connect.Disconnect, si)

	# set out ID in context to be used in request - so we'll see it in logs
	# TBD - expose and use outside :-)
	reqCtx = VmomiSupport.GetRequestContext()
	reqCtx["realUser"]='dvolplug'
	return si


# helper to print out all VMs info
def printVMInfo(si):
	container = si.content.rootFolder  # starting point to look into
	containerView = \
		si.content.viewManager.CreateContainerView(container,
																type=[vim.VirtualMachine],
																recursive=True)
	for child in containerView.view:
		summary = child.summary
		print("Name       : ", summary.config.name)
		print("Path       : ", summary.config.vmPathName)
		print("Guest      : ", summary.config.guestFullName)
		print("Instance UUID : ", summary.config.instanceUuid)
		print("Bios UUID     : ", summary.config.uuid)



#
def findDeviceByPath(vmdkPath, vm):

 	for d in vm.config.hardware.device:
 		if type(d) != vim.vm.device.VirtualDisk:
 			continue

 		# TODO: use device_disk_uuid = d.backing.uuid  # FFU
		# for now - ugly hack to convert "[ds] dir/name" to fullpath
		# we also assume homogeneous mounts here... well, we are on 1 esx after all
		dsPath = d.backing.datastore.host[0].mountInfo.path
		dev = os.path.join(dsPath, d.backing.fileName.split(" ")[1])
		if dev == vmdkPath:
			print "findDeviceByPath: MATCH: " + vmdkPath
			return d
		else:
			print "findDeviceByPath: skip: " + dev + " since it's not " + vmdkPath

	return None


# attaches *existing* disk to a vm on a PVSCI controller
# (we need PVSCSI to avoid SCSI rescans in the guest)
def disk_attach(vmdkPath, vm):
  # NOTE:
  # vSphere is very picky about unitNumbers and controllers of virtual
  # disks. Every controller supports 15 virtual disks, and the unit
  # numbers need to be unique within the controller and range from
  # 0 to 15 with 7 being reserved. It is up to the API client to add
  # controllers as needed. Controller keys are in the range of 1000 to 1003
  # (1000 + busNumber).
  max_scsi_controllers = 4

  # changes spec content goes here
  dev_changes = []

  devices = vm.config.hardware.device

  # Make sure we have a PVSCI and add it if we don't

  # TODO: add more controllers if we are out of slots.
  # for now we will throw if we are out of slots

  # get all scsi controllers (pvsci, lsi logic, whatever)
  controllers = [d for d in devices if isinstance(d, vim.VirtualSCSIController)]

  # check if we already have a pvsci one
  pvsci = [d for d in controllers if type(d) == vim.ParaVirtualSCSIController]
  if len(pvsci) > 0:
    diskSlot = None  # need to find out
    controllerKey = pvsci[0].key
  else:
    print "Warning: PVSCI adapter is missing - trying to add one..."
    diskSlot = 0  # starting on a fresh controller
    if len(controllers) == max_scsi_controllers:
      raise StandardError("Error: cannot create PVSCI adapter - VM is out of bus slots")

    # find empty bus slot for the controller:
    taken = set([c.busNumber for c in controllers])
    avail = set(range(0,4)) - taken

    if len(avail) == 0:
      raise  StandardError("Internal error:  can't allocate a bus slot but should be able to.")

    key = avail.pop()           # bus slot
    controllerKey = key + 1000  # controller key (1000 is for SCSI controllers)
    diskSlot = 0
    controller_spec = vim.VirtualDeviceConfigSpec(
      operation = 'add',
      device = vim.ParaVirtualSCSIController(
        key = controllerKey,
        busNumber = key,
        sharedBus = 'noSharing',
      ),
    )
    dev_changes.append(controller_spec)

  # Now find a slot on the controller  , if needed
  if not diskSlot:
    taken = set([dev.unitNumber for dev in devices
                 if type(dev) == vim.VirtualDisk and dev.controllerKey == controllerKey])
    # search in 15 slots, with unit_number 7 reserved for scsi controller
    availSlots = set(range (0,6) + range (8,16))  - taken

    if len(availSlots) == 0:
      raise StandardError("We don't support this many disks yet")

    diskSlot = availSlots.pop()
    print " controllerKey=", controllerKey, " slot=", diskSlot

  # add disk here
  disk_spec = vim.VirtualDeviceConfigSpec(
    operation = 'add',
    device = vim.VirtualDisk(
      backing = vim.VirtualDiskFlatVer2BackingInfo(
        fileName = "[] " + vmdkPath ,
        diskMode = 'persistent',
      ),
      deviceInfo = vim.Description(
        label = "dockerDataVolume",
        # TODO: Use docker data volume name for label
        # this way we use it on detach
        summary = "dockerDataVolume",
      ),
      unitNumber = diskSlot,
      controllerKey = controllerKey,
    ),
  )
  dev_changes.append(disk_spec)

  spec = vim.vm.ConfigSpec()
  spec.deviceChange = dev_changes

  try:
  	wait_for_tasks(si, [vm.ReconfigVM_Task(spec=spec)])
  except vim.fault.GenericVmConfigFault as ex:
    for f in ex.faultMessage:
      print f.message
  else:
    print "disk attached ", vmdkPath


# detach disk (by full path) from a vm
def disk_detach(vmdkPath, vm):

  # Find device object by vmkd path
  # TODO : the right way is to FIND BY disk UUID.
  device = findDeviceByPath(vmdkPath, vm)

  if not device:
  		# TBD: Docker asks to detach something not attached :-) .
  		# Better message is needed
  	 	print "**** SOMETHING IS VERY WRONG: detach_disk did not find " + vmdkPath
  	  	return

  spec = vim.vm.ConfigSpec()
  dev_changes = []

  disk_spec = vim.vm.device.VirtualDeviceSpec()
  disk_spec.operation = vim.vm.device.VirtualDeviceSpec.Operation.remove
  disk_spec.device = device
  dev_changes.append(disk_spec)
  spec.deviceChange = dev_changes

  try:
  		wait_for_tasks(si, [vm.ReconfigVM_Task(spec=spec)])	# si is global
  except vim.fault.GenericVmConfigFault as ex:
  		for f in ex.faultMessage:
			print f.message
  else:
		print "disk detached ", vmdkPath


# Main - connect, load VMCI shared lib and does main loop
def main():
	global si  # we maintain only one connection

	si = connectLocal()

	printVMInfo(si) # just making sure we can do it - print

	# Load and use DLL with vsocket shim to listen for docker requests
	l = cdll.LoadLibrary(BinLoc + "/libvmci_srv.so")

	bsize = 4096 # max buf size for json strings... to be fixed TBD
	txt = create_string_buffer(bsize)

	af = c_int() ; vmciFd = c_int(); cartel = c_int32()
	sock = l.vmci_init(byref(af), byref(vmciFd))
	while True:

		c = l.vmci_get_one_op(sock, af, byref(cartel), txt, c_int(bsize))
		if c == c_int(-1):
			break

		# Get VM name & ID from VSI (we only get cartelID from vmci, need to convert)
		vmmLeader = vsi.get("/userworld/cartel/%s/vmmLeader" % str(cartel.value))
		groupInfo = vsi.get("/vm/%s/vmmGroupInfo" % vmmLeader)

		# vmId - get and convert to format understood by vmodl as a VM key
		# end result should be like this 564d6865-2f33-29ad-6feb-87ea38f9083b"
		# see KB http://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=1880
		s = groupInfo["uuid"]
		vmId   = "{0}-{1}-{2}-{3}-{4}".format(s[0:8], s[9:12], s[12:16], s[16:20], s[20:32])
		vmName = groupInfo["displayName"]
		cfgPath = groupInfo["cfgPath"]

		req = json.loads(txt.value, "utf-8")
		# note: Connection can time out on idle. TODO: to refresh in that case
		details = req["details"]
		opts = details["Opts"] if "Opts" in details else None
		ret = executeRequest(vmName, vmId, cfgPath, req["cmd"], details["Name"], opts)
                print "executeRequest ret = ", ret
		err = l.vmci_reply(c, c_char_p(json.dumps(ret)))
		print "execute_request: VMCI replied with errcode ", err

	l.close(sock, vmciFd)


#-----------------------------------------------------------
#
# Support for 'wait for task completion'
# Keep it here to keep a single file for now
#
"""
Written by Michael Rice <michael@michaelrice.org>

Github: https://github.com/michaelrice
Website: https://michaelrice.github.io/
Blog: http://www.errr-online.com/
This code has been released under the terms of the Apache 2 licenses
http://www.apache.org/licenses/LICENSE-2.0.html

Helper module for task operations.
"""

def wait_for_tasks(service_instance, tasks):
    """Given the service instance si and tasks, it returns after all the
   tasks are complete
   """
    property_collector = service_instance.content.propertyCollector
    task_list = [str(task) for task in tasks]
    # Create filter
    obj_specs = [vmodl.query.PropertyCollector.ObjectSpec(obj=task)
                 for task in tasks]
    property_spec = vmodl.query.PropertyCollector.PropertySpec(type=vim.Task,
                                                               pathSet=[],
                                                               all=True)
    filter_spec = vmodl.query.PropertyCollector.FilterSpec()
    filter_spec.objectSet = obj_specs
    filter_spec.propSet = [property_spec]
    pcfilter = property_collector.CreateFilter(filter_spec, True)
    try:
        version, state = None, None
        # Loop looking for updates till the state moves to a completed state.
        while len(task_list):
            update = property_collector.WaitForUpdates(version)
            for filter_set in update.filterSet:
                for obj_set in filter_set.objectSet:
                    task = obj_set.obj
                    for change in obj_set.changeSet:
                        if change.name == 'info':
                            state = change.val.state
                        elif change.name == 'info.state':
                            state = change.val
                        else:
                            continue

                        if not str(task) in task_list:
                            continue

                        if state == vim.TaskInfo.State.success:
                            # Remove task from taskList
                            task_list.remove(str(task))
                        elif state == vim.TaskInfo.State.error:
                            raise task.info.error
            # Move to next version
            version = update.version
    finally:
        if pcfilter:
            pcfilter.Destroy()

#------------------------



# start the server
if __name__ == "__main__":
    main()
