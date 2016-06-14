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

##
## Implements the interface for side car based implementation
## of a KV store for vmdks.
##

from ctypes import \
        CDLL, POINTER, byref, \
        c_void_p, c_char_p, c_int32, c_bool, c_uint32, c_uint64
import json
import logging

# Side car create/open options
KV_SIDECAR_CREATE = 0

# Start size for a side car
KV_CREATE_SIZE = 0

# VSphere lib to access ESX proprietary APIs.
DISK_LIB = "/lib/libvmsnapshot.so"
lib = None
useSideCarCreate = False
DVOL_KEY = "docker-volume-vsphere"

# Results in a buffered, locked, filter-less open,
# all vmdks are opened with these flags
VMDK_OPEN_FLAGS = 524312

# Default kv side car alignment
KV_ALIGN = 4096

# Load the disk lib API library
def loadDiskLib():
    global lib

    if not lib:
        lib = CDLL(DISK_LIB)
        lib.DiskLib_Init.argtypes = []
        lib.DiskLib_Init.restype = c_bool
        lib.DiskLib_Init()


# Loads the ESX library used to access proprietary ESX disk lib API.
# Create arg/result definitions for those that we use.
def kvESXInit():
    global useSideCarCreate

    # Load disklib APIs
    loadDiskLib()

    # Define all of the functions we are interested in
    lib.DiskLib_OpenWithInfo.argtypes = [c_char_p, c_int32, POINTER(c_uint32),
                                         POINTER(c_uint32), POINTER(c_uint32)]
    lib.DiskLib_OpenWithInfo.restype = int

    lib.DiskLib_Close.argtypes = [c_uint32]
    lib.DiskLib_Close.restype = int

    # Check if this library supports create API
    try:
        lib.DiskLib_SidecarCreate.argtypes = [c_uint32, c_char_p, c_uint64,
                                              c_int32, POINTER(c_uint32)]
        lib.DiskLib_SidecarCreate.restype = int
        useSideCarCreate = True
    except:
        # do nothing
        logging.debug(
            "ESX version doesn't support create API, using open instead.")

    lib.DiskLib_SidecarOpen.argtypes = [c_uint32, c_char_p, c_int32,
                                        POINTER(c_uint32)]
    lib.DiskLib_SidecarOpen.restype = int

    lib.DiskLib_SidecarClose.argtypes = [c_uint32, c_char_p, POINTER(c_uint32)]
    lib.DiskLib_SidecarClose.restype = int

    lib.DiskLib_SidecarMakeFileName.argtypes = [c_char_p, c_char_p]
    lib.DiskLib_SidecarMakeFileName.restype = c_char_p

    lib.ObjLib_Pread.argtypes = [c_uint32, c_void_p, c_uint64, c_uint64]
    lib.ObjLib_Pread.restype = c_int32

    lib.ObjLib_Pwrite.argtypes = [c_uint32, c_void_p, c_uint64, c_uint64]
    lib.ObjLib_Pwrite.restype = c_int32

    lib.DiskLib_DBGet.argtypes = [c_uint32, c_char_p, POINTER(c_char_p)]
    lib.DiskLib_DBGet.restype = c_uint32

    lib.DiskLib_DBSet.argtypes = [c_uint32, c_char_p, c_char_p]
    lib.DiskLib_DBSet.restype = c_uint32

    return None


# Open a VMDK given its path, the VMDK is opened locked just to
# ensure we have exclusive access and its not already in use.
def volOpenPath(volpath):
    dhandle = c_uint32(0)
    ihandle = c_uint32(0)
    key = c_uint32(0)

    res = lib.DiskLib_OpenWithInfo(volpath, VMDK_OPEN_FLAGS, byref(key),
                                   byref(dhandle), byref(ihandle))

    if res != 0:
        logging.warning("Open %s failed - %x", volpath, res)

    return dhandle


# Create the side car for the volume identified by volpath.
def create(volpath, kv_dict):
    disk = c_uint32(0)
    objHandle = c_uint32(0)

    disk = volOpenPath(volpath)

    if disk == c_uint32(0):
        return False

    if useSideCarCreate:
        res = lib.DiskLib_SidecarCreate(disk, DVOL_KEY, KV_CREATE_SIZE,
                                        KV_SIDECAR_CREATE, byref(objHandle))
    else:
        res = lib.DiskLib_SidecarOpen(disk, DVOL_KEY, KV_SIDECAR_CREATE,
                                      byref(objHandle))

    if res != 0:
        logging.warning("Side car create for %s failed - %x", volpath, res)
        lib.DiskLib_Close(disk)
        return False

    lib.DiskLib_SidecarClose(disk, DVOL_KEY, byref(objHandle))
    lib.DiskLib_Close(disk)

    return save(volpath, kv_dict)


# Delete the the side car for the given volume
def delete(volpath):
    disk = c_uint32(0)

    disk = volOpenPath(volpath)

    if disk == c_uint32(0):
        return False

    res = lib.DiskLib_SidecarDelete(disk, DVOL_KEY)
    if res != 0:
        logging.warning("Side car delete for %s failed - %x", volpath, res)
        lib.DiskLib_Close(disk)
        return False

    lib.DiskLib_Close(disk)
    return True


# Align a given string to the specified block boundary.
def align_str(kv_str, block):
   # Align string to the next block boundary. The -1 is to accommodate
   # a newline at the end of the string.
   aligned_len = int((len(kv_str) + block - 1) / block) * block - 1
   return '{:<{width}}\n'.format(kv_str, width=aligned_len)


# Load and return dictionary from the sidecar
def load(volpath):
    metaFile = lib.DiskLib_SidecarMakeFileName(volpath, DVOL_KEY)

    try:
       with open(metaFile, "r") as fh:
          kv_str = fh.read()
    except:
        logging.exception("Failed to access %s", metaFile);
        return None

    try:
       return json.loads(kv_str)
    except ValueError:
       logging.exception("Failed to decode meta-data for %s", volpath);
       return None


# Save the dictionary to side car.
def save(volpath, kv_dict):
    metaFile = lib.DiskLib_SidecarMakeFileName(volpath, DVOL_KEY)

    kv_str = json.dumps(kv_dict)

    try:
       with open(metaFile, "w") as fh:
          fh.write(align_str(kv_str, KV_ALIGN))
    except:
        logging.exception("Failed to save meta-data for %s", volpath);
        return False

    return True

