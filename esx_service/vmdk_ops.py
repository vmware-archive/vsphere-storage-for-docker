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
ESX-side service  handling VMDK requests from VMCI clients

The requests are JSON formatted.

All operations are using requester VM (docker host) datastore and
"Name" in request refers to vmdk basename
VMDK name is formed as [vmdatastore] dockvols/"Name".vmdk

Commands ("cmd" in request):
		"create" - create a VMDK in "[vmdatastore] dvol"
		"remove" - remove a VMDK. We assume it's not open, and fail if it is
		"list"   - enumerate VMDKs
		"get"    - get info about an individual volume (vmdk)
		"attach" - attach a VMDK to the requesting VM
		"detach" - detach a VMDK from the requesting VM (assuming it's unmounted)

'''

import atexit
import getopt
import json
import logging
import os
import os.path
import re
import signal
import subprocess
import sys
import traceback
import threading
import time
from ctypes import *

from vmware import vsi

import pyVim
from pyVim.connect import Connect, Disconnect
from pyVim import vmconfig

from pyVmomi import VmomiSupport, vim, vmodl

sys.dont_write_bytecode = True

# Location of utils used by the plugin.
TOP_DIR = "/usr/lib/vmware/vmdkops"
BIN_LOC  = os.path.join(TOP_DIR, "bin")
LIB_LOC  = os.path.join(TOP_DIR, "lib")
LIB_LOC64 = os.path.join(TOP_DIR, "lib64")
PY_LOC  = os.path.join(TOP_DIR, "Python")

# We won't accept names longer than that
MAX_VOL_NAME_LEN = 100
MAX_DS_NAME_LEN  = 100

# vmdkops python utils are in PY_LOC, so add to path.
sys.path.insert(0, PY_LOC)


import log_config
import volume_kv as kv
import vmdk_utils
import vsan_policy
import vsan_info

# Python version 3.5.1
PYTHON64_VERSION = 50659824

# External tools used by the plugin.
OBJ_TOOL_CMD = "/usr/lib/vmware/osfs/bin/objtool open -u "
OSFS_MKDIR_CMD = "/usr/lib/vmware/osfs/bin/osfs-mkdir -n "
MKFS_CMD = BIN_LOC + "/mkfs.ext4 -qF -L "
VMDK_CREATE_CMD = "/sbin/vmkfstools"
VMDK_DELETE_CMD = "/sbin/vmkfstools -U "

# Defaults
DOCK_VOLS_DIR = "dockvols"  # place in the same (with Docker VM) datastore
MAX_JSON_SIZE = 1024 * 4  # max buf size for query json strings. Queries are limited in size
MAX_SKIP_COUNT = 16       # max retries on VMCI Get Ops failures

# Volume data returned on Get request
CAPACITY = 'capacity'
SIZE = 'size'
ALLOCATED = 'allocated'
LOCATION = 'datastore'
CREATED_BY_VM = 'created by VM'
ATTACHED_TO_VM = 'attached to VM'

# Virtual machine power states
VM_POWERED_OFF = "poweredOff"

# Maximum number of PVSCSI targets
PVSCSI_MAX_TARGETS = 16

# Service instance provide from connection to local hostd
si = None

# VMCI library used to communicate with clients
lib = None

# Run executable on ESX as needed for vmkfstools invocation (until normal disk create is written)
# Returns the integer return value and the stdout str on success and integer return value and
# the stderr str on error
def RunCommand(cmd):
    """RunCommand

   Runs command specified by user

   @param command to execute
   """
    logging.debug("Running cmd %s", cmd)

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
def createVMDK(vmdk_path, vm_name, vol_name, opts={}):
    logging.info("*** createVMDK: %s opts = %s", vmdk_path, opts)
    if os.path.isfile(vmdk_path):
        return err("File %s already exists" % vmdk_path)

    try:
        validate_opts(opts, vmdk_path)
    except ValidationError as e:
        return err(e.msg)

    cmd = make_create_cmd(opts, vmdk_path)
    rc, out = RunCommand(cmd)
    if rc != 0:
        return err("Failed to create %s. %s" % (vmdk_path, out))

    if not create_kv_store(vm_name, vmdk_path, opts):
        msg = "Failed to create metadata kv store for {0}".format(vmdk_path)
        logging.warning(msg)
        removeVMDK(vmdk_path)
        return err(msg)

    return formatVmdk(vmdk_path, vol_name)


def make_create_cmd(opts, vmdk_path):
    """ Return the command used to create a VMDK """
    if not "size" in opts:
        size = kv.DEFAULT_DISK_SIZE
    else:
        size = str(opts["size"])
    logging.debug("Setting vmdk size to %s for %s", size, vmdk_path)

    if not kv.DISK_ALLOCATION_FORMAT in opts:
        disk_format = kv.DEFAULT_ALLOCATION_FORMAT
    else:
        disk_format = str(opts[kv.DISK_ALLOCATION_FORMAT])
    logging.debug("Setting vmdk disk allocation format to %s for %s",
                  disk_format, vmdk_path)

    if kv.VSAN_POLICY_NAME in opts:
        # Note that the --policyFile option gets ignored if the
        # datastore is not VSAN
        policy_file = vsan_policy.policy_path(opts[kv.VSAN_POLICY_NAME])
        return "{0} -d {1} -c {2} --policyFile {3} {4}".format(VMDK_CREATE_CMD, disk_format, size,
                                                               policy_file, vmdk_path)
    else:
        return "{0} -d {1} -c {2} {3}".format(VMDK_CREATE_CMD, disk_format, size, vmdk_path)


def create_kv_store(vm_name, vmdk_path, opts):
    """ Create the metadata kv store for a volume """
    vol_meta = {kv.STATUS: kv.DETACHED,
                kv.VOL_OPTS: opts,
                kv.CREATED: time.asctime(time.gmtime()),
                kv.CREATED_BY: vm_name}
    return kv.create(vmdk_path, vol_meta)


def validate_opts(opts, vmdk_path):
    """
    Validate available options. Current options are:
     * size - The size of the disk to create
     * vsan-policy-name - The name of an existing policy to use
     * diskformat - The allocation format of allocated disk
    """
    valid_opts = [kv.SIZE, kv.VSAN_POLICY_NAME, kv.DISK_ALLOCATION_FORMAT, kv.ATTACH_AS]
    defaults = [kv.DEFAULT_DISK_SIZE, kv.DEFAULT_VSAN_POLICY, kv.DEFAULT_ALLOCATION_FORMAT, kv.DEFAULT_ATTACH_AS]
    invalid = frozenset(opts.keys()).difference(valid_opts)
    if len(invalid) != 0:
        msg = 'Invalid options: {0} \n'.format(list(invalid)) \
               + 'Valid options and defaults: ' \
               + '{0}'.format(zip(list(valid_opts), defaults))
        raise ValidationError(msg)

    if kv.SIZE in opts:
        validate_size(opts[kv.SIZE])
    if kv.VSAN_POLICY_NAME in opts:
        validate_vsan_policy_name(opts[kv.VSAN_POLICY_NAME], vmdk_path)
    if kv.DISK_ALLOCATION_FORMAT in opts:
        validate_disk_allocation_format(opts[kv.DISK_ALLOCATION_FORMAT])
    if kv.ATTACH_AS in opts:
        validate_attach_as(opts[kv.ATTACH_AS])


def validate_size(size):
    """
    Ensure size is given in a human readable format <int><unit> where int is an
    integer and unit is either 'mb', 'gb', or 'tb'. e.g. 22mb
    """
    if not size.lower().endswith(('kb', 'mb', 'gb', 'tb'
                                  )) or not size[:-2].isdigit():
        msg = ('Invalid format for size. \n'
               'Valid sizes must be of form X[kKmMgGtT]b where X is an'
               'integer. Default = 100mb')
        raise ValidationError(msg)


def validate_vsan_policy_name(policy_name, vmdk_path):
    """
    Ensure that the policy file exists
    """
    if not vsan_info.is_on_vsan(vmdk_path):
        raise ValidationError('Cannot use a VSAN policy on a non-VSAN datastore')

    if not vsan_policy.policy_exists(policy_name):
        raise ValidationError('Policy {0} does not exist'.format(policy_name))

def validate_disk_allocation_format(alloc_format):
    """
    Ensure format is valid.
    """
    if not alloc_format in kv.VALID_ALLOCATION_FORMATS :
        raise ValidationError("Disk Allocation Format \'{0}\' is not supported."
                              " Valid options are: {1}".format(alloc_format, kv.VALID_ALLOCATION_FORMATS))

def validate_attach_as(attach_type):
    """
    Ensure that we recognize the attach type
    """
    if not attach_type in kv.ATTACH_AS_TYPES :
        raise ValidationError("Attach type '{0}' is not supported."
                              " Valid options are: {1}".format(attach_type, kv.ATTACH_AS_TYPES))

# Returns the UUID if the vmdk_path is for a VSAN backed.
def get_vsan_uuid(vmdk_path):
    f = open(vmdk_path)
    data = f.read()
    f.close()

    # For now we look for a VSAN URI, later vvol.
    exp = re.compile("RW .* VMFS \"vsan:\/\/(.*)\"")

    try:
        return exp.search(data).group(1)
    except:
        return None

# Create and return the devFS path for a VSAN object UUID.
# Also returns true if clean up is needed.
# cleanup_vsan_devfs should be called to clean up before
# removing the VSAN object.
def get_vsan_devfs_path(uuid):

    logging.debug("Got volume UUID %s", uuid)

    # Objtool creates a link thats usable to
    # read write to vsan object.
    cmd = "{0} {1}".format(OBJ_TOOL_CMD, uuid)
    rc, out = RunCommand(cmd)
    fpath="/vmfs/devices/vsan/{0}".format(uuid)
    if rc == 0 and os.path.isfile(fpath):
        return fpath, True
    logging.error("Failed to create devFS node for %s", uuid)
    return None, False

# Clean up vsan devfs path
def cleanup_vsan_devfs_path(devfs_path):
    try:
        os.remove(devfs_path)
        logging.debug("Unlinked %s", devfs_path)
        return True
    except OSError as ex:
        logging.error("Failed to remove backing device %s",
                      devfs_path, str(ex))
    return False

# Returns the flat file for a VMDK.
def get_backing_flat_file(vmdk_path):
    return vmdk_path.replace(".vmdk", "-flat.vmdk")

# Return a backing file path for given vmdk path or none
# if a backing can't be found. Returns True if clean up
# is needed. Do not forget to call cleanup_backing_device when done.
def get_backing_device(vmdk_path):
    flatBacking = get_backing_flat_file(vmdk_path)
    if os.path.isfile(flatBacking):
        return flatBacking, False

    uuid = get_vsan_uuid(vmdk_path)

    if uuid:

        return get_vsan_devfs_path(uuid)

    return None, False

def cleanup_backing_device(backing, cleanup_device):
    if cleanup_device:
    	return cleanup_vsan_devfs_path(backing)
    return True

def formatVmdk(vmdk_path, vol_name):
    # Get backing for given vmdk path. This is the backing
    # device that will be formatted.
    backing, needs_cleanup = get_backing_device(vmdk_path)

    if backing is None:
        logging.warning("Failed to format %s.", vmdk_path)
        return err("Failed to format %s." % vmdk_path)

    # Format it as ext4.
    cmd = "{0} {1} {2}".format(MKFS_CMD, vol_name, backing)
    rc, out = RunCommand(cmd)

    # clean up any resources backing file might have created
    # during get_backing_device function call.
    cleanup_backing_device(backing, needs_cleanup)

    if rc != 0:
        logging.warning("Failed to format %s - %s", vmdk_path, out)
        if removeVMDK(vmdk_path) == None:
            return err("Failed to format %s." % vmdk_path)
        else:
            return err(
                "Unable to format %s and unable to delete volume. Please delete it manually."
                % vmdk_path)
    return None

# Return volume ingo
def vol_info(vol_meta, vol_size_info, datastore):
    vinfo = {CREATED_BY_VM : vol_meta[kv.CREATED_BY],
             kv.CREATED : vol_meta[kv.CREATED],
             kv.STATUS : vol_meta[kv.STATUS]}

    vinfo[CAPACITY] = {}
    vinfo[CAPACITY][SIZE] = vol_size_info[SIZE]
    vinfo[CAPACITY][ALLOCATED] = vol_size_info[ALLOCATED]
    vinfo[LOCATION] = datastore

    if kv.ATTACHED_VM_NAME in vol_meta:
       vinfo[ATTACHED_TO_VM] = vol_meta[kv.ATTACHED_VM_NAME]
    if kv.VSAN_POLICY_NAME in vol_meta:
       vinfo[kv.kv.VSAN_POLICY_NAME] = vol_meta[kv.kv.VSAN_POLICY_NAME]
    if kv.VOL_OPTS in vol_meta and \
       kv.DISK_ALLOCATION_FORMAT in vol_meta[kv.VOL_OPTS]:
       vinfo[kv.DISK_ALLOCATION_FORMAT] = vol_meta[kv.VOL_OPTS][kv.DISK_ALLOCATION_FORMAT]

    return vinfo


# Return error, or None for OK
def removeVMDK(vmdk_path):
    logging.info("*** removeVMDK: %s", vmdk_path)
    cmd = "{0} {1}".format(VMDK_DELETE_CMD, vmdk_path)
    rc, out = RunCommand(cmd)
    if rc != 0:
        return err("Failed to remove %s. %s" % (vmdk_path, out))

    return None


def getVMDK(vmdk_path, vol_name, datastore):
    """Checks if the volume exists, and returns error if it does not"""
    # Note: will return more Volume info here, when Docker API actually accepts it
    if not os.path.isfile(vmdk_path):
        return err("Volume {0} not found (file: {1})".format(vol_name, vmdk_path))
    # Return volume info - volume policy, size, allocated capacity, allocation
    # type, creat-by, create time.
    return vol_info(kv.getAll(vmdk_path), kv.get_vol_info(vmdk_path), datastore)


def listVMDK(vm_datastore):
    """
    Returns a list of volume names (note: may be an empty list).
    Each volume name is returned as either `volume@datastore`, or just `volume`
    for volumes on vm_datastore
    """
    vmdks = vmdk_utils.get_volumes()
    # build  fully qualified vol name for each volume found
    return [{u'Name': get_full_vol_name(x['filename'], x['datastore'], vm_datastore),
             u'Attributes': {}} \
            for x in vmdks]


# Return VM managed object, reconnect if needed. Throws if fails twice.
def findVmByUuid(vm_uuid):
    vm = None
    try:
        vm = si.content.searchIndex.FindByUuid(None, vm_uuid, True, False)
    except Exception as ex:
        logging.warning("Failed to find VM by uuid=%s, retrying...\n%s",
                        vm_uuid, str(ex))
    if vm:
        return vm
    #
    # Retry. It can throw if connect/search fails. But search can't return None
    # since we get UUID from VMM so VM must exist
    #
    connectLocal()
    vm = si.content.searchIndex.FindByUuid(None, vm_uuid, True, False)
    logging.info("Found VM name=%s, id=%s ", vm.config.name, vm_uuid)
    return vm


# Return error, or None for OK.
def attachVMDK(vmdk_path, vm_uuid):
    vm = findVmByUuid(vm_uuid)
    logging.info("*** attachVMDK: %s to %s VM uuid = %s",
                 vmdk_path, vm.config.name, vm_uuid)
    return disk_attach(vmdk_path, vm)


# Return error, or None for OK.
def detachVMDK(vmdk_path, vm_uuid):
    vm = findVmByUuid(vm_uuid)
    logging.info("*** detachVMDK: %s from %s VM uuid = %s",
                 vmdk_path, vm.config.name, vm_uuid)
    return disk_detach(vmdk_path, vm)


# Check existence (and creates if needed) the path for docker volume VMDKs
def get_vol_path(datastore):
    # The folder for Docker volumes is created on <datastore>/DOCK_VOLS_DIR
    path = os.path.join("/vmfs/volumes", datastore, DOCK_VOLS_DIR)

    if os.path.isdir(path):
        # If the path exists then return it as is.
        logging.debug("Found %s, returning", path)
        return path

    # The osfs tools are usable for all datastores
    cmd = "{0} {1}".format(OSFS_MKDIR_CMD, path)
    rc, out = RunCommand(cmd)
    if rc == 0:
        logging.info("Created %s", path)
        return path

    logging.warning("Failed to create %s", path)
    return None


def known_datastores():
    """returns names of know datastores"""
    return [i[0] for i in vmdk_utils.get_datastores()]

def parse_vol_name(full_vol_name):
    """
    Parses volume[@datastore] and returns (volume, datastore)
    On parse errors raises ValidationError with syntax explanation
    """
    # Parse volume name with regexp package
    #
    # Caveat: we block '-NNNNNN' in end of volume name to make sure that volume
    # name never conflicts with VMDK snapshot name (e.g. 'disk-000001.vmdk').
    # Note that N is a digit and there are exactly 6 of them (hardcoded in ESXi)
    # vmdk_utils.py:list_vmdks() explicitly relies on this assumption.
    #
    try:
        at = full_vol_name.rindex('@')
        vol_name = full_vol_name[:at]
        ds_name = full_vol_name[at + 1:]
    except ValueError:
        # '@' not found
        vol_name = full_vol_name
        ds_name = None
    # now block the '-NNNNN' volume names
    if re.match(vmdk_utils.SNAP_NAME_REGEXP, vol_name):
        raise ValidationError("Volume names ending with '-NNNNNN' (where N is a digit) are not supported")
    if len(vol_name) > MAX_VOL_NAME_LEN:
        raise ValidationError("Volume name is too long (max len is {0})".format(MAX_VOL_NAME_LEN))
    if ds_name and len(ds_name) > MAX_DS_NAME_LEN:
        raise ValidationError("Datastore name is too long (max len is {0})".format(MAX_DS_NAME_LEN))
    return vol_name, ds_name


def get_full_vol_name(vmdk_name, datastore, vm_datastore):
    """
    Forms full volume name from vmdk file name an datastore as volume@datastore
    For volumes on vm_datastore, just returns volume name
    """
    vol_name = vmdk_utils.strip_vmdk_extension(vmdk_name)
    logging.debug("get_full_vol_name: %s %s %s", vmdk_name, datastore, vm_datastore)
    if datastore == vm_datastore:
        return vol_name
    return "{0}@{1}".format(vol_name, datastore)


# TBD - move to vmdk_utils
def get_datastore_name(config_path):
    """Returns datastore NAME in config_path (not url-name which may be used in path)"""
    # path is always /vmfs/volumes/<datastore>/... , so extract datastore:
    config_ds_name = config_path.split("/")[3]
    ds_name = [x[0] for x in vmdk_utils.get_datastores() \
                    if x[0] == config_ds_name or x[1] == config_ds_name ]
    if len(ds_name) != 1:
        logging.error("get_datastore_name: found more than one match: %s" % ds_name)
    logging.debug("get_datastore_name: path=%s name=%s" % (config_ds_name, ds_name))
    return ds_name[0]

# gets the requests, calculates path for volumes, and calls the relevant handler
def executeRequest(vm_uuid, vm_name, config_path, cmd, full_vol_name, opts):
    """
    Executes a <cmd> request issused from a VM.
    The request is about volume <full_volume_name> in format volume@datastore.
    If @datastore is omitted, the one where the VM resides is used.
    For VM, the function gets vm_uuid, vm_name and config_path
    <opts> is a json options string blindly passed to a specific operation

    Returns None (if all OK) or error string
    """

    vm_datastore = get_datastore_name(config_path)
    if cmd == "list":
        return listVMDK(vm_datastore)

    try:
        vol_name, datastore = parse_vol_name(full_vol_name)
    except ValidationError as ex:
        return err(str(ex))
    if not datastore:
        datastore = vm_datastore
    elif datastore not in known_datastores():
        return err("Invalid datastore '%s'.\n" \
                   "Known datastores: %s.\n" \
                   "Default datastore: %s" \
                   % (datastore, ", ".join(known_datastores()), vm_datastore))

    # get /vmfs/volumes/<volid>/dockvols path on ESX:
    path = get_vol_path(datastore)

    if path is None:
        return err("Failed to initialize volume path {0}".format(path))

    vmdk_path = vmdk_utils.get_vmdk_path(path, vol_name)

    if cmd == "get":
        response = getVMDK(vmdk_path, vol_name, datastore)
    elif cmd == "create":
        response = createVMDK(vmdk_path, vm_name, vol_name, opts)
    elif cmd == "remove":
        response = removeVMDK(vmdk_path)
    elif cmd == "attach":
        response = attachVMDK(vmdk_path, vm_uuid)
    elif cmd == "detach":
        response = detachVMDK(vmdk_path, vm_uuid)
    else:
        return err("Unknown command:" + cmd)

    return response

def connectLocal():
    '''
	connect and do stuff on local machine
	'''
    global si  #

    # Connect to localhost as dcui
    # User "dcui" is a local Admin that does not lose permissions
    # when the host is in lockdown mode.
    si = pyVim.connect.Connect(host='localhost', user='dcui')
    if not si:
        raise SystemExit("Failed to connect to localhost as 'dcui'.")

    atexit.register(pyVim.connect.Disconnect, si)

    # set out ID in context to be used in request - so we'll see it in logs
    reqCtx = VmomiSupport.GetRequestContext()
    reqCtx["realUser"] = 'dvolplug'
    return si


def findDeviceByPath(vmdk_path, vm):
    logging.debug("findDeviceByPath: Looking for device {0}".format(vmdk_path))
    for d in vm.config.hardware.device:
        if type(d) != vim.vm.device.VirtualDisk:
            continue

        # Disks of all backing have a backing object with a filename attribute.
        # The filename identifies the virtual disk by name and can be used
        # to match with the given volume name.
        # Filename format is as follows:
        #   "[<datastore name>] <parent-directory>/<vmdk-descriptor-name>"
        backing_disk = d.backing.fileName.split(" ")[1]

        # Construct the parent dir and vmdk name, resolving
        # links if any.
        dvol_dir = os.path.dirname(vmdk_path)
        real_vol_dir = os.path.basename(os.path.realpath(dvol_dir))
        virtual_disk = os.path.join(real_vol_dir, os.path.basename(vmdk_path))
        if virtual_disk == backing_disk:
            logging.debug("findDeviceByPath: MATCH: %s", backing_disk)
            return d
    return None

# Find the PCI slot number
def get_controller_pci_slot(vm, pvscsi, key_offset):
    ''' Return PCI slot number of the given PVSCSI controller
    Input parameters:
    vm: VM configuration
    pvscsi: given PVSCSI controller
    key_offset: offset from the bus number, controller_key - key_offset
    is equal to the slot number of this given PVSCSI controller
    '''
    if pvscsi.slotInfo:
       return str(pvscsi.slotInfo.pciSlotNumber)
    else:
       # Slot number is got from from the VM config
       key = 'scsi{0}.pciSlotNumber'.format(pvscsi.key -
                                            key_offset)
       slot = [cfg for cfg in vm.config.extraConfig \
               if cfg.key == key]
       # If the given controller exists
       if slot:
          return slot[0].value
       else:
          return None

def dev_info(unit_number, pci_slot_number):
    '''Return a dictionary with Unit/Bus for the vmdk (or error)'''
    return {'Unit': str(unit_number),
            'ControllerPciSlotNumber': pci_slot_number}

def reset_vol_meta(vmdk_path):
    '''Clears metadata for vmdk_path'''
    vol_meta = kv.getAll(vmdk_path)
    if not vol_meta:
       vol_meta = {}
    logging.debug("Reseting meta-data for disk=%s", vmdk_path)
    if set(vol_meta.keys()) & {kv.STATUS, kv.ATTACHED_VM_UUID, kv.ATTACHED_VM_NAME}:
          logging.debug("Old meta-data for %s was (status=%s VM name=%s uuid=%s)",
                        vmdk_path, vol_meta[kv.STATUS],
                        vol_meta[kv.ATTACHED_VM_NAME],
                        vol_meta[kv.ATTACHED_VM_UUID])
    vol_meta[kv.STATUS] = kv.DETACHED
    vol_meta[kv.ATTACHED_VM_UUID] = None
    vol_meta[kv.ATTACHED_VM_NAME] = None
    if not kv.setAll(vmdk_path, vol_meta):
       msg = "Failed to save volume metadata for {0}.".format(vmdk_path)
       logging.warning("reset_vol_meta: " + msg)
       return err(msg)

def setStatusAttached(vmdk_path, vm):
    '''Sets metadata for vmdk_path to (attached, attachedToVM=uuid'''
    logging.debug("Set status=attached disk=%s VM name=%s uuid=%s", vmdk_path,
                  vm.config.name, vm.config.uuid)
    vol_meta = kv.getAll(vmdk_path)
    if not vol_meta:
        vol_meta = {}
    vol_meta[kv.STATUS] = kv.ATTACHED
    vol_meta[kv.ATTACHED_VM_UUID] = vm.config.uuid
    vol_meta[kv.ATTACHED_VM_NAME] = vm.config.name
    if not kv.setAll(vmdk_path, vol_meta):
        logging.warning("Attach: Failed to save Disk metadata for %s", vmdk_path)


def setStatusDetached(vmdk_path):
    '''Sets metadata for vmdk_path to "detached"'''
    logging.debug("Set status=detached disk=%s", vmdk_path)
    vol_meta = kv.getAll(vmdk_path)
    if not vol_meta:
        vol_meta = {}
    vol_meta[kv.STATUS] = kv.DETACHED
    # If attachedVMName is present, so is attachedVMUuid
    try:
        del vol_meta[kv.ATTACHED_VM_UUID]
        del vol_meta[kv.ATTACHED_VM_NAME]
    except:
        pass
    if not kv.setAll(vmdk_path, vol_meta):
        logging.warning("Detach: Failed to save Disk metadata for %s", vmdk_path)


def getStatusAttached(vmdk_path):
    '''Returns (attached, uuid, attach_as) tuple. For 'detached' status uuid is None'''

    vol_meta = kv.getAll(vmdk_path)
    try:
        attach_as = vol_meta[kv.VOL_OPTS][kv.ATTACH_AS]
    except:
        attach_as = kv.DEFAULT_ATTACH_AS

    if not vol_meta or kv.STATUS not in vol_meta:
        return False, None, attach_as

    attached = (vol_meta[kv.STATUS] == kv.ATTACHED)
    try:
        uuid = vol_meta[kv.ATTACHED_VM_UUID]
    except:
        uuid = None
    return attached, uuid, attach_as

def handle_stale_attach(vmdk_path, kv_uuid):
       '''
       Clear volume state for cases where the VM that attached the disk
       earlier is powered off or removed. Detach the disk from the VM
       if it's powered off.
       '''
       cur_vm = findVmByUuid(kv_uuid)

       if cur_vm:
          # Detach the disk only if VM is powered off
          if cur_vm.runtime.powerState == VM_POWERED_OFF:
             logging.info("Detaching disk %s from VM(powered off) - %s\n",
                             vmdk_path, cur_vm.config.name)
             device = findDeviceByPath(vmdk_path, cur_vm)
             if device:
                msg = disk_detach_int(vmdk_path, cur_vm, device)
                if msg:
                   msg += " failed to detach disk {0} from VM={1}.".format(vmdk_path,
                                                                           cur_vm.config.name)
                   return err(msg)
             else:
                logging.warning("Failed to find disk %s in powered off VM - %s, resetting volume metadata\n",
                                vmdk_path, cur_vm.config.name)
                ret = reset_vol_meta(vmdk_path)
                if ret:
                   return ret
          else:
             msg = "Disk {0} already attached to VM={1}".format(vmdk_path,
                                                                cur_vm.config.name)
             return err(msg)
       else:
          logging.warning("Failed to find VM %s that attached the disk %s, resetting volume metadata",
                          cur_vm.config.name, vmdk_path)
          ret = reset_vol_meta(vmdk_path)
          if ret:
             return ret

def add_pvscsi_controller(vm, controllers, max_scsi_controllers, offset_from_bus_number):
    ''' 
    Add a new PVSCSI controller, return (controller_key, err) pair
    '''
    # find empty bus slot for the controller:
    taken = set([c.busNumber for c in controllers])
    avail = set(range(0, max_scsi_controllers)) - taken

    key = avail.pop()  # bus slot
    controller_key = key + offset_from_bus_number
    disk_slot = 0
    controller_spec = vim.VirtualDeviceConfigSpec(
        operation='add',
        device=vim.ParaVirtualSCSIController(key=controller_key,
                                                busNumber=key,
                                                sharedBus='noSharing', ), )
    # changes spec content goes here
    pvscsi_change = []
    pvscsi_change.append(controller_spec)
    spec = vim.vm.ConfigSpec()
    spec.deviceChange = pvscsi_change

    try:
        wait_for_tasks(si, [vm.ReconfigVM_Task(spec=spec)])
    except vim.fault.VimFault as ex:
        msg=("Failed to add PVSCSI Controller: %s", ex.msg)
        return None, err(msg)
    logging.debug("Added a PVSCSI controller, controller_id=%d", controller_key)    
    return controller_key, None

def find_disk_slot_in_controller(vm, devices, pvsci, idx, offset_from_bus_number):
    '''
    Find an empty disk slot in the given controller, return disk_slot if an empty slot
    can be found, otherwise, return None
    '''
    disk_slot = None
    controller_key = pvsci[idx].key
    taken = set([dev.unitNumber
             for dev in devices
             if type(dev) == vim.VirtualDisk and dev.controllerKey ==
             controller_key])
    # search in 15 slots, with unit_number 7 reserved for scsi controller
    avail_slots = (set(range(0, 7)) | set(range(8, PVSCSI_MAX_TARGETS))) - taken
    logging.debug("idx=%d controller_key=%d avail_slots=%d", idx, controller_key, len(avail_slots))

    if len(avail_slots) != 0:
        disk_slot = avail_slots.pop()
        pci_slot_number = get_controller_pci_slot(vm, pvsci[idx],
                                                  offset_from_bus_number)
                 
        logging.debug("Find an available slot: controller_key = %d slot = %d", controller_key, disk_slot)
    else:
        logging.warning("No available slot in this controller: controller_key = %d", controller_key)
    return disk_slot        

def find_available_disk_slot(vm, devices, pvsci, offset_from_bus_number):
    '''
    Iterate through all the existing PVSCSI controllers attached to a VM to find an empty
    disk slot. Return disk_slot is an empty slot can be found, otherwise, return None
    '''
    idx = 0
    disk_slot = None
    while ((disk_slot is None) and (idx < len(pvsci))):
            disk_slot = find_disk_slot_in_controller(vm, devices, pvsci, idx, offset_from_bus_number)
            if (disk_slot is None):
                idx = idx + 1;
    return idx, disk_slot            
            
def disk_attach(vmdk_path, vm):
    '''
    Attaches *existing* disk to a vm on a PVSCI controller
    (we need PVSCSI to avoid SCSI rescans in the guest)
    return error or unit:bus numbers of newly attached disk.
    '''

    kv_status_attached, kv_uuid, attach_mode = getStatusAttached(vmdk_path)
    logging.info("Attaching {0} as {1}".format(vmdk_path, attach_mode))

    # If the volume is attached then check if the attach is stale (VM is powered off).
    # Otherwise, detach the disk from the VM it's attached to.
    if kv_status_attached and kv_uuid != vm.config.uuid:
       ret_err = handle_stale_attach(vmdk_path, kv_uuid)
       if ret_err:
          return ret_err

    # NOTE: vSphere is very picky about unit numbers and controllers of virtual
    # disks. Every controller supports 15 virtual disks, and the unit
    # numbers need to be unique within the controller and range from
    # 0 to 15 with 7 being reserved (for older SCSI controllers).
    # It is up to the API client to add controllers as needed.
    # SCSI Controller keys are in the range of 1000 to 1003 (1000 + bus_number).
    offset_from_bus_number = 1000
    max_scsi_controllers = 4


    devices = vm.config.hardware.device

    # get all scsi controllers (pvsci, lsi logic, whatever)
    controllers = [d for d in devices
                   if isinstance(d, vim.VirtualSCSIController)]

    # Check if this disk is already attached, and if it is - skip the disk
    # attach and the checks on attaching a controller if needed.
    device = findDeviceByPath(vmdk_path, vm)
    if device:
        # Disk is already attached.
        logging.warning("Disk %s already attached. VM=%s",
                        vmdk_path, vm.config.uuid)
        setStatusAttached(vmdk_path, vm)
        # Get that controller to which the device is configured for
        pvsci = [d for d in controllers
                   if type(d) == vim.ParaVirtualSCSIController and
                      d.key == device.controllerKey]
        
        return dev_info(device.unitNumber,
                        get_controller_pci_slot(vm, pvsci[0],
                                                offset_from_bus_number))
        

    # Disk isn't attached, make sure we have a PVSCI and add it if we don't
    # check if we already have a pvsci one
    pvsci = [d for d in controllers
             if type(d) == vim.ParaVirtualSCSIController]
    disk_slot = None         
    if len(pvsci) > 0:
        idx, disk_slot = find_available_disk_slot(vm, devices, pvsci, offset_from_bus_number);
        if (disk_slot is not None):
            controller_key = pvsci[idx].key
            pci_slot_number = get_controller_pci_slot(vm, pvsci[idx],
                                                      offset_from_bus_number)
            logging.debug("Find an available disk slot, controller_key=%d, slot_id=%d",
                          controller_key, disk_slot)
        
    if (disk_slot is None):
        disk_slot = 0  # starting on a fresh controller
        if len(controllers) >= max_scsi_controllers:
            msg = "Failed to place new disk - The maximum number of supported volumes has been reached."
            logging.error(msg + " VM=%s", vm.config.uuid)
            return err(msg)

        logging.info("Adding a PVSCSI controller")
        
        controller_key, ret_err = add_pvscsi_controller(vm, controllers, max_scsi_controllers, 
                                                        offset_from_bus_number)

        if (ret_err):
            return ret_err    
            
        # Find the controller just added
        devices = vm.config.hardware.device
        pvsci = [d for d in devices
                 if type(d) == vim.ParaVirtualSCSIController and
                 d.key == controller_key]
        pci_slot_number = get_controller_pci_slot(vm, pvsci[0],
                                                  offset_from_bus_number)
        logging.info("Added a PVSCSI controller, controller_key=%d pci_slot_number=%d",
                      controller_key, pci_slot_number)
    
    # add disk as independent, so it won't be snapshotted with the Docker VM
    disk_spec = vim.VirtualDeviceConfigSpec(
        operation='add',
        device=
        vim.VirtualDisk(backing=vim.VirtualDiskFlatVer2BackingInfo(
            fileName="[] " + vmdk_path,
            diskMode=attach_mode, ),
                        deviceInfo=vim.Description(
                            # TODO: use docker volume name here. Issue #292
                            label="dockerDataVolume",
                            summary="dockerDataVolume", ),
                        unitNumber=disk_slot,
                        controllerKey=controller_key, ), )
    disk_changes = []
    disk_changes.append(disk_spec)

    spec = vim.vm.ConfigSpec()
    spec.deviceChange = disk_changes

    try:
        wait_for_tasks(si, [vm.ReconfigVM_Task(spec=spec)])
    except vim.fault.VimFault as ex:
        msg = ex.msg
        # Use metadata (KV) for extra logging
        if kv_status_attached:
            # KV  claims we are attached to a different VM'.
            msg += " disk {0} already attached to VM={1}".format(vmdk_path,
                                                                 kv_uuid)
            if kv_uuid == vm.config.uuid:
                msg += "(Current VM)"
        return err(msg)

    setStatusAttached(vmdk_path, vm)
    logging.info("Disk %s successfully attached. controller pci_slot_number=%s, disk_slot=%d",
                 vmdk_path, pci_slot_number, disk_slot)
    return dev_info(disk_slot, pci_slot_number)


def err(string):
    return {u'Error': string}


def disk_detach(vmdk_path, vm):
    """detach disk (by full path) from a vm amd return None or err(msg)"""

    device = findDeviceByPath(vmdk_path, vm)

    if not device:
       # Could happen if the disk attached to a different VM - attach fails
       # and docker will insist to sending "unmount/detach" which also fails.
       msg = "*** Detach failed: disk={0} not found. VM={1}".format(
       vmdk_path, vm.config.uuid)
       logging.warning(msg)
       return err(msg)

    return disk_detach_int(vmdk_path, vm, device)

def disk_detach_int(vmdk_path, vm, device):
    spec = vim.vm.ConfigSpec()
    dev_changes = []

    disk_spec = vim.vm.device.VirtualDeviceSpec()
    disk_spec.operation = vim.vm.device.VirtualDeviceSpec.Operation.remove
    disk_spec.device = device
    dev_changes.append(disk_spec)
    spec.deviceChange = dev_changes

    try:
        wait_for_tasks(si, [vm.ReconfigVM_Task(spec=spec)])
    except vim.Fault.VimFault as ex:
        ex_type, ex_value, ex_traceback = sys.exc_info()
        msg = "Failed to detach %s: %s" % (vmdk_path, ex.msg)
        logging.warning("%s\n%s", msg, "".join(traceback.format_tb(ex_traceback)))
        return err(msg)

    setStatusDetached(vmdk_path)
    logging.info("Disk detached %s", vmdk_path)
    return None


def signal_handler_stop(signalnum, frame):
    logging.warn("Received signal num: ' %d '", signalnum)
    sys.exit(0)

def load_vmci():
   global lib

   logging.info("Loading VMCI server lib.")
   if sys.hexversion >= PYTHON64_VERSION:
       lib = CDLL(os.path.join(LIB_LOC64, "libvmci_srv.so"), use_errno=True)
   else:
       lib = CDLL(os.path.join(LIB_LOC, "libvmci_srv.so"), use_errno=True)

# load VMCI shared lib , listen on vSocket in main loop, handle requests
def handleVmciRequests(port):
    VMCI_ERROR = -1 # VMCI C code uses '-1' to indicate failures
    ECONNABORTED = 103 # Error on non privileged client

    bsize = MAX_JSON_SIZE
    txt = create_string_buffer(bsize)

    cartel = c_int32()
    sock = lib.vmci_init(c_uint(port))
    if sock == VMCI_ERROR:
        errno = get_errno()
        raise OSError("Failed to initialize vSocket listener: %s (errno=%d)" \
                        %  (os.strerror(errno), errno))

    skip_count = MAX_SKIP_COUNT  # retries for vmci_get_one_op failures
    while True:
        c = lib.vmci_get_one_op(sock, byref(cartel), txt, c_int(bsize))
        logging.debug("lib.vmci_get_one_op returns %d, buffer '%s'",
                      c, txt.value)

        errno = get_errno()
        if errno == ECONNABORTED:
            logging.warn("Client with non privileged port attempted a request")
            continue
        if c == VMCI_ERROR:
            # We can self-correct by reoping sockets internally. Give it a chance.
            logging.warning("vmci_get_one_op failed ret=%d: %s (errno=%d) Retrying...",
                            c, os.strerror(errno), errno)
            skip_count = skip_count - 1
            if skip_count <= 0:
                raise Exception(
                    "vmci_get_one_op: too many errors. Giving up.")
            continue
        else:
            skip_count = MAX_SKIP_COUNT  # reset the counter, just in case

        # Get VM name & ID from VSI (we only get cartelID from vmci, need to convert)
        vmm_leader = vsi.get("/userworld/cartel/%s/vmmLeader" %
                            str(cartel.value))
        group_info = vsi.get("/vm/%s/vmmGroupInfo" % vmm_leader)

        vm_name = group_info["displayName"]
        cfg_path = group_info["cfgPath"]
        uuid = group_info["uuid"]
        # pyVmomi expects uuid like this one: 564dac12-b1a0-f735-0df3-bceb00b30340
        # to get it from uuid in VSI vms/<id>/vmmGroup, we use the following format:
        UUID_FORMAT = "{0}{1}{2}{3}-{4}{5}-{6}{7}-{8}{9}-{10}{11}{12}{13}{14}{15}"
        vm_uuid = UUID_FORMAT.format(*uuid.replace("-",  " ").split())

        try:
            req = json.loads(txt.value.decode('utf-8'))
        except ValueError as e:
            ret = {u'Error': "Failed to parse json '%s'." % txt.value}
        else:
            details = req["details"]
            opts = details["Opts"] if "Opts" in details else {}
            threading.currentThread().setName(vm_name)
            ret = executeRequest(vm_uuid=vm_uuid,
                                 vm_name=vm_name,
                                 config_path=cfg_path,
                                 cmd=req["cmd"],
                                 full_vol_name=details["Name"],
                                 opts=opts)
            logging.info("executeRequest '%s' completed with ret=%s",
                         req["cmd"], ret)

        ret_string = json.dumps(ret)
        response = lib.vmci_reply(c, c_char_p(ret_string.encode()))
        errno = get_errno()
        logging.debug("lib.vmci_reply: VMCI replied with errcode %s", response)
        if response == VMCI_ERROR:
            logging.warning("vmci_reply returned error %s (errno=%d)",
                            os.strerror(errno), errno)

    lib.close(sock)  # close listening socket when the loop is over

def usage():
    print("Usage: %s -p <vSocket Port to listen on>" % sys.argv[0])

def main():
    log_config.configure()
    logging.info("=== Starting vmdkops service ====")
    signal.signal(signal.SIGINT, signal_handler_stop)
    signal.signal(signal.SIGTERM, signal_handler_stop)
    try:
        port = 1019
        opts, args = getopt.getopt(sys.argv[1:], 'hp:')
    except getopt.error as msg:
        if msg:
           logging.exception(msg)
        usage()
        return 1
    for a, v in opts:
        if a == '-p':
            port = int(v)
        if a == '-h':
            usage()
            return 0

    try:
        # Load and use DLL with vsocket shim to listen for docker requests
        load_vmci()

        kv.init()
        connectLocal()
        handleVmciRequests(port)
    except Exception as e:
        logging.exception(e)


def getTaskList(prop_collector, tasks):
    # Create filter
    obj_specs = [vmodl.query.PropertyCollector.ObjectSpec(obj=task)
                 for task in tasks]
    property_spec = vmodl.query.PropertyCollector.PropertySpec(type=vim.Task,
                                                               pathSet=[],
                                                               all=True)
    filter_spec = vmodl.query.PropertyCollector.FilterSpec()
    filter_spec.objectSet = obj_specs
    filter_spec.propSet = [property_spec]
    return prop_collector.CreateFilter(filter_spec, True)

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
    task_list = [str(task) for task in tasks]
    property_collector = service_instance.content.propertyCollector
    try:
        pcfilter = getTaskList(property_collector, tasks)
    except vim.fault.NotAuthenticated:
        # Reconnect and retry
        logging.warning("Reconnecting and retry")
        connectLocal()
        property_collector = si.content.propertyCollector
        pcfilter = getTaskList(property_collector, tasks)

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


class ValidationError(Exception):
    """ An exception for option validation errors """

    def __init__(self, msg):
        self.msg = msg

    def __str__(self):
        return repr(self.msg)

# start the server
if __name__ == "__main__":
    main()
