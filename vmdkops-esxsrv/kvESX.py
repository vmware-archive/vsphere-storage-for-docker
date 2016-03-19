##
## Implements the interface for side car based implementation
## of a KV store for vmdks.
##

from ctypes import *
import json
import subprocess
import logging

# Side car create/open options
KV_SIDECAR_CREATE = 0

# Start size for a side car
KV_CREATE_SIZE = 0

# VSphere lib to access ESX proprietary APIs.
diskLib = "/lib/libvmsnapshot.so"
lib = None
useSideCarCreate = False
dVolKey = "vmdk-plugin-vol"

# Maps to OPEN_BUFFERED | OPEN_LOCK | OPEN_NOFILTERS
# all vmdks are opened with these flags
vmdkOpenFlags = 524312

# Load the disk lib API library
def loadDiskLib():
   global lib

   if not lib:
      lib = CDLL(diskLib)
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
    lib.DiskLib_OpenWithInfo.argtypes = [c_char_p, c_int32, POINTER(c_uint32), POINTER(c_uint32), POINTER(c_uint32)]
    lib.DiskLib_OpenWithInfo.restype = c_uint32

    lib.DiskLib_Close.argtypes = [c_uint32]
    lib.DiskLib_Close.restype = c_uint32

    # Check if this library supports create API
    try: 
       lib.DiskLib_SidecarCreate.argtypes = [c_uint32, c_char_p, c_uint64, c_int32, POINTER(c_uint32)]
       lib.DiskLib_SidecarCreate.restype = c_uint32
       useSideCarCreate = True
    except:
       # do nothing
       logging.debug ("ESX version doesn't support create API, using open instead.")

    lib.DiskLib_SidecarOpen.argtypes = [c_uint32, c_char_p, c_int32, POINTER(c_uint32)]
    lib.DiskLib_SidecarOpen.restype = c_uint32

    lib.DiskLib_SidecarClose.argtypes = [c_uint32, c_char_p, POINTER(c_uint32)]
    lib.DiskLib_SidecarClose.restype = c_uint32

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
   objHandle = c_uint32(0)
   res = c_uint32(0)

   res = lib.DiskLib_OpenWithInfo(volpath, vmdkOpenFlags, pointer(key), pointer(dhandle), pointer(ihandle))

   if res != 0:
       logging.warning ("Open %s failed - %x" % (volpath, res))

   return dhandle

# Create the side car for the volume identified by volpath.
def create(volpath, kvDict):
   disk = c_uint32(0)
   objHandle = c_uint32(0)
   res = c_uint32(0)

   disk = volOpenPath(volpath)

   if disk == c_uint32(0):
      return False

   if useSideCarCreate:
      res = lib.DiskLib_SidecarCreate(disk, dVolKey, KV_CREATE_SIZE, KV_SIDECAR_CREATE, pointer(objHandle))
   else:
      res = lib.DiskLib_SidecarOpen(disk, dVolKey, KV_SIDECAR_CREATE, pointer(objHandle))

   if res != 0:
      logging.warning ("Side car create for %s failed - %x" % (volpath, res))
      lib.DiskLib_Close(disk)
      return False

   lib.DiskLib_SidecarClose(disk, dVolKey, pointer(objHandle))
   lib.DiskLib_Close(disk)

   return save(volpath, kvDict)

# Delete the the side car for the given volume
def delete(volpath):
   disk = c_uint32(0)
   res = c_uint32(0)

   disk = volOpenPath(volpath)

   if disk == c_uint32(0):
      return False

   res = lib.DiskLib_SidecarDelete(disk, dVolKey)
   if res != 0:
      logging.warning ("Side car delete for %s failed - %x" % (volpath, res))
      lib.DiskLib_Close(disk)
      return False

   lib.DiskLib_Close(disk)
   return True

# Load and return dictionary from the sidecar
def load(volpath):
   metaFile = lib.DiskLib_SidecarMakeFileName(volpath, dVolKey)

   try:
      fh = open(metaFile, "r+")
   except IOError:
      return None

   kvDict = json.load(fh)

   fh.close()

   return kvDict

# Save the dictionary to side car.
def save(volpath, kvDict):
   metaFile = lib.DiskLib_SidecarMakeFileName(volpath, dVolKey)

   try:
      fh = open(metaFile, "w+")
   except IOError:
      return False

   json.dump(kvDict, fh)

   fh.close()
   return True

