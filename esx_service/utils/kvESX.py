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

"""
Implements the interface for side car based implementation
of a KV store for vmdks.
"""

from ctypes import \
        CDLL, POINTER, byref, Structure,\
        c_char_p, c_int32, c_bool, c_uint32, c_uint64
import json
import logging
import sys
import errno
import time
import threadutils
import vmdk_utils

# Python version 3.5.1
PYTHON64_VERSION = 50659824

# Conversion constants
KB = 1024
MB = (1024 * 1024)
GB = (MB * 1024)

# Side car create/open options
KV_SIDECAR_CREATE = 0

# Start size for a side car
KV_CREATE_SIZE = 0

# VSphere lib to access ESX proprietary APIs.
DISK_LIB64 = "/lib64/libvmsnapshot.so"
DISK_LIB = "/lib/libvmsnapshot.so"
lib = None
use_sidecar_create = False
DVOL_KEY = "docker-volume-vsphere"

# Volume attributes
VOL_SIZE = 'size'
VOL_ALLOC = 'allocated'

# Results in a buffered, locked, filter-less open,
# all vmdks are opened with these flags
VMDK_OPEN_DEFAULT = 524312
VMDK_OPEN_DISKCHAIN_NOIO = 524291
VMDK_MAX_SNAPS = 32

# Default kv side car alignment
KV_ALIGN = 4096

# Flag to track the version of Python on the platform
is_64bits = False

# Get a Reentrant locking decorator
diskLibLock = threadutils.get_lock_decorator(reentrant=True)


class disk_info(Structure):
    _fields_ = [('size', c_uint64),
                ('allocated', c_uint64),
                ('fill1', c_uint64),
                ('fill2', c_uint64),
                ('fill3', c_uint32),
                ('fill4', c_uint32)]


@diskLibLock
def load_disk_lib(lib_name):
    """
    Load the disk lib API library
    """
    global lib
    if not lib:
        lib = CDLL(lib_name)
        lib.DiskLib_Init.argtypes = []
        lib.DiskLib_Init.restype = c_bool
        lib.DiskLib_Init()
    return


@diskLibLock
def disk_lib_init():
    """
    Initialize disk lib interfaces
    """
    global is_64bits
    global use_sidecar_create

    # Define arg types for disk lib apis.
    if sys.hexversion >= PYTHON64_VERSION:
        is_64bits = True
        load_disk_lib(DISK_LIB64)
        lib.DiskLib_OpenWithInfo.argtypes = [c_char_p, c_int32,
                                             POINTER(c_uint32),
                                             POINTER(c_uint64),
                                             POINTER(c_uint64)]
        lib.DiskLib_Close.argtypes = [c_uint64]

        lib.DiskLib_SidecarOpen.argtypes = [c_uint64, c_char_p, c_int32,
                                            POINTER(c_uint64)]
        lib.DiskLib_SidecarClose.argtypes = [c_uint64, c_char_p,
                                             POINTER(c_uint64)]
        lib.DiskLib_DBGet.argtypes = [c_uint64, c_char_p, POINTER(c_char_p)]
        lib.DiskLib_DBSet.argtypes = [c_uint64, c_char_p, c_char_p]
        lib.DiskLib_GetSize.argtypes = [c_uint64, c_uint32,
                                        c_uint32, POINTER(disk_info)]
        # Check if this library supports create API
        try:
            lib.DiskLib_SidecarCreate.argtypes = [c_uint64, c_char_p, c_uint64,
                                                  c_int32, POINTER(c_uint64)]
            lib.DiskLib_SidecarCreate.restype = int
            use_sidecar_create = True
        except:
            logging.debug(
                "ESX version doesn't support create API, using open instead.")
            pass
    else:
        load_disk_lib(DISK_LIB)
        lib.DiskLib_OpenWithInfo.argtypes = [c_char_p, c_int32,
                                             POINTER(c_uint32),
                                             POINTER(c_uint32),
                                             POINTER(c_uint32)]
        lib.DiskLib_Close.argtypes = [c_uint32]
        lib.DiskLib_SidecarOpen.argtypes = [c_uint32, c_char_p, c_int32,
                                            POINTER(c_uint32)]
        lib.DiskLib_SidecarClose.argtypes = [c_uint32, c_char_p,
                                             POINTER(c_uint32)]
        lib.DiskLib_DBGet.argtypes = [c_uint32, c_char_p, POINTER(c_char_p)]
        lib.DiskLib_DBSet.argtypes = [c_uint32, c_char_p, c_char_p]
        lib.DiskLib_GetSize.argtypes = [c_uint32, c_uint32,
                                        c_uint32, POINTER(disk_info)]

       # Check if this library supports create API
        try:
            lib.DiskLib_SidecarCreate.argtypes = [c_uint32, c_char_p, c_uint64,
                                                  c_int32, POINTER(c_uint32)]
            lib.DiskLib_SidecarCreate.restype = int
            use_sidecar_create = True
        except:
            logging.debug(
                "ESX version doesn't support create API, using open instead.")

    lib.DiskLib_SidecarMakeFileName.argtypes = [c_char_p, c_char_p]

    # Define result types for disk lib apis.
    lib.DiskLib_OpenWithInfo.restype = int
    lib.DiskLib_Close.restype = int
    lib.DiskLib_SidecarOpen.restype = int
    lib.DiskLib_SidecarClose.restype = int
    lib.DiskLib_SidecarMakeFileName.restype = c_char_p
    lib.DiskLib_DBGet.restype = c_uint32
    lib.DiskLib_DBSet.restype = c_uint32
    lib.DiskLib_GetSize.restype = c_uint32

    return


def kv_esx_init():
    """
    Load and init disklib
    """
    disk_lib_init()


@diskLibLock
def vol_open_path(volpath, open_flags=VMDK_OPEN_DEFAULT):
    """
    Open a VMDK given its path, the VMDK is opened locked just to
    ensure we have exclusive access and its not already in use.
    """
    dhandle = get_uint(0)
    ihandle = get_uint(0)
    key = c_uint32(0)

    res = lib.DiskLib_OpenWithInfo(volpath.encode(), open_flags,
                                   byref(key), byref(dhandle),
                                   byref(ihandle))
    if res != 0:
        logging.warning("Open %s failed - %x", volpath, res)
    return dhandle


@diskLibLock
def create(volpath, kv_dict):
    """
    Create the side car for the volume identified by volpath.
    """
    obj_handle = get_uint(0)
    dhandle = vol_open_path(volpath)

    if not disk_is_valid(dhandle):
        return False
    if use_sidecar_create:
        res = lib.DiskLib_SidecarCreate(dhandle, DVOL_KEY.encode(),
                                        KV_CREATE_SIZE, KV_SIDECAR_CREATE,
                                        byref(obj_handle))
    else:
        res = lib.DiskLib_SidecarOpen(dhandle, DVOL_KEY.encode(),
                                      KV_SIDECAR_CREATE,
                                      byref(obj_handle))
    if res != 0:
        logging.warning("Side car create for %s failed - %x", volpath, res)
        lib.DiskLib_Close(dhandle)
        return False

    lib.DiskLib_SidecarClose(dhandle, DVOL_KEY.encode(), byref(obj_handle))
    lib.DiskLib_Close(dhandle)

    return save(volpath, kv_dict)


@diskLibLock
def delete(volpath):
    """
    Delete the the side car for the given volume
    """
    dhandle = vol_open_path(volpath)
    if not disk_is_valid(dhandle):
        return False
    res = lib.DiskLib_SidecarDelete(dhandle, DVOL_KEY.encode())
    if res != 0:
        logging.warning("Side car delete for %s failed - %x", volpath, res)
        lib.DiskLib_Close(dhandle)
        return False

    lib.DiskLib_Close(dhandle)
    return True


def align_str(kv_str, block):
    """
    Align a given string to the specified block boundary.
    """
    # Align string to the next block boundary. The -1 is to accommodate
    # a newline at the end of the string.
    aligned_len = int((len(kv_str) + block - 1) / block) * block - 1
    return '{:<{width}}\n'.format(kv_str, width=aligned_len)


@diskLibLock
def load(volpath):
    """
    Load and return dictionary from the sidecar
    """
    meta_file = lib.DiskLib_SidecarMakeFileName(volpath.encode(),
                                                DVOL_KEY.encode())
    retry_count = 0
    vol_name = vmdk_utils.get_volname_from_vmdk_path(volpath)
    while True:
        try:
            with open(meta_file, "r") as fh:
                kv_str = fh.read()
            break
        except IOError as open_error:
            # This is a workaround to the timing/locking with metadata files issue #626
            if open_error.errno == errno.EBUSY and retry_count <= vmdk_utils.VMDK_RETRY_COUNT:
                logging.warning("Meta file %s busy for load(), retrying...", meta_file)
                vmdk_utils.log_volume_lsof(vol_name)
                retry_count += 1
                time.sleep(vmdk_utils.VMDK_RETRY_SLEEP)
            else:
                logging.exception("Failed to access %s", meta_file)
                return None

    try:
        return json.loads(kv_str)
    except ValueError:
        # Adding this log for DEBUG
        logging.warning("kv_str from meta file is %s ", kv_str)
        logging.exception("Failed to decode meta-data for %s", volpath)
        return None


@diskLibLock
def save(volpath, kv_dict):
    """
    Save the dictionary to side car.
    """
    meta_file = lib.DiskLib_SidecarMakeFileName(volpath.encode(),
                                                DVOL_KEY.encode())
    kv_str = json.dumps(kv_dict)

    retry_count = 0
    vol_name = vmdk_utils.get_volname_from_vmdk_path(volpath)
    while True:
        try:
            with open(meta_file, "w") as fh:
                fh.write(align_str(kv_str, KV_ALIGN))
            break
        except IOError as open_error:
            # This is a workaround to the timing/locking with metadata files issue #626
            if open_error.errno == errno.EBUSY and retry_count <= vmdk_utils.VMDK_RETRY_COUNT:
                logging.warning("Meta file %s busy for save(), retrying...", meta_file)
                vmdk_utils.log_volume_lsof(vol_name)
                retry_count += 1
                time.sleep(vmdk_utils.VMDK_RETRY_SLEEP)
            else:
                logging.exception("Failed to save meta-data for %s", volpath)
                return False

    return True


@diskLibLock
def get_info(volpath):
    """
    Return disk stats for the volume
    """
    dhandle = vol_open_path(volpath, VMDK_OPEN_DISKCHAIN_NOIO)

    if not disk_is_valid(dhandle):
        logging.warning("Failed to open disk - %s", volpath)
        return None

    sinfo = disk_info()
    res = lib.DiskLib_GetSize(dhandle, 0, VMDK_MAX_SNAPS, byref(sinfo))

    lib.DiskLib_Close(dhandle)
    if res != 0:
        logging.warning("Failed to get size of disk %s - %x", volpath, res)
        return None

    return {VOL_SIZE: convert(sinfo.size), VOL_ALLOC: convert(sinfo.allocated)}


def get_uint(val):
    """
    Return a C unsigned int
    """
    if is_64bits:
        return c_uint64(val)
    else:
        return c_uint32(val)

def disk_is_valid(dhandle):
    """
    Check if disk is valid
    """
    if is_64bits:
        return dhandle.value != c_uint64(0).value
    else:
        return dhandle.value != c_uint32(0).value

def convert(size):
    """
    Size conversions
    """
    if size < KB:
        return str(size)
    elif size < MB:
        return '{0}{1}'.format(int(size / KB), 'KB')
    elif size < GB:
        return '{0}{1}'.format(int(size / MB), 'MB')
    else:
        return '{0}{1}'.format(int(size / GB), 'GB')
