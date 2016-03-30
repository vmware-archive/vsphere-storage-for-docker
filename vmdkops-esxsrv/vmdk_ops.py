#!/usr/bin/env python

'''
ESX-side service  handling VMDK create/attach requests from VMCI clients

The requests (create/delete/attach/detach) are JSON formatted.

All operations are using requester VM (docker host) datastore and
"Name" in request refers to vmdk basename
VMDK name is formed as [vmdatastore] dockvols/"Name".vmdk

Commands ("cmd" in request):
		"create" - create a VMDK in "[vmdatastore] dvol"
		"remove" - remove a VMDK. We assume it's not open, and fail if it is
		"list"   - [future, need docker support] enumerate related VMDK
		"attach" - attach a VMDK to the requesting VM
		"detach" - detach a VMDK from the requesting VM (assuming it's unmounted)

'''

from ctypes import *
import json
import os
import subprocess
import atexit
import time
import logging
import signal
import sys
sys.dont_write_bytecode = True

from vmware import vsi

import pyVim
from pyVim.connect import Connect, Disconnect
from pyVim.invt import GetVmFolder, FindChild
from pyVim import vmconfig

from pyVmomi import VmomiSupport, vim, vmodl

import volumeKVStore as kv

# defaults
DockVolsDir = "dockvols" # place in the same (with Docker VM) datastore
MaxDescrSize = 10000     # we assume files smaller that that to be descriptor files
MaxJsonSize = 1024 * 4   # max buf size for query json strings. Queries are limited in size
MaxSkipCount = 100       # max retries on VMCI Get Ops failures
DefaultDiskSize = "100mb"
BinLoc  = "/usr/lib/vmware/vmdkops/bin/"

# default log file. Should be synced with CI and make wrappers in ../*scripts
LogFile = "/var/log/vmware/docker-vmdk-plugin.log"

def LogSetup(logfile):
    logging.basicConfig(filename=logfile,
                        level=logging.DEBUG,
                        format='%(asctime)-12s %(process)d [%(levelname)s] %(message)s',
                        datefmt='%x %X')
    logging.info("===" + time.strftime('%x %X %Z') + "Starting vmdkops service ===")


# Run executable on ESX as needed for vmkfstools invocation (until normal disk create is written)
# Returns the integer return value and the stdout str on success and integer return value and
# the stderr str on error
def RunCommand(cmd):
   """RunCommand

   Runs command specified by user

   @param command to execute
   """
   logging.debug ("Running cmd %s" % cmd)

   p = subprocess.Popen(cmd,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE,
                        shell=True)
   o, e = p.communicate()
   s = p.returncode

   if s != 0:
       return (s, e)

   return (s, o)

# returns error, or None for OK
# opts is  dictionary of {option: value}.
# for now we care about size and (maybe) policy
def createVMDK(vmdkPath, volName, opts=""):
	logging.info ("*** createVMDK: %s opts=%s" % (vmdkPath, opts))
	if os.path.isfile(vmdkPath):
		return err("File %s already exists" % vmdkPath)

	if not opts or not "size" in opts:
		size = DefaultDiskSize
		logging.debug("SETTING DEFAULT SIZE to " +  size)
	else:
		size = str(opts["size"])
		#TODO: check it is compliant format, and correct if possible
		logging.debug("SETTING  SIZE to " + size)

        cmd = "/sbin/vmkfstools -d thin -c {0} {1}".format(size, vmdkPath)
        rc, out = RunCommand(cmd)

        if rc != 0:
            if removeVMDK(vmdkPath) == None:
                return err("Failed to create %s." % vmdkPath)
            else:
                return err("Unable to create %s and unable to delete volume. Please delete it manually." % vmdkPath)

        # Create the kv store for the disk before its attached
        ret = kv.create(vmdkPath, "detached", opts)
        if ret != True:
           logging.warning ("Failed creating meta data store for %s" % vmdkPath)
           removeVMDK(vmdkPath)
           return err("Failed to create meta-data store for %s" % vmdkPath)

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
            logging.warning ("Failed to format %s. %s" % (vmdkPath, out))
            if removeVMDK(vmdkPath) == None:
                return err("Failed to format %s." % vmdkPath)
            else:
                return err("Unable to format %s and unable to delete volume. Please delete it manually." % vmdkPath)
	return None

#returns error, or None for OK
def removeVMDK(vmdkPath):
	logging.info("*** removeVMDK: " + vmdkPath)
        cmd = "/sbin/vmkfstools -U {0}".format(vmdkPath)
        rc, out = RunCommand(cmd)
        if rc != 0:
            return err("Failed to remove %s. %s" % (vmdkPath, out))

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
	except vim.fault.NotAuthenticated as ex:
		connectLocal() 					#  retry
		vm = FindChild(GetVmFolder(), vmName)

	if not vm:
		raise Exception("VM" + vmName + "not found")

	return vm

#returns error, or None for OK
def attachVMDK(vmdkPath, vmName):
	vm = findVmByName(vmName)
	logging.info ("*** attachVMDK: " + vmdkPath + " to "   + vmName +
                  " uuid=" + vm.config.uuid)
	return disk_attach(vmdkPath, vm)

#returns error, or None for OK
def detachVMDK(vmdkPath, vmName):
	vm = findVmByName(vmName)
	logging.info("*** detachVMDK: " + vmdkPath + " from "  + vmName +
                 " VM uuid=" + vm.config.uuid)
	return disk_detach(vmdkPath, vm)


# check existence (and creates if needed) the path
# NOTE / TBD: for vsan we may need to use osfs_mkdir instead of regular os.mkdir
def getVolPath(vmConfigPath):
    path = os.path.join("/".join(vmConfigPath.split("/")[0:4]),  DockVolsDir)
    try:
        os.mkdir(path)
        logging.info(path +" created")
    except OSError:
		pass
    return path

def getVmdkName(path, volName):
    # form full name as <path-to-volumes>/<volname>.vmdk
    return  os.path.join(path, "%s.vmdk" % volName)

# gets the requests, calculates path for volumes, and calls the relevant handler
def executeRequest(vmName, vmId, configPath, cmd, volName, opts):
	# get /vmfs/volumes/<volid> path on ESX:
    path     = getVolPath(configPath)
    vmdkPath = getVmdkName(path, volName)


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
        return err("Unknown command:" + cmd)


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


# helper to logging.info out all VMs info
def printVMInfo(si):
	container = si.content.rootFolder  # starting point to look into
	containerView = si.content.viewManager.CreateContainerView(container,
			type=[vim.VirtualMachine],
			recursive=True)

	for child in containerView.view:
		summary = child.summary
		logging.info("Name       : " + summary.config.name)
		logging.info("Path       : " + summary.config.vmPathName)
		logging.info("Guest      : " + summary.config.guestFullName)
		logging.info("Instance UUID : " + summary.config.instanceUuid)
		logging.info("Bios UUID     : " + summary.config.uuid)



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
			logging.debug("findDeviceByPath: MATCH: " + vmdkPath)
			return d

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
    logging.warning("Warning: PVSCI adapter is missing - trying to add one...")
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

  # Check if this disk is already attached, skip the
  # attach below if it is

  status = kv.get(vmdkPath, 'status')
  logging.info("Attaching {0} with status {1}".format(vmdkPath,  status))
  if status and status != 'detached':
     vmUuid = kv.get(vmdkPath, 'attachedVMUuid')
     if vmUuid:
        if vmUuid == vm.config.uuid:
           msg = "{0} is already attached, skipping duplicate request.".format(vmdkPath)
        else:
           msg = "{0} is attached to VM ID={1}, skipping attach request".format(vmdkPath, vmUuid)
        logging.warning(msg)
        return err(msg)

  # Now find a slot on the controller  , if needed
  if not diskSlot:
    taken = set([dev.unitNumber for dev in devices
                 if type(dev) == vim.VirtualDisk and dev.controllerKey == controllerKey])
    # search in 15 slots, with unit_number 7 reserved for scsi controller
    availSlots = set(range (0,6) + range (8,16))  - taken

    if len(availSlots) == 0:
      raise StandardError("We don't support this many disks yet")

    diskSlot = availSlots.pop()
    logging.debug(" controllerKey=%d slot=%d" % (controllerKey, diskSlot))
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
      volMeta = kv.getAll(vmdkPath)
      if volMeta:
         volMeta['status'] = 'attached'
         volMeta['attachedVMUuid'] = vm.config.uuid
         kv.setAll(vmdkPath, volMeta)
  except vim.fault.VimFault as ex:
      return err(ex.msg)
  else:
    logging.info("disk attached " + vmdkPath)

  return None


def err(string):
    return {u'Error': string}


# detach disk (by full path) from a vm
# returns None or err(msg)
def disk_detach(vmdkPath, vm):

  # Find device object by vmkd path
  # TODO : the right way is to FIND BY disk UUID.
  device = findDeviceByPath(vmdkPath, vm)

  if not device:
      # TBD: Docker asks to detach something not attached :-) .
      msg = "**** INTERNAL ERROR: can't find the disk to detach: " + vmdkPath
      logging.error(msg)
      return err(msg)

  spec = vim.vm.ConfigSpec()
  dev_changes = []

  disk_spec = vim.vm.device.VirtualDeviceSpec()
  disk_spec.operation = vim.vm.device.VirtualDeviceSpec.Operation.remove
  disk_spec.device = device
  dev_changes.append(disk_spec)
  spec.deviceChange = dev_changes

  try:
     wait_for_tasks(si, [vm.ReconfigVM_Task(spec=spec)])	# si is global
     volMeta = kv.getAll(vmdkPath)
     if volMeta:
        volMeta['status'] = 'detached'
        del volMeta['attachedVMUuid']
        kv.setAll(vmdkPath, volMeta)
  except vim.fault.GenericVmConfigFault as ex:
     for f in ex.faultMessage:
        logging.warning(f.message)
     return err("Failed to detach " + vmdkPath)

  logging.info("Disk detached " + vmdkPath)
  return None



def signal_handler_stop(signalnum, frame):
    logging.warn("Received stop signal num: " + `signalnum`)
    sys.exit(0)


# load VMCI shared lib , listen on vSocket in main loop, handle requests
def handleVmciRequests():
	# Load and use DLL with vsocket shim to listen for docker requests
	lib = cdll.LoadLibrary(BinLoc + "/libvmci_srv.so")

	bsize = MaxJsonSize
	txt = create_string_buffer(bsize)

	cartel = c_int32()
	sock = lib.vmci_init()
	skipCount = MaxSkipCount # retries for vmci_get_one_op failures
	while True:
		c = lib.vmci_get_one_op(sock, byref(cartel), txt, c_int(bsize))
		logging.debug("lib.vmci_get_one_op returns %d, buffer '%s'" %(c, txt.value))

		if c == -1:
			# VMCI Get Ops can self-correct by reoping sockets internally. Give it a chance.
			logging.warning("VMCI Get Ops failed - ignoring and moving on.")
			skipCount = skipCount - 1
			if skipCount <= 0:
				raise Exception("Too many errors from VMCI Get Ops - giving up.")
			continue
		else:
			skipCount = MaxSkipCount # reset the counter, just in case

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

		try:
			req = json.loads(txt.value, "utf-8")
		except ValueError as e:
			ret = {u'Error': "Failed to parse json '%s'." % (txt,value)}
		else:
			# note: Connection can time out on idle. TODO: to refresh in that case
			details = req["details"]
			opts = details["Opts"] if "Opts" in details else None
			ret = executeRequest(vmName, vmId, cfgPath, req["cmd"], details["Name"], opts)
			logging.debug("executeRequest ret = %s" % ret)

		err = lib.vmci_reply(c, c_char_p(json.dumps(ret)))
		logging.debug("lib.vmci_reply: VMCI replied with errcode %s " % err)

	lib.close(sock) # close listening socket when the loop is over


def main():
    LogSetup(LogFile)
    signal.signal(signal.SIGINT, signal_handler_stop)
    signal.signal(signal.SIGTERM, signal_handler_stop)

    try:
        kv.init()
        global si  # we maintain only one connection
        si = connectLocal()
        printVMInfo(si) # just making sure we can do it - logging.info
        handleVmciRequests()
    except Exception, e:
    	logging.exception(e)

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
