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
import time
from ctypes import *

from vmware import vsi

import pyVim
from pyVim.connect import Connect, Disconnect
from pyVim import vmconfig

from pyVmomi import VmomiSupport, vim, vmodl
from pyVmomi.VmomiSupport import newestVersions

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
import auth_data_const
import auth_api
import error_code
from error_code import ErrorCode
from error_code import error_code_to_message
import vm_listener

# Python version 3.5.1
PYTHON64_VERSION = 50659824

# External tools used by the plugin.
OBJ_TOOL_CMD = "/usr/lib/vmware/osfs/bin/objtool open -u "
OSFS_MKDIR_CMD = "/usr/lib/vmware/osfs/bin/osfs-mkdir -n "

# Defaults
DOCK_VOLS_DIR = "dockvols"  # place in the same (with Docker VM) datastore
MAX_JSON_SIZE = 1024 * 4  # max buf size for query json strings. Queries are limited in size
MAX_SKIP_COUNT = 16       # max retries on VMCI Get Ops failures
VMDK_ADAPTER_TYPE = 'busLogic'  # default adapter type

# Server side understand protocol version. If you are changing client/server protocol we use
# over VMCI, PLEASE DO NOT FORGET TO CHANGE IT FOR CLIENT in file <esx_vmdkcmd.go> !
SERVER_PROTOCOL_VERSION = 2

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

# Run executable on ESX as needed.
# Returns int with return value,  and a string with either stdout (on success) or  stderr (on error)
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
def createVMDK(vmdk_path, vm_name, vol_name,
               opts={}, vm_uuid=None, tenant_uuid=None, datastore_url=None):
    logging.info("*** createVMDK: %s opts = %s vm_name=%s vm_uuid=%s tenant_uuid=%s datastore_url=%s",
                  vmdk_path, opts, vm_name, vm_uuid, tenant_uuid, datastore_url)

    if os.path.isfile(vmdk_path):
        # We are mostly here due to race or Plugin VMCI retry #1076
        logging.warning("File %s already exists", vmdk_path)
        return None

    try:
        validate_opts(opts, vmdk_path)
    except ValidationError as e:
        return err(e.msg)

    if kv.CLONE_FROM in opts:
        return cloneVMDK(vm_name, vmdk_path, opts,
                         vm_uuid, datastore_url)

    if not kv.DISK_ALLOCATION_FORMAT in opts:
        disk_format = kv.DEFAULT_ALLOCATION_FORMAT
        # Update opts with DISK_ALLOCATION_FORMAT for volume metadata
        opts[kv.DISK_ALLOCATION_FORMAT] = kv.DEFAULT_ALLOCATION_FORMAT
    else:
        disk_format = kv.VALID_ALLOCATION_FORMATS[opts[kv.DISK_ALLOCATION_FORMAT]]

    # VirtualDiskSpec
    vdisk_spec = vim.VirtualDiskManager.FileBackedVirtualDiskSpec()
    vdisk_spec.adapterType = VMDK_ADAPTER_TYPE
    vdisk_spec.diskType = disk_format

    if  kv.SIZE in opts:
        vdisk_spec.capacityKb = convert.convert_to_KB(opts[kv.SIZE])
    else:
        vdisk_spec.capacityKb = convert.convert_to_KB(kv.DEFAULT_DISK_SIZE)

    # Form datastore path from vmdk_path
    volume_datastore_path = vmdk_utils.get_datastore_path(vmdk_path)
    logging.debug("volume_datastore_path=%s", volume_datastore_path)

    si = get_si()
    task = si.content.virtualDiskManager.CreateVirtualDisk(
        name=volume_datastore_path, spec=vdisk_spec)
    try:
        wait_for_tasks(si, [task])
    except vim.fault.VimFault as ex:
        return err("Failed to create volume: {0}".format(ex.msg))

    logging.debug("Successfully created %s volume", vmdk_path)

    # Handle vsan policy
    if kv.VSAN_POLICY_NAME in opts:
        # Attempt to set policy to vmdk
        # set_policy_to_vmdk() deleted vmdk if couldn't set policy
        set_err = set_policy_to_vmdk(vmdk_path=vmdk_path,
                                     opts=opts,
                                     vol_name=vol_name)
        if set_err:
            return set_err

    if not create_kv_store(vm_name, vmdk_path, opts):
        msg = "Failed to create metadata kv store for {0}".format(vmdk_path)
        logging.warning(msg)
        error_info = err(msg)
        clean_err = cleanVMDK(vmdk_path=vmdk_path,
                              vol_name=vol_name)

        if clean_err:
            logging.warning("Failed to clean %s file: %s", vmdk_path, clean_err)
            error_info = error_info + clean_err

        return error_info

    # create succeed, insert the volume information into "volumes" table
    if tenant_uuid:
        vol_size_in_MB = convert.convert_to_MB(auth.get_vol_size(opts))
        auth.add_volume_to_volumes_table(tenant_uuid, datastore_url, vol_name, vol_size_in_MB)
    else:
        logging.debug(error_code_to_message[ErrorCode.VM_NOT_BELONG_TO_TENANT].format(vm_name))


def cloneVMDK(vm_name, vmdk_path, opts={}, vm_uuid=None, datastore_url=None):
    logging.info("*** cloneVMDK: %s opts = %s vm_uuid=%s datastore_url=%s", vmdk_path, opts, vm_uuid, datastore_url)

    # Get source volume path for cloning
    error_info, tenant_uuid, tenant_name = auth.get_tenant(vm_uuid)
    if error_info:
        return err(error_info)

    try:
        src_volume, src_datastore = parse_vol_name(opts[kv.CLONE_FROM])
    except ValidationError as ex:
        return err(str(ex))
    if not src_datastore:
        src_datastore_url = datastore_url
        src_datastore = vmdk_utils.get_datastore_name(datastore_url)
    elif not vmdk_utils.validate_datastore(src_datastore):
        return err("Invalid datastore '%s'.\n" \
                    "Known datastores: %s.\n" \
                    "Default datastore_url: %s" \
                    % (src_datastore, ", ".join(get_datastore_names_list()), datastore_url))
    else:
        src_datastore_url = vmdk_utils.get_datastore_url(src_datastore)

    error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid,
                                                          src_datastore_url, auth.CMD_ATTACH, {})
    if error_info:
        errmsg = "Failed to authorize VM: {0}, datastore: {1}".format(error_info, src_datastore)
        logging.warning("*** cloneVMDK: %s", errmsg)
        return err(errmsg)

    src_path, errMsg = get_vol_path(src_datastore, tenant_name)
    if src_path is None:
        return err("Failed to initialize source volume path {0}: {1}".format(src_path, errMsg))

    src_vmdk_path = vmdk_utils.get_vmdk_path(src_path, src_volume)
    logging.debug("cloneVMDK: src path=%s vol=%s vmdk_path=%s", src_path, src_volume, src_vmdk_path)
    if not os.path.isfile(src_vmdk_path):
        return err("Could not find volume for cloning %s" % opts[kv.CLONE_FROM])

    # Form datastore path from vmdk_path
    dest_vol = vmdk_utils.get_datastore_path(vmdk_path)
    source_vol = vmdk_utils.get_datastore_path(src_vmdk_path)
    lockname = "{}.{}.{}".format(src_datastore, tenant_name, src_volume)
    with lockManager.get_lock(lockname):
        # Verify if the source volume is in use.
        attached, uuid, attach_as, attached_vm_name = getStatusAttached(src_vmdk_path)
        if attached:
            if handle_stale_attach(vmdk_path, uuid):
                return err("Source volume {0} is in use by VM {1} and can't be cloned.".format(src_volume,
                    attached_vm_name))

        # Reauthorize with size info of the volume being cloned
        src_vol_info = kv.get_vol_info(src_vmdk_path)
        datastore = vmdk_utils.get_datastore_from_vmdk_path(vmdk_path)
        datastore_url = vmdk_utils.get_datastore_url(datastore)
        opts["size"] = src_vol_info["size"]
        error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid,
                                                              datastore_url, auth.CMD_CREATE, opts)
        if error_info:
            return err(error_info)

        # Handle the allocation format
        if not kv.DISK_ALLOCATION_FORMAT in opts:
            disk_format = kv.DEFAULT_ALLOCATION_FORMAT
            # Update opts with DISK_ALLOCATION_FORMAT for volume metadata
            opts[kv.DISK_ALLOCATION_FORMAT] = kv.DEFAULT_ALLOCATION_FORMAT
        else:
            disk_format = kv.VALID_ALLOCATION_FORMATS[opts[kv.DISK_ALLOCATION_FORMAT]]

        # VirtualDiskSpec
        vdisk_spec = vim.VirtualDiskManager.VirtualDiskSpec()
        vdisk_spec.adapterType = VMDK_ADAPTER_TYPE
        vdisk_spec.diskType = disk_format

        # Clone volume
        si = get_si()
        task = si.content.virtualDiskManager.CopyVirtualDisk(
            sourceName=source_vol, destName=dest_vol, destSpec=vdisk_spec)
        try:
            wait_for_tasks(si, [task])
        except vim.fault.VimFault as ex:
            return err("Failed to clone volume: {0}".format(ex.msg))

    vol_name = vmdk_utils.strip_vmdk_extension(src_vmdk_path.split("/")[-1])

    # Handle vsan policy
    if kv.VSAN_POLICY_NAME in opts:
        # Attempt to set policy to vmdk
        # set_policy_to_vmdk() deleted vmdk if couldn't set policy
        set_err = set_policy_to_vmdk(vmdk_path=vmdk_path,
                                     opts=opts,
                                     vol_name=vol_name)

        if set_err:
            return set_err

    # Update volume meta
    vol_meta = kv.getAll(vmdk_path)
    vol_meta[kv.CREATED_BY] = vm_name
    vol_meta[kv.CREATED] = time.asctime(time.gmtime())
    vol_meta[kv.VOL_OPTS][kv.CLONE_FROM] = src_volume
    vol_meta[kv.VOL_OPTS][kv.DISK_ALLOCATION_FORMAT] = opts[kv.DISK_ALLOCATION_FORMAT]
    if kv.ACCESS in opts:
        vol_meta[kv.VOL_OPTS][kv.ACCESS] = opts[kv.ACCESS]
    if kv.ATTACH_AS in opts:
        vol_meta[kv.VOL_OPTS][kv.ATTACH_AS] = opts[kv.ATTACH_AS]

    if not kv.setAll(vmdk_path, vol_meta):
        msg = "Failed to create metadata kv store for {0}".format(vmdk_path)
        logging.warning(msg)
        removeVMDK(vmdk_path)
        return err(msg)

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
               + '{0}'.format(list(zip(list(valid_opts), defaults)))
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

    if not size.lower().endswith(('mb', 'gb', 'tb'
                                  )) or not size[:-2].isdigit():
        msg = ('Invalid format for size. \n'
               'Valid sizes must be of form X[mMgGtT]b where X is an'
               'integer. Default = 100mb')
        raise ValidationError(msg)


def validate_vsan_policy_name(policy_name, vmdk_path):
    """
    Ensure that the policy file exists
    """
    if not vsan_info.is_on_vsan(vmdk_path):
        raise ValidationError('Cannot use a VSAN policy on a non-VSAN datastore')

    if not vsan_policy.policy_exists(policy_name):
        err_msg = 'Policy {0} does not exist.'.format(policy_name)

        # If valid policies exist, append their names along with error message
        # for available policy names that can be used
        avail_policies = vsan_policy.get_policies()
        if avail_policies:
            avail_msg = ' Available policies are: {0}'.format(list(avail_policies.keys()))
            err_msg = err_msg + avail_msg
        raise ValidationError(err_msg)

def set_policy_to_vmdk(vmdk_path, opts, vol_name=None):
    """
    Set VSAN policy to the vmdk object
    If failed, delete the vmdk file and return the error info to be displayed
    on client
    """
    out = vsan_policy.set_policy_by_name(vmdk_path, opts[kv.VSAN_POLICY_NAME])
    if out:
        # If policy is incompatible/wrong, return the error and delete the vmdk_path
        msg = ("Failed to create volume %s: %s" % (vol_name, out))
        logging.warning(msg)
        error_info = err(msg)
        clean_err = cleanVMDK(vmdk_path=vmdk_path,
                            vol_name=vol_name)

        if clean_err:
            logging.warning("Failed to clean %s file: %s", vmdk_path, clean_err)
            error_info = error_info + clean_err

        return error_info

    return None

def validate_disk_allocation_format(alloc_format):
    """
    Ensure format is valid.
    """
    if not alloc_format in kv.VALID_ALLOCATION_FORMATS :
        raise ValidationError("Disk Allocation Format \'{0}\' is not supported."
                            " Valid options are: {1}.".format(
                            alloc_format, list(kv.VALID_ALLOCATION_FORMATS)))

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
        vm_name = vm_uuid2name(vol_meta[kv.ATTACHED_VM_UUID])
        if vm_name:
            vinfo[ATTACHED_TO_VM] = vm_name
        elif kv.ATTACHED_VM_NAME in vol_meta:
            # If vm name couldn't be retrieved through uuid, use name from KV
            vinfo[ATTACHED_TO_VM] = vol_meta[kv.ATTACHED_VM_NAME]
        else:
            vinfo[ATTACHED_TO_VM] = vol_meta[kv.ATTACHED_VM_UUID]
    if kv.ATTACHED_VM_DEV in vol_meta:
        vinfo[kv.ATTACHED_VM_DEV] = vol_meta[kv.ATTACHED_VM_DEV]

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


def cleanVMDK(vmdk_path, vol_name=None):
    """
    Delete the vmdk file. Retry if the attempt fails
    Invoked as a part of removeVMDK procedure and
    cases requiring deletion of vmdk file only (when meta file
    hasn't been generated)
    eg: Unsuccesful attempt to apply vsan policy and when failed
    to create metadata for vmdk_path
    """
    logging.info("*** cleanVMDK: %s", vmdk_path)

    # Form datastore path from vmdk_path
    volume_datastore_path = vmdk_utils.get_datastore_path(vmdk_path)

    retry_count = 0
    while True:
        si = get_si()
        task = si.content.virtualDiskManager.DeleteVirtualDisk(name=volume_datastore_path)
        try:
            # Wait for delete, exit loop on success
            wait_for_tasks(si, [task])
            break
        except vim.fault.FileNotFound as ex:
            logging.warning("*** removeVMDK: File not found error: %s", ex.msg)
            return None
        except vim.fault.VimFault as ex:
            if retry_count == vmdk_utils.VMDK_RETRY_COUNT or "Error caused by file" not in ex.msg:
                return err("Failed to remove volume: {0}".format(ex.msg))
            else:
                logging.warning("*** removeVMDK: Retrying removal on error: %s", ex.msg)
                vmdk_utils.log_volume_lsof(vol_name)
                retry_count += 1
                time.sleep(vmdk_utils.VMDK_RETRY_SLEEP)

    return None

# Return error, or None for OK
def removeVMDK(vmdk_path, vol_name=None, vm_name=None, tenant_uuid=None, datastore_url=None):
    """
    Checks the status of the vmdk file using its meta file
    If it is not attached, then cleans(deletes) the vmdk file.
    If clean is successful, delete the volume from volume table
    """
    logging.info("*** removeVMDK: %s", vmdk_path)

    # Check the current volume status
    kv_status_attached, kv_uuid, attach_mode, attached_vm_name = getStatusAttached(vmdk_path)
    if kv_status_attached:
        ret = handle_stale_attach(vmdk_path, kv_uuid)
        if ret:
            if vol_name is None:
                vol_name = vmdk_utils.get_volname_from_vmdk_path(vmdk_path)
            logging.info("*** removeVMDK: %s is in use, volume = %s VM = %s VM-uuid = %s (%s)",
                vmdk_path, vol_name, attached_vm_name, kv_uuid, ret)
            return err("Failed to remove volume {0}, in use by VM = {1}.".format(vol_name, attached_vm_name))

    # Cleaning .vmdk file
    clean_err = cleanVMDK(vmdk_path, vol_name)

    if clean_err:
        logging.warning("Failed to clean %s file: %s", vmdk_path, clean_err)
        return clean_err

    # clean succeeded, remove infomation of this volume from volumes table
    if tenant_uuid:
        error_info = auth.remove_volume_from_volumes_table(tenant_uuid, datastore_url, vol_name)
        return error_info
    elif not vm_name:
        logging.debug(error_code_to_message[ErrorCode.VM_NOT_BELONG_TO_TENANT].format(vm_name))

    return None


def getVMDK(vmdk_path, vol_name, datastore):
    """Checks if the volume exists, and returns error if it does not"""
    # Note: will return more Volume info here, when Docker API actually accepts it
    logging.debug("getVMDK: vmdk_path=%s vol_name=%s, datastore=%s", vmdk_path, vol_name, datastore)
    file_exist = os.path.isfile(vmdk_path)
    logging.debug("getVMDK: file_exist=%d", file_exist)
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

def listVMDK(tenant):
    """
    Returns a list of volume names (note: may be an empty list).
    Each volume name is returned as either `volume@datastore`, or just `volume`
    for volumes on vm_datastore
    """
    vmdk_utils.init_datastoreCache(force=True)
    vmdks = vmdk_utils.get_volumes(tenant)
    # build  fully qualified vol name for each volume found
    return [{u'Name': get_full_vol_name(x['filename'], x['datastore']),
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
    # If tenant_name is set to None, the folder for Docker
    # volumes is created on <datastore>/DOCK_VOLS_DIR
    # If tenant_name is set, the folder for Dock volume
    # is created on <datastore>/DOCK_VOLS_DIR/tenant_uuid
    # a symlink <datastore>/DOCK_VOLS_DIR/tenant_name will be created to point to
    # path <datastore>/DOCK_VOLS_DIR/tenant_uuid
    # If the dock volume folder already exists,
    # the path returned contains tenant name not UUID.
    # This is to make logs more readable. OS will resolve this path
    # as a symlink with tenant_name will already be present.

    readable_path = path = dock_vol_path = os.path.join("/vmfs/volumes", datastore, DOCK_VOLS_DIR)

    if tenant_name:
        error_info, tenant = auth_api.get_tenant_from_db(tenant_name)
        if error_info:
            logging.error("get_vol_path: failed to find tenant info for tenant %s", tenant_name)
            path = dock_vol_path
        path = os.path.join(dock_vol_path, tenant.id)
        readable_path = os.path.join(dock_vol_path, tenant_name)

    if os.path.isdir(path):
        # If the readable_path exists then return, else return path with no symlinks
        if os.path.exists(readable_path):
            logging.debug("Found %s, returning", readable_path)
            return readable_path, None
        else:
            logging.warning("Internal: Tenant name symlink not found for path %s", readable_path)
            logging.debug("Found %s, returning", path)
            return path, None

    if not os.path.isdir(dock_vol_path):
        # The osfs tools are usable for DOCK_VOLS_DIR on all datastores.
        cmd = "{} '{}'".format(OSFS_MKDIR_CMD, dock_vol_path)
        logging.info("Creating %s, running '%s'", dock_vol_path, cmd)
        rc, out = RunCommand(cmd)
        if rc != 0:
            errMsg = "{0} creation failed - {1} on datastore {2}".format(DOCK_VOLS_DIR, os.strerror(rc), datastore)
            logging.warning(errMsg)
            return None, err(errMsg)

    if tenant_name and not os.path.isdir(path):
        # The mkdir command is used to create "tenant_name" folder inside DOCK_VOLS_DIR on "datastore"
        logging.info("Creating directory %s", path)
        try:
            os.mkdir(path)
        except Exception as ex:
            errMsg = "Failed to initialize volume path {} - {}".format(path, ex)
            logging.warning(errMsg)
            return None, err(errMsg)

        # create the symbol link /vmfs/volumes/datastore_name/dockvol/tenant_name
        symlink_path = os.path.join(dock_vol_path, tenant_name)
        if not os.path.isdir(symlink_path):
            os.symlink(path, symlink_path)
            logging.info("Symlink %s is created to point to path %s", symlink_path, path)

    logging.info("Created %s", path)
    return readable_path, None

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


def get_full_vol_name(vmdk_name, datastore):
    """
    Forms full volume name from vmdk file name an datastore as volume@datastore
    """
    vol_name = vmdk_utils.strip_vmdk_extension(vmdk_name)
    return "{0}@{1}".format(vol_name, datastore)

def datastore_path_exist(datastore_name):
    """ Check whether path /vmfs/volumes/datastore_name" exist or not """
    ds_path = os.path.join("/vmfs/volumes/", datastore_name)
    return os.path.exists(ds_path)

def get_datastore_name(datastore_url):
    """ Get datastore_name with given datastore_url """
    logging.debug("get_datastore_name: datastore_url=%s", datastore_url)
    datastore_name = vmdk_utils.get_datastore_name(datastore_url)
    if datastore_name is None or not datastore_path_exist(datastore_name):
        # path /vmfs/volumes/datastore_name does not exist
        # the possible reason is datastore_name which got from
        # datastore cache is invalid(old name) need to refresh
	    # cache, and try again, may still return None
        logging.debug("get_datastore_name: datastore_name=%s path to /vmfs/volumes/datastore_name does not exist",
                      datastore_name)
        vmdk_utils.init_datastoreCache(force=True)
        datastore_name = vmdk_utils.get_datastore_name(datastore_url)
        logging.debug("get_datastore_name: After refresh get datastore_name=%s", datastore_name)

    return datastore_name

def authorize_check(vm_uuid, datastore_url, cmd, opts, use_default_ds, datastore, vm_datastore):
    """
        Check command from vm can be executed on the datastore or not
        Return None on success or error_info if the command cannot be executed
    """
    if use_default_ds:
        # first check whether it has privilege to default_datastore
        error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid, datastore_url, cmd, opts)
        if error_info:
            return error_info
    else:
        # user passsed in volume with format vol@datastore
        # check the privilege to that datastore
        error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid, datastore_url, cmd, opts)
        # no privilege exist for the given datastore
        # if the given datastore is the same as vm_datastore
        # then we can check privilege against "_VM_DS"
        if error_info == error_code_to_message[ErrorCode.PRIVILEGE_NO_PRIVILEGE] and datastore == vm_datastore:
            error_info, tenant_uuid, tenant_name = auth.authorize(vm_uuid, auth_data_const.VM_DS_URL, cmd, opts)
        if error_info:
            return error_info

    return None


# gets the requests, calculates path for volumes, and calls the relevant handler
def executeRequest(vm_uuid, vm_name, config_path, cmd, full_vol_name, opts):
    """
    Executes a <cmd> request issused from a VM.
    The request is about volume <full_volume_name> in format volume@datastore.
    If @datastore is omitted, "default_datastore" will be used if "default_datastore"
    is specified for the tenant which VM belongs to;
    the one where the VM resides is used is "default_datastore" is not specified.
    For VM, the function gets vm_uuid, vm_name and config_path
    <opts> is a json options string blindly passed to a specific operation

    Returns None (if all OK) or error string
    """
    logging.debug("config_path=%s", config_path)
    vm_datastore_url = vmdk_utils.get_datastore_url_from_config_path(config_path)
    vm_datastore = get_datastore_name(vm_datastore_url)
    logging.debug("executeRequest: vm_datastore = %s, vm_datastore_url = %s",
                  vm_datastore, vm_datastore_url)

    error_info, tenant_uuid, tenant_name = auth.get_tenant(vm_uuid)
    if error_info:
        # For docker volume ls, docker prints a list of cached volume names in case
        # of error(in this case, orphan VM). See Issue #990
        # Explicity providing empty list of volumes to avoid misleading output.
        if (cmd == "list") and (not tenant_uuid):
            return []
        else:
            return err(error_info)

    # default_datastore must be set for tenant
    error_info, default_datastore_url = auth_api.get_default_datastore_url(tenant_name)
    if error_info:
        return err(error_info.msg)
    elif not default_datastore_url:
        err_msg = error_code_to_message[ErrorCode.DS_DEFAULT_NOT_SET].format(tenant_name)
        logging.warning(err_msg)
        return err(err_msg)

    # default_datastore could be a real datastore name or a hard coded  one "_VM_DS"
    default_datastore = get_datastore_name(default_datastore_url)

    logging.debug("executeRequest: vm uuid=%s name=%s, tenant_name=%s, default_datastore=%s",
                  vm_uuid, vm_name, tenant_name, default_datastore)

    if cmd == "list":
        threadutils.set_thread_name("{0}-nolock-{1}".format(vm_name, cmd))
        # if default_datastore is not set, should return error
        return listVMDK(tenant_name)

    try:
        vol_name, datastore = parse_vol_name(full_vol_name)
    except ValidationError as ex:
        return err(str(ex))

    if datastore and not vmdk_utils.validate_datastore(datastore):
        return err("Invalid datastore '%s'.\n" \
                   "Known datastores: %s.\n" \
                   "Default datastore: %s" \
                   % (datastore, ", ".join(get_datastore_names_list()), default_datastore))

    if not datastore:
        datastore_url = default_datastore_url
        datastore = default_datastore
        use_default_ds = True
    else:
        datastore_url = vmdk_utils.get_datastore_url(datastore)
        use_default_ds = False

    logging.debug("executeRequest: vm_uuid=%s, vm_name=%s, tenant_name=%s, tenant_uuid=%s, "
                  "default_datastore_url=%s datastore_url=%s",
                  vm_uuid, vm_name, tenant_uuid, tenant_name, default_datastore_url, datastore_url)

    error_info = authorize_check(vm_uuid, datastore_url, cmd, opts, use_default_ds, datastore, vm_datastore)
    if error_info:
        return err(error_info)

    # get_vol_path() need to pass in a real datastore name
    if datastore == auth_data_const.VM_DS:
        datastore = vm_datastore
        # set datastore_url to a real datastore_url
        # createVMDK() and removeVMDK() need to pass in
        # a real datastore_url instead of url of _VM_DS
        datastore_url = vm_datastore_url

    path, errMsg = get_vol_path(datastore, tenant_name)
    logging.debug("executeRequest for tenant %s with path %s", tenant_name, path)
    if path is None:
        return errMsg

    vmdk_path = vmdk_utils.get_vmdk_path(path, vol_name)

    # Set up locking for volume operations.
    # Lock name defaults to combination of DS,tenant name and vol name
    lockname = "{}.{}.{}".format(vm_datastore, tenant_name, vol_name)
    # Set thread name to vm_name-lockname
    threadutils.set_thread_name("{0}-{1}".format(vm_name, lockname))

    # Get a lock for the volume
    logging.debug("Trying to acquire lock: %s", lockname)
    with lockManager.get_lock(lockname):
        logging.debug("Acquired lock: %s", lockname)

        if cmd == "get":
            response = getVMDK(vmdk_path, vol_name, datastore)
        elif cmd == "create":
            response = createVMDK(vmdk_path=vmdk_path,
                                  vm_name=vm_name,
                                  vm_uuid=vm_uuid,
                                  vol_name=vol_name,
                                  opts=opts,
                                  tenant_uuid=tenant_uuid,
                                  datastore_url=datastore_url)
        elif cmd == "remove":
            response = removeVMDK(vmdk_path=vmdk_path,
                                  vol_name=vol_name,
                                  vm_name=vm_name,
                                  tenant_uuid=tenant_uuid,
                                  datastore_url=datastore_url)

        # For attach/detach reconfigure tasks, hold a per vm lock.
        elif cmd == "attach":
            with lockManager.get_lock(vm_uuid):
                response = attachVMDK(vmdk_path, vm_uuid)
        elif cmd == "detach":
            with lockManager.get_lock(vm_uuid):
                response = detachVMDK(vmdk_path, vm_uuid)
        else:
            return err("Unknown command:" + cmd)

    logging.debug("Released lock: %s", lockname)
    return response

def connectLocalSi():
    '''
	Initialize a connection to the local SI
	'''
    global _service_instance
    if not _service_instance:
        try:
            logging.info("Connecting to the local Service Instance as 'dcui' ")

            # Connect to local server as user "dcui" since this is the Admin that does not lose its
            # Admin permissions even when the host is in lockdown mode. User "dcui" does not have a
            # password - it is used by the ESXi local application DCUI (Direct Console User Interface)
            # Version must be set to access newer features, such as VSAN.
            _service_instance = pyVim.connect.Connect(
                host='localhost',
                user='dcui',
                version=newestVersions.Get('vim'))
        except Exception as e:
            logging.exception("Failed to create the local Service Instance as 'dcui', continuing... : ")
            return

    # set out ID in context to be used in request - so we'll see it in logs
    reqCtx = VmomiSupport.GetRequestContext()
    reqCtx["realUser"] = 'dvolplug'
    atexit.register(pyVim.connect.Disconnect, _service_instance)

def get_si():
    '''
	Return a connection to the local SI
	'''
    with lockManager.get_lock('siLock'):
        global _service_instance
        try:
            _service_instance.CurrentTime()
        except:
            # service_instance is invalid (could be stale)
            # reset it to None and try to connect again.
            _service_instance = None
            connectLocalSi()

        return _service_instance

def is_service_available():
    """
    Check if connection to hostd service is available
    """
    if not get_si():
        return False
    return True

def get_datastore_names_list():
    """returns names of known datastores"""
    return [i[0] for i in vmdk_utils.get_datastores()]

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
        ds, disk_path = d.backing.fileName.rsplit("]", 1)
        datastore = ds[1:]
        backing_disk = disk_path.lstrip()
        logging.debug("findDeviceByPath: datastore=%s, backing_disk=%s", datastore, backing_disk)

        # Construct the parent dir and vmdk name, resolving
        # links if any.
        dvol_dir = os.path.dirname(vmdk_path)
        datastore_prefix = os.path.realpath(os.path.join("/vmfs/volumes", datastore)) + '/'
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
    vol_meta[kv.ATTACHED_VM_NAME] = None
    if not kv.setAll(vmdk_path, vol_meta):
       msg = "Failed to save volume metadata for {0}.".format(vmdk_path)
       logging.warning("reset_vol_meta: " + msg)
       return err(msg)

def setStatusAttached(vmdk_path, vm, vm_dev_info=None):
    '''Sets metadata for vmdk_path to (attached, attachedToVM=uuid'''
    logging.debug("Set status=attached disk=%s VM name=%s uuid=%s", vmdk_path,
                  vm.config.name, vm.config.uuid)
    vol_meta = kv.getAll(vmdk_path)
    if not vol_meta:
        vol_meta = {}
    vol_meta[kv.STATUS] = kv.ATTACHED
    vol_meta[kv.ATTACHED_VM_UUID] = vm.config.uuid
    vol_meta[kv.ATTACHED_VM_NAME] = vm.config.name
    if vm_dev_info:
        vol_meta[kv.ATTACHED_VM_DEV] = vm_dev_info
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
        del vol_meta[kv.ATTACHED_VM_DEV]
    except:
        pass
    if not kv.setAll(vmdk_path, vol_meta):
        logging.warning("Detach: Failed to save Disk metadata for %s", vmdk_path)


def getStatusAttached(vmdk_path):
    '''
    Returns (attached, uuid, attach_as, vm_name) tuple. For 'detached' status
    uuid and vm_name are None.
    '''

    vol_meta = kv.getAll(vmdk_path)
    try:
        attach_as = vol_meta[kv.VOL_OPTS][kv.ATTACH_AS]
    except:
        attach_as = kv.DEFAULT_ATTACH_AS

    if not vol_meta or kv.STATUS not in vol_meta:
        return False, None, attach_as, None

    attached = (vol_meta[kv.STATUS] == kv.ATTACHED)
    try:
        uuid = vol_meta[kv.ATTACHED_VM_UUID]
    except:
        uuid = None

    try:
        vm_name = vol_meta[kv.ATTACHED_VM_NAME]
    except:
        vm_name = None

    return attached, uuid, attach_as, vm_name

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
                   logging.warning(msg)
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
             logging.warning(msg)
             return err(msg)
       else:
          logging.warning("Failed to find VM (id %s) attaching the disk %s, resetting volume metadata",
                          kv_uuid, vmdk_path)
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

    kv_status_attached, kv_uuid, attach_mode, _ = getStatusAttached(vmdk_path)
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

    vm_dev_info = dev_info(disk_slot, pci_slot_number)

    setStatusAttached(vmdk_path, vm, vm_dev_info)
    logging.info("Disk %s successfully attached. controller pci_slot_number=%s, disk_slot=%d",
                 vmdk_path, pci_slot_number, disk_slot)

    return vm_dev_info


def err(string):
    return {u'Error': string}


def disk_detach(vmdk_path, vm):
    """detach disk (by full path) from a vm and return None or err(msg)"""

    device = findDeviceByPath(vmdk_path, vm)

    if not device:
       # Could happen if the disk attached to a different VM - attach fails
       # and docker will insist to sending "unmount/detach" which also fails.
       # Or Plugin retrying operation due to socket errors #1076
       # Return success since disk is anyway not attached
       logging.warning("*** Detach disk={0} not found. VM={1}".format(
                       vmdk_path, vm.config.uuid))
       return None

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
def set_vol_opts(name, tenant_name, options):
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

    logging.debug("set_vol_opts: name=%s options=%s vol_name=%s, datastore=%s",
                  name, options, vol_name, datastore)

    if not datastore:
       msg = "Invalid datastore '{0}'.\n".format(datastore)
       logging.warning(msg)
       return False

    datastore_url = vmdk_utils.get_datastore_url(datastore)

    # try to set opts on a volume which was created by a non-exist tenant
    # fail the request
    if tenant_name:
    # if tenant_name is "None", which means the function is called without multi-tenancy
        error_info = auth_api.check_tenant_exist(tenant_name)
        if not error_info:
            logging.warning(error_code_to_message[ErrorCode.TENANT_NOT_EXIST].format(tenant_name))
            return False

    # get /vmfs/volumes/<datastore_url>/dockvols path on ESX:
    path, errMsg = get_vol_path(datastore, tenant_name)

    if path is None:
       msg = "Failed to get datastore path {0}".format(path)
       logging.warning(msg)
       return False

    vmdk_path = vmdk_utils.get_vmdk_path(path, vol_name)

    logging.debug("set_vol_opts: path=%s vmdk_path=%s", path, vmdk_path)

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

def msg_about_signal(signalnum, details="exiting"):
    logging.warn("Received signal num: %d, %s.", signalnum, details)
    logging.warn("Operations in flight will be terminated")

def signal_handler_stop(signalnum, frame):
    msg_about_signal(signalnum, "exiting")
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
            logging.debug("execRequestThread: req=%s", req)
            # If req from client does not include version number, set the version to
            # SERVER_PROTOCOL_VERSION by default to make backward compatible
            client_protocol_version = int(req["version"]) if "version" in req else SERVER_PROTOCOL_VERSION
            logging.debug("execRequestThread: version=%d", client_protocol_version)
            if client_protocol_version != SERVER_PROTOCOL_VERSION:
                if client_protocol_version < SERVER_PROTOCOL_VERSION:
                    reply_string = err("vSphere Docker Volume Service client version ({}) is older than server version ({}), "
                                    "please update the client.".format(client_protocol_version, SERVER_PROTOCOL_VERSION))
                else:
                    reply_string = err("vSphere Docker Volume Service client version ({}) is newer than server version ({}), "
                                    "please update the server.".format(client_protocol_version, SERVER_PROTOCOL_VERSION))
                send_vmci_reply(client_socket, reply_string)

            opts = req["details"]["Opts"] if "Opts" in req["details"] else {}
            reply_string = executeRequest(vm_uuid=vm_uuid,
                                vm_name=vm_name,
                                config_path=cfg_path,
                                cmd=req["cmd"],
                                full_vol_name=req["details"]["Name"],
                                opts=opts)

            logging.info("executeRequest '%s' completed with ret=%s", req["cmd"], reply_string)
            send_vmci_reply(client_socket, reply_string)

    except Exception as ex_thr:
        logging.exception("Unhandled Exception:")
        reply_string = err("Server returned an error: {0}".format(repr(ex_thr)))
        send_vmci_reply(client_socket, reply_string)

# code to grab/release VMCI listening socket
g_vmci_listening_socket = None

def vmci_grab_listening_socket(port):
    """call C code to open/bind/listen on the VMCI socket"""
    global g_vmci_listening_socket
    if  g_vmci_listening_socket:
        logging.error("VMCI Listening socket - multiple init") # message for us. Should never  happen
        return

    g_vmci_listening_socket = lib.vmci_init(c_uint(port))
    if g_vmci_listening_socket == VMCI_ERROR:
        errno = get_errno()
        raise OSError("Failed to initialize vSocket listener: %s (errno=%d)" \
                        %  (os.strerror(errno), errno))

def vmci_release_listening_socket():
    """Calls C code to release the VMCI listening socket"""
    if g_vmci_listening_socket:
        lib.vmci_close(g_vmci_listening_socket)

# load VMCI shared lib , listen on vSocket in main loop, handle requests
def handleVmciRequests(port):
    skip_count = MAX_SKIP_COUNT  # retries for vmci_get_one_op failures
    bsize = MAX_JSON_SIZE
    txt = create_string_buffer(bsize)
    cartel = c_int32()
    vmci_grab_listening_socket(port)


    while True:
        c = lib.vmci_get_one_op(g_vmci_listening_socket, byref(cartel), txt, c_int(bsize))
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

        if not get_si():
            svc_connect_err = 'Service is presently unavailable, ensure the ESXi Host Agent is running on this host'
            logging.warning(svc_connect_err)
            send_vmci_reply(client_socket, err(svc_connect_err))
            continue

        # Fire a thread to execute the request
        threadutils.start_new_thread(
            target=execRequestThread,
            args=(client_socket, cartel.value, txt.value))

    vmci_release_listening_socket() # close listening socket when the loop is over

def usage():
    print("Usage: %s -p <vSocket Port to listen on>" % sys.argv[0])

def main():
    log_config.configure()
    logging.info("==== Starting vmdkops service pid=%d ====", os.getpid())
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

        # start the daemon. Do all the task to start the listener through the daemon
        threadutils.start_new_thread(target=vm_listener.start_vm_changelistener,
                                 daemon=True)
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
    # Setting LANG environment variable if it is unset to ensure proper encoding
    if os.environ.get('LANG') is None:
        os.environ['LANG'] = "en_US.UTF-8"
        os.execve(__file__, sys.argv, os.environ)
    main()
