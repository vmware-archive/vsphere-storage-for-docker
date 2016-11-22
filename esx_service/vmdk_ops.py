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
PY2_LOC = os.path.join(PY_LOC, "2")

# We won't accept names longer than that
MAX_VOL_NAME_LEN = 100
MAX_DS_NAME_LEN  = 100

# vmdkops python utils are in PY_LOC, so insert to path ahead of other stuff
sys.path.insert(0, PY_LOC)

# if we are on Python 2, add py2-only stuff as a fallback
if sys.version_info.major == 2:
    sys.path.append(PY2_LOC)

import threadutils
import log_config
import volume_kv as kv
import vmdk_utils
import vsan_policy
import vsan_info
import auth
import sqlite3
import convert
import error_code

# Python version 3.5.1
PYTHON64_VERSION = 50659824

# External tools used by the plugin.
OBJ_TOOL_CMD = "/usr/lib/vmware/osfs/bin/objtool open -u "
OSFS_MKDIR_CMD = "/usr/lib/vmware/osfs/bin/osfs-mkdir -n "
MKDIR_CMD = "/bin/mkdir"
VMDK_CREATE_CMD = "/sbin/vmkfstools"
VMDK_DELETE_CMD = "/sbin/vmkfstools -U "

# For retries on vmkfstools
VMDK_RETRY_COUNT = 5
VMDK_RETRY_SLEEP = 1

# Defaults
DOCK_VOLS_DIR = "dockvols"  # place in the same (with Docker VM) datastore
MAX_JSON_SIZE = 1024 * 4  # max buf size for query json strings. Queries are limited in size
MAX_SKIP_COUNT = 16       # max retries on VMCI Get Ops failures

# Error codes
VMCI_ERROR = -1 # VMCI C code uses '-1' to indicate failures
ECONNABORTED = 103 # Error on non privileged client

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
_service_instance = None

# VMCI library used to communicate with clients
lib = None

# For managing resource locks.
lockManager = threadutils.LockManager()

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
                         universal_newlines=True,
                         shell=True)
    o, e = p.communicate()
    s = p.returncode

    if s != 0:
        return (s, e)

    return (s, o)


# returns error, or None for OK
# opts is  dictionary of {option: value}.
# for now we care about size and (maybe) policy
def createVMDK(vmdk_path, vm_name, vol_name, opts={}, vm_uuid=None, tenant_uuid=None, datastore=None):
    logging.info("*** createVMDK: %s opts = %s", vmdk_path, opts)
    if os.path.isfile(vmdk_path):
        return err("File %s already exists" % vmdk_path)

    try:
        validate_opts(opts, vmdk_path)
    except ValidationError as e:
        return err(e.msg)

    if kv.CLONE_FROM in opts:
        return cloneVMDK(vm_name, vmdk_path, opts,
                         vm_uuid, datastore)

    cmd = make_create_cmd(opts, vmdk_path)
    rc, out = RunCommand(cmd)
    if rc != 0:
        return err("Failed to create %s. %s" % (vmdk_path, out))

    if not create_kv_store(vm_name, vmdk_path, opts):
        msg = "Failed to create metadata kv store for {0}".format(vmdk_path)
        logging.warning(msg)
        error_info = err(msg)
        remove_err = removeVMDK(vmdk_path=vmdk_path, 
                                vol_name=vol_name, 
                                vm_name=vm_name,
                                tenant_uuid=tenant_uuid,
                                datastore=datastore)
        if remove_err:
            error_info = error_info + remove_err
        return error_info
   
    backing, needs_cleanup = get_backing_device(vmdk_path)
    cleanup_backing_device(backing, needs_cleanup)

    # create succeed, insert the volume information into "volumes" table
    if tenant_uuid:
        vol_size_in_MB = convert.convert_to_MB(auth.get_vol_size(opts))
        auth.add_volume_to_volumes_table(tenant_uuid, datastore, vol_name, vol_size_in_MB)
    else:
        logging.debug(error_code.VM_NOT_BELONG_TO_TENANT.format(vm_name))

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


def cloneVMDK(vm_name, vmdk_path, opts={}, vm_uuid=None, vm_datastore=None):
    logging.info("*** cloneVMDK: %s opts = %s", vmdk_path, opts)

    # Get source volume path for cloning
    error_info, tenant_uuid, tenant_name = auth.get_tenant(vm_uuid)
    if error_info:
        return err(error_info)

    try:
        src_volume, src_datastore = parse_vol_name(opts[kv.CLONE_FROM])
    except ValidationError as ex:
        return err(str(ex))
    if not src_datastore:
        src_datastore = vm_datastore
    elif not vmdk_utils.validate_datastore(src_datastore):
        return err("Invalid datastore '%s'.\n" \
                    "Known datastores: %s.\n" \
                    "Default datastore: %s" \
                    % (src_datastore, ", ".join(get_datastore_names_list), vm_datastore))

    error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid,
                                                          src_datastore, auth.CMD_ATTACH, {})
    if error_info:
        errmsg = "Failed to authorize VM: {0}, datastore: {1}".format(error_info, src_datastore)
        logging.warning("*** cloneVMDK: %s", errmsg)
        return err(errmsg)

    src_path, errMsg = get_vol_path(src_datastore, tenant_name)
    if src_path is None:
        return err("Failed to initialize source volume path {0}: {1}".format(src_path, errMsg))

    src_vmdk_path = vmdk_utils.get_vmdk_path(src_path, src_volume)
    if not os.path.isfile(src_vmdk_path):
        return err("Could not find volume for cloning %s" % opts[kv.CLONE_FROM])

    with lockManager.get_lock(src_volume):
        # Verify if the source volume is in use.
        attached, uuid, attach_as = getStatusAttached(src_vmdk_path)
        if attached:
            if handle_stale_attach(vmdk_path, uuid):
                return err("Source volume cannot be in use when cloning")

        # Reauthorize with size info of the volume being cloned
        src_vol_info = kv.get_vol_info(src_vmdk_path)
        datastore = vmdk_utils.get_datastore_from_vmdk_path(vmdk_path)
        opts["size"] = src_vol_info["size"]
        error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid,
                                                              datastore, auth.CMD_CREATE, opts)
        if error_info:
            return err(error_info)

        # Handle the allocation format
        if not kv.DISK_ALLOCATION_FORMAT in opts:
            disk_format = kv.DEFAULT_ALLOCATION_FORMAT
        else:
            disk_format = str(opts[kv.DISK_ALLOCATION_FORMAT])

        # VirtualDiskSpec
        vdisk_spec = vim.VirtualDiskManager.VirtualDiskSpec()
        vdisk_spec.adapterType = 'busLogic'
        vdisk_spec.diskType = disk_format

        # Form datastore path from vmdk_path
        dest_vol = vmdk_utils.get_datastore_path(vmdk_path)
        source_vol = vmdk_utils.get_datastore_path(src_vmdk_path)

        si = get_si()
        task = si.content.virtualDiskManager.CopyVirtualDisk(
            sourceName=source_vol, destName=dest_vol, destSpec=vdisk_spec)
        try:
            wait_for_tasks(si, [task])
        except vim.fault.VimFault as ex:
            return err("Failed to clone volume: {0}".format(ex.msg))

    # Update volume meta
    vol_name = vmdk_utils.strip_vmdk_extension(src_vmdk_path.split("/")[-1])
    vol_meta = kv.getAll(vmdk_path)
    vol_meta[kv.CREATED_BY] = vm_name
    vol_meta[kv.CREATED] = time.asctime(time.gmtime())
    vol_meta[kv.VOL_OPTS][kv.CLONE_FROM] = src_volume
    vol_meta[kv.VOL_OPTS][kv.DISK_ALLOCATION_FORMAT] = disk_format
    if kv.ACCESS in opts:
        vol_meta[kv.VOL_OPTS][kv.ACCESS] = opts[kv.ACCESS]
    if kv.ATTACH_AS in opts:
        vol_meta[kv.VOL_OPTS][kv.ATTACH_AS] = opts[kv.ATTACH_AS]

    if not kv.setAll(vmdk_path, vol_meta):
        msg = "Failed to create metadata kv store for {0}".format(vmdk_path)
        logging.warning(msg)
        removeVMDK(vmdk_path)
        return err(msg)

    backing, needs_cleanup = get_backing_device(vmdk_path)
    cleanup_backing_device(backing, needs_cleanup)


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
    valid_opts = [kv.SIZE, kv.VSAN_POLICY_NAME, kv.DISK_ALLOCATION_FORMAT,
                  kv.ATTACH_AS, kv.ACCESS, kv.FILESYSTEM_TYPE, kv.CLONE_FROM]
    defaults = [kv.DEFAULT_DISK_SIZE, kv.DEFAULT_VSAN_POLICY,\
                kv.DEFAULT_ALLOCATION_FORMAT, kv.DEFAULT_ATTACH_AS,\
                kv.DEFAULT_ACCESS, kv.DEFAULT_FILESYSTEM_TYPE, kv.DEFAULT_CLONE_FROM]
    invalid = frozenset(opts.keys()).difference(valid_opts)
    if len(invalid) != 0:
        msg = 'Invalid options: {0} \n'.format(list(invalid)) \
               + 'Valid options and defaults: ' \
               + '{0}'.format(zip(list(valid_opts), defaults))
        raise ValidationError(msg)

    # For validation of clone (in)compatible options
    clone = True if kv.CLONE_FROM in opts else False

    if kv.SIZE in opts:
        validate_size(opts[kv.SIZE], clone)
    if kv.VSAN_POLICY_NAME in opts:
        validate_vsan_policy_name(opts[kv.VSAN_POLICY_NAME], vmdk_path)
    if kv.DISK_ALLOCATION_FORMAT in opts:
        validate_disk_allocation_format(opts[kv.DISK_ALLOCATION_FORMAT])
    if kv.ATTACH_AS in opts:
        validate_attach_as(opts[kv.ATTACH_AS])
    if kv.ACCESS in opts:
        validate_access(opts[kv.ACCESS])
    if kv.FILESYSTEM_TYPE in opts:
        validate_fstype(opts[kv.FILESYSTEM_TYPE], clone)


def validate_size(size, clone=False):
    """
    Ensure size is given in a human readable format <int><unit> where int is an
    integer and unit is either 'mb', 'gb', or 'tb'. e.g. 22mb
    """
    if clone:
        raise ValidationError("Cannot define the size for a clone")

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
                            " Valid options are: {1}.".format(
                            alloc_format, kv.VALID_ALLOCATION_FORMATS))

def validate_attach_as(attach_type):
    """
    Ensure that we recognize the attach type
    """
    if not attach_type in kv.ATTACH_AS_TYPES :
        raise ValidationError("Attach type '{0}' is not supported."
                              " Valid options are: {1}".format(attach_type, kv.ATTACH_AS_TYPES))

def validate_access(access_type):
    """
    Ensure that we recognize the access type
    """
    if not access_type in kv.ACCESS_TYPES :
       raise ValidationError("Access type '{0}' is not supported."
                             " Valid options are: {1}".format(access_type,
                                                              kv.ACCESS_TYPES))

def validate_fstype(fstype, clone=False):
    """
    Ensure that we don't accept fstype for a clone
    """
    if clone:
        raise ValidationError("Cannot define the filesystem type for a clone")

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
        logging.error("Failed to remove backing device %s, err %s",
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

# Return volume ingo
def vol_info(vol_meta, vol_size_info, datastore):
    vinfo = {CREATED_BY_VM : vol_meta[kv.CREATED_BY],
             kv.CREATED : vol_meta[kv.CREATED],
             kv.STATUS : vol_meta[kv.STATUS]}

    vinfo[CAPACITY] = {}
    vinfo[CAPACITY][SIZE] = vol_size_info[SIZE]
    vinfo[CAPACITY][ALLOCATED] = vol_size_info[ALLOCATED]
    vinfo[LOCATION] = datastore

    if kv.ATTACHED_VM_UUID in vol_meta:
       vm = findVmByUuid(vol_meta[kv.ATTACHED_VM_UUID])
       if vm:
          vinfo[ATTACHED_TO_VM] = vm.config.name
    if kv.VOL_OPTS in vol_meta:
       if kv.FILESYSTEM_TYPE in vol_meta[kv.VOL_OPTS]:
          vinfo[kv.FILESYSTEM_TYPE] = vol_meta[kv.VOL_OPTS][kv.FILESYSTEM_TYPE]
       if kv.VSAN_POLICY_NAME in vol_meta[kv.VOL_OPTS]:
          vinfo[kv.VSAN_POLICY_NAME] = vol_meta[kv.VOL_OPTS][kv.VSAN_POLICY_NAME]
       if kv.DISK_ALLOCATION_FORMAT in vol_meta[kv.VOL_OPTS]:
          vinfo[kv.DISK_ALLOCATION_FORMAT] = vol_meta[kv.VOL_OPTS][kv.DISK_ALLOCATION_FORMAT]
       else:
          vinfo[kv.DISK_ALLOCATION_FORMAT] = kv.DEFAULT_ALLOCATION_FORMAT
       if kv.ATTACH_AS in vol_meta[kv.VOL_OPTS]:
          vinfo[kv.ATTACH_AS] = vol_meta[kv.VOL_OPTS][kv.ATTACH_AS]
       else:
          vinfo[kv.ATTACH_AS] = kv.DEFAULT_ATTACH_AS
       if kv.ACCESS in vol_meta[kv.VOL_OPTS]:
          vinfo[kv.ACCESS] = vol_meta[kv.VOL_OPTS][kv.ACCESS]
       else:
          vinfo[kv.ACCESS] = kv.DEFAULT_ACCESS
       if kv.CLONE_FROM in vol_meta[kv.VOL_OPTS]:
          vinfo[kv.CLONE_FROM] = vol_meta[kv.VOL_OPTS][kv.CLONE_FROM]
       else:
          vinfo[kv.CLONE_FROM] = kv.DEFAULT_CLONE_FROM

    return vinfo


# Return error, or None for OK
def removeVMDK(vmdk_path, vol_name=None, vm_name=None, tenant_uuid=None, datastore=None):
    logging.info("*** removeVMDK: %s", vmdk_path)

    # Check the current volume status
    kv_status_attached, kv_uuid, attach_mode = getStatusAttached(vmdk_path)
    if kv_status_attached:
        if handle_stale_attach(vmdk_path, kv_uuid):
            logging.info("*** removeVMDK: %s is in use, VM uuid = %s", vmdk_path, kv_uuid)
            return err("Failed to remove volume {0}, in use by VM uuid = {1}.".format(
                vmdk_path, kv_uuid))

    cmd = "{0} {1}".format(VMDK_DELETE_CMD, vmdk_path)
    # Workaround timing/locking issues.
    retry_count = 0
    while True:
        rc, out = RunCommand(cmd)
        if rc != 0 and "lock" in out:
            if retry_count == VMDK_RETRY_COUNT:
                return err("Failed to remove %s. %s" % (vmdk_path, out))
            logging.info("*** removeVMDK: %s, coudn't lock volume for removal. Retrying...",
                         vmdk_path)
            retry_count += 1
            time.sleep(VMDK_RETRY_SLEEP)
            continue
        elif rc != 0:
            return err("Failed to remove %s. %s" % (vmdk_path, out))
        else:
            # remove succeed, remove infomation of this volume from volumes table
            if tenant_uuid:
                error_info = auth.remove_volume_from_volumes_table(tenant_uuid, datastore, vol_name)
                return error_info
            else:
                if not vm_name:
                    logging.debug(error_code.VM_NOT_BELONG_TO_TENANT.format(vm_name))

            return None

def getVMDK(vmdk_path, vol_name, datastore):
    """Checks if the volume exists, and returns error if it does not"""
    # Note: will return more Volume info here, when Docker API actually accepts it
    if not os.path.isfile(vmdk_path):
        return err("Volume {0} not found (file: {1})".format(vol_name, vmdk_path))
    # Return volume info - volume policy, size, allocated capacity, allocation
    # type, creat-by, create time.
    try:
        result = vol_info(kv.getAll(vmdk_path),
                          kv.get_vol_info(vmdk_path),
                          datastore)
    except Exception as ex:
        logging.error("Failed to get disk details for %s (%s)" % (vmdk_path, ex))
        return None

    return result

def listVMDK(vm_datastore, tenant):
    """
    Returns a list of volume names (note: may be an empty list).
    Each volume name is returned as either `volume@datastore`, or just `volume`
    for volumes on vm_datastore
    """
    vmdks = vmdk_utils.get_volumes(tenant)
    # build  fully qualified vol name for each volume found
    return [{u'Name': get_full_vol_name(x['filename'], x['datastore'], vm_datastore),
             u'Attributes': {}} \
            for x in vmdks]


# Return VM managed object, reconnect if needed. Throws if fails twice.
def findVmByUuid(vm_uuid):
    si = get_si()
    vm = si.content.searchIndex.FindByUuid(None, vm_uuid, True, False)
    return vm

def vm_uuid2name(vm_uuid):
    vm = findVmByUuid(vm_uuid)
    if not vm or not vm.config:
        return None
    return vm.config.name

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
def get_vol_path(datastore, tenant_name=None):
    # If the command is NOT running under a tenant, the folder for Docker
    # volumes is created on <datastore>/DOCK_VOLS_DIR
    # If the command is running under a tenant, the folder for Dock volume
    # is created on <datastore>/DOCK_VOLS_DIR/tenant_name
    dock_vol_path = os.path.join("/vmfs/volumes", datastore, DOCK_VOLS_DIR)
    if tenant_name:
        path = os.path.join(dock_vol_path, tenant_name)
    else:
        path = dock_vol_path

    if os.path.isdir(path):
        # If the path exists then return it as is.
        logging.debug("Found %s, returning", path)
        return path, None

    if not os.path.isdir(dock_vol_path):
        # The osfs tools are usable for DOCK_VOLS_DIR on all datastores
        cmd = "{0} {1}".format(OSFS_MKDIR_CMD, dock_vol_path)
        rc, out = RunCommand(cmd)
        if rc != 0:
            errMsg = "{0} creation failed - {1} on {2}".format(DOCK_VOLS_DIR, os.strerror(rc), datastore)
            logging.warning(errMsg)
            return None, err(errMsg)
    if tenant_name and not os.path.isdir(path):
        # The mkdir command is used to create "tenant_name" folder inside DOCK_VOLS_DIR on "datastore"
        cmd = "{0} {1}".format(MKDIR_CMD, path)
        rc, out = RunCommand(cmd)
        if rc != 0:
            errMsg = "Failed to initialize volume path {0} - {1}".format(path, out)
            logging.warning(errMsg)
            return None, err(errMsg)

    logging.info("Created %s", path)
    return path, None

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
    logging.debug("config_path=%s", config_path)
    vm_datastore = get_datastore_name(config_path)
    error_info, tenant_uuid, tenant_name = auth.get_tenant(vm_uuid)
    if error_info:
        return err(error_info)

    if cmd == "list":
        return listVMDK(vm_datastore, tenant_name)

    try:
        vol_name, datastore = parse_vol_name(full_vol_name)
    except ValidationError as ex:
        return err(str(ex))
    if not datastore:
        datastore = vm_datastore
    elif not vmdk_utils.validate_datastore(datastore):
        return err("Invalid datastore '%s'.\n" \
                "Known datastores: %s.\n" \
                "Default datastore: %s" \
                % (datastore, ", ".join(get_datastore_names_list), vm_datastore))

    error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid, datastore, cmd, opts)
    if error_info:
        return err(error_info)

    # get /vmfs/volumes/<volid>/dockvols path on ESX:
    path, errMsg = get_vol_path(datastore, tenant_name)
    logging.debug("executeRequest %s %s", tenant_name, path)
    if path is None:
        return errMsg

    vmdk_path = vmdk_utils.get_vmdk_path(path, vol_name)

    if cmd == "get":
        response = getVMDK(vmdk_path, vol_name, datastore)
    elif cmd == "create":              
        response = createVMDK(vmdk_path=vmdk_path, 
                              vm_name=vm_name, 
                              vol_name=vol_name, 
                              opts=opts, 
                              tenant_uuid=tenant_uuid, 
                              datastore=datastore)
    elif cmd == "remove":
        response = removeVMDK(vmdk_path=vmdk_path, 
                              vol_name=vol_name,
                              vm_name=vm_name,
                              tenant_uuid=tenant_uuid,
                              datastore=datastore)
    elif cmd == "attach":
        response = attachVMDK(vmdk_path, vm_uuid)
    elif cmd == "detach":
        response = detachVMDK(vmdk_path, vm_uuid)
    else:
        return err("Unknown command:" + cmd)

    return response

def connectLocalSi(force=False):
    '''
	Initialize a connection to the local SI
	'''
    global _service_instance
    if not _service_instance:
        try:
            logging.info("Connecting to the local Service Instance")
            _service_instance = pyVim.connect.Connect(host='localhost', user='dcui')
        except Exception as e:
            logging.exception("Failed to the local Service Instance as 'dcui', exiting: ")
            sys.exit(1)
    elif force:
        logging.warning("Reconnecting to the local Service Instance")
        _service_instance = pyVim.connect.Connect(host='localhost', user='dcui')

    # set out ID in context to be used in request - so we'll see it in logs
    reqCtx = VmomiSupport.GetRequestContext()
    reqCtx["realUser"] = 'dvolplug'
    atexit.register(pyVim.connect.Disconnect, _service_instance)

def get_si():
    '''
	Return a connection to the local SI
	'''
    with lockManager.get_lock('siLock'):
        try:
            _service_instance.CurrentTime()
        except:
            connectLocalSi(force=True)
    return _service_instance

def get_datastore_names_list():
    """returns names of known datastores"""
    return [i[0] for i in vmdk_utils.get_datastores()]

def get_datastore_url(datastore):
    si = get_si()
    res = [d.info.url for d in si.content.rootFolder.childEntity[0].datastore if d.info.name == datastore]
    return res[0]

def findDeviceByPath(vmdk_path, vm):
    logging.debug("findDeviceByPath: Looking for device {0}".format(vmdk_path))
    for d in vm.config.hardware.device:
        if type(d) != vim.vm.device.VirtualDisk:
            continue

        # Disks of all backing have a backing object with a filename attribute.
        # The filename identifies the virtual disk by name and can be used
        # to match with the given volume name.
        # Filename format is as follows:
        #   "[<datastore name>] <parent-directory>/tenant/<vmdk-descriptor-name>"
        logging.debug("d.backing.fileName %s", d.backing.fileName)
        backing_disk = d.backing.fileName.split(" ")[1]

        # datastore='[datastore name]'
        datastore = d.backing.fileName.split(" ")[0]
        datastore = datastore[1:-1]

        # Construct the parent dir and vmdk name, resolving
        # links if any.
        dvol_dir = os.path.dirname(vmdk_path)
        datastore_url = get_datastore_url(datastore)
        datastore_prefix = os.path.realpath(datastore_url) + '/'
        real_vol_dir = os.path.realpath(dvol_dir).replace(datastore_prefix, "")
        virtual_disk = os.path.join(real_vol_dir, os.path.basename(vmdk_path))
        logging.debug("dvol_dir=%s datastore_prefix=%s real_vol_dir=%s", dvol_dir, datastore_prefix,real_vol_dir)
        logging.debug("backing_disk=%s virtual_disk=%s", backing_disk, virtual_disk)
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
    if set(vol_meta.keys()) & {kv.STATUS, kv.ATTACHED_VM_UUID}:
          logging.debug("Old meta-data for %s was (status=%s VM uuid=%s)",
                        vmdk_path, vol_meta[kv.STATUS],
                        vol_meta[kv.ATTACHED_VM_UUID])
    vol_meta[kv.STATUS] = kv.DETACHED
    vol_meta[kv.ATTACHED_VM_UUID] = None
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
        si = get_si()
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
        logging.info("Added a PVSCSI controller, controller_key=%d pci_slot_number=%s",
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
        si = get_si()
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
    si = get_si()
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


# Edit settings for a volume identified by its full path
def set_vol_opts(name, options):
    # Create a dict of the options, the options are provided as
    # "access=read-only" and we get a dict like {'access': 'read-only'}
    opts_list = "".join(options.replace("=", ":").split())
    opts = dict(i.split(":") for i in opts_list.split(","))

    # create volume path
    try:
       vol_name, datastore = parse_vol_name(name)
    except ValidationError as ex:
       logging.exception(ex)
       return False

    if not datastore:
       msg = "Invalid datastore '{0}'.\n".format(datastore)
       logging.warning(msg)
       return False

    # get /vmfs/volumes/<datastore>/dockvols path on ESX:
    path, errMsg = get_vol_path(datastore)

    if path is None:
       msg = "Failed to get datastore path {0}".format(path)
       logging.warning(msg)
       return False

    vmdk_path = vmdk_utils.get_vmdk_path(path, vol_name)

    if not os.path.isfile(vmdk_path):
       msg = 'Volume {0} not found.'.format(vol_name)
       logging.warning(msg)
       return False

    # For now only allow resetting the access and attach-as options.
    valid_opts = {
        kv.ACCESS : kv.ACCESS_TYPES,
        kv.ATTACH_AS : kv.ATTACH_AS_TYPES
    }

    invalid = frozenset(opts.keys()).difference(valid_opts.keys())
    if len(invalid) != 0:
        msg = 'Invalid options: {0} \n'.format(list(invalid)) \
               + 'Options that can be edited: ' \
               + '{0}'.format(list(valid_opts))
        raise ValidationError(msg)

    has_invalid_opt_value = False
    for key in opts.keys():
        if key in valid_opts:
            if not opts[key] in valid_opts[key]:
                msg = 'Invalid option value {0}.\n'.format(opts[key]) +\
                    'Supported values are {0}.\n'.format(valid_opts[key])
                logging.warning(msg)
                has_invalid_opt_value = True

    if has_invalid_opt_value:
        return False

    vol_meta = kv.getAll(vmdk_path)
    if vol_meta:
       if not vol_meta[kv.VOL_OPTS]:
           vol_meta[kv.VOL_OPTS] = {}
       for key in opts.keys():
           vol_meta[kv.VOL_OPTS][key] = opts[key]
       return kv.setAll(vmdk_path, vol_meta)

    return False

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


def send_vmci_reply(client_socket, reply_string):
    reply = json.dumps(reply_string)
    response = lib.vmci_reply(client_socket, c_char_p(reply.encode()))
    errno = get_errno()
    logging.debug("lib.vmci_reply: VMCI replied with errcode %s", response)
    if response == VMCI_ERROR:
        logging.warning("vmci_reply returned error %s (errno=%d)",
                        os.strerror(errno), errno)

def execRequestThread(client_socket, cartel, request):
    '''
    Execute requests in a thread context with a per volume locking.
    '''
    # Before we start, block to allow main thread or other running threads to advance.
    # https://docs.python.org/2/faq/library.html#none-of-my-threads-seem-to-run-why
    time.sleep(0.001)
    try:
        # Get VM name & ID from VSI (we only get cartelID from vmci, need to convert)
        vmm_leader = vsi.get("/userworld/cartel/%s/vmmLeader" % str(cartel))
        group_info = vsi.get("/vm/%s/vmmGroupInfo" % vmm_leader)
        vm_name = group_info["displayName"]
        cfg_path = group_info["cfgPath"]
        uuid = group_info["uuid"]
        # pyVmomi expects uuid like this one: 564dac12-b1a0-f735-0df3-bceb00b30340
        # to get it from uuid in VSI vms/<id>/vmmGroup, we use the following format:
        UUID_FORMAT = "{0}{1}{2}{3}-{4}{5}-{6}{7}-{8}{9}-{10}{11}{12}{13}{14}{15}"
        vm_uuid = UUID_FORMAT.format(*uuid.replace("-",  " ").split())

        try:
            req = json.loads(request.decode('utf-8'))
        except ValueError as e:
            reply_string = {u'Error': "Failed to parse json '%s'." % request}
            send_vmci_reply(client_socket, reply_string)
        else:
            details = req["details"]
            opts = details["Opts"] if "Opts" in details else {}

            # Lock name defaults to volume name, or socket# if request has no volume defined
            lockname = details["Name"] if len(details["Name"]) > 0 else "socket{0}".format(client_socket)
            # Set thread name to vm_name-lockname
            threading.currentThread().setName("{0}-{1}".format(vm_name, lockname))

            # Get a resource lock
            rsrcLock = lockManager.get_lock(lockname)

            logging.debug("Trying to aquire lock: %s", lockname)
            with rsrcLock:
                logging.debug("Aquired lock: %s", lockname)

                reply_string = executeRequest(vm_uuid=vm_uuid,
                                    vm_name=vm_name,
                                    config_path=cfg_path,
                                    cmd=req["cmd"],
                                    full_vol_name=details["Name"],
                                    opts=opts)

                logging.info("executeRequest '%s' completed with ret=%s", req["cmd"], reply_string)
                send_vmci_reply(client_socket, reply_string)
            logging.debug("Released lock: %s", lockname)

    except Exception as ex_thr:
        logging.exception("Unhandled Exception:")
        reply_string = err("Server returned an error: {0}".format(repr(ex_thr)))
        send_vmci_reply(client_socket, reply_string)

# load VMCI shared lib , listen on vSocket in main loop, handle requests
def handleVmciRequests(port):
    skip_count = MAX_SKIP_COUNT  # retries for vmci_get_one_op failures
    bsize = MAX_JSON_SIZE
    txt = create_string_buffer(bsize)
    cartel = c_int32()
    sock = lib.vmci_init(c_uint(port))

    if sock == VMCI_ERROR:
        errno = get_errno()
        raise OSError("Failed to initialize vSocket listener: %s (errno=%d)" \
                        %  (os.strerror(errno), errno))

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

        client_socket = c # Bind to avoid race conditions.
        # Fire a thread to execute the request
        threading.Thread(
            target=execRequestThread,
            args=(client_socket, cartel.value, txt.value)).start()

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
        connectLocalSi()
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


def wait_for_tasks(si, tasks):
    """Given the service instance si and tasks, it returns after all the
   tasks are complete
   """
    task_list = [str(task) for task in tasks]
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
