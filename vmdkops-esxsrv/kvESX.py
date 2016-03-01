##
## Implements the interface for side car based implementation
## of a KV store for vmdks.
##

from ctypes import *
import re
import dvolKeys

# Side car create/open options
KV_SIDECAR_CREATE = 0

# Max KV pair length
KV_MAX_LEN = 128

# Start size for a side car
KV_CREATE_SIZE = 0

# Backdoor into VSphere lib APIs
diskLib = "/lib/libvmsnapshot.so"
lib = None
dVolKey = None

# Maps to OPEN_BUFFERED | OPEN_LOCK | OPEN_NOFILTERS
# all vmdks are opened with these flags
vmdkOpenFlags = 524312

# Load the library if not done already
def loadDiskLib():
   global lib
   if not lib:
      lib = CDLL(diskLib)
      lib.DiskLib_Init.argtypes = []
      lib.DiskLib_Init.restype = c_bool
      lib.DiskLib_Init()
   return None


# Loads the back-door library used to access ESX disk lib API. Create arg/result
# definitions for those that we use.
def kvESXInit(kvPrefix):
    global dVolKey
    dVolKey = kvPrefix
    print "Using %s for side car" % dVolKey

    # Load disklib APIs
    loadDiskLib()

    # Define all of the functions we are interested in
    lib.DiskLib_OpenWithInfo.argtypes = [c_char_p, c_int32, POINTER(c_uint32), POINTER(c_uint32), POINTER(c_uint32)]
    lib.DiskLib_OpenWithInfo.restype = c_int32

    lib.DiskLib_Close.argtypes = [c_uint32]
    lib.DiskLib_Close.restype = c_int32

    lib.DiskLib_SidecarCreate.argtypes = [c_uint32, c_char_p, c_uint64, c_int32, POINTER(c_uint32)]
    lib.DiskLib_SidecarCreate.restype = c_int32

    lib.DiskLib_SidecarOpen.argtypes = [c_uint32, c_char_p, c_int32, POINTER(c_uint32)]
    lib.DiskLib_SidecarOpen.restype = c_int32

    lib.DiskLib_SidecarClose.argtypes = [c_uint32, c_char_p, POINTER(c_uint32)]
    lib.DiskLib_SidecarClose.restype = c_int32

    lib.DiskLib_SidecarMakeFileName.argtypes = [c_char_p, c_char_p]
    lib.DiskLib_SidecarMakeFileName.restype = c_char_p

    lib.ObjLib_Pread.argtypes = [c_uint32, c_void_p, c_uint64, c_uint64]
    lib.ObjLib_Pread.restype = c_int32

    lib.ObjLib_Pwrite.argtypes = [c_uint32, c_void_p, c_uint64, c_uint64]
    lib.ObjLib_Pwrite.restype = c_int32

    lib.DiskLib_DBGet.argtypes = [c_uint32, c_char_p, POINTER(c_char_p)]
    lib.DiskLib_DBGet.restype = c_int32

    lib.DiskLib_DBSet.argtypes = [c_uint32, c_char_p, c_char_p]
    lib.DiskLib_DBSet.restype = c_int32
    
    return None

def kvVolPathOpen(volpath):
   dhandle = c_uint32(0)
   ihandle = c_uint32(0)
   key = c_uint32(0)
   objHandle = c_uint32(0)
   res = c_int32(0)

   res = lib.DiskLib_OpenWithInfo(volpath, vmdkOpenFlags, pointer(key), pointer(dhandle), pointer(ihandle))

   if res != 0:
      print "Open %s failed - %x" % (volpath, res)

   return dhandle

# Create the side car for the volume identified by volpath.
def sidecarCreate(volpath):
   disk = c_uint32(0)
   objHandle = c_uint32(0)
   res = 0

   disk = kvVolPathOpen(volpath)

   if disk == 0:
      return False

   res = lib.DiskLib_SidecarCreate(disk, dVolKey, KV_CREATE_SIZE, KV_SIDECAR_CREATE, pointer(objHandle))
   if res != 0:
      print "Side car create for %s failed - %x" % (volpath, res)
      lib.DiskLib_Close(disk)
      return False
   
   lib.DiskLib_SidecarClose(disk, dVolKey, pointer(objHandle))
   lib.DiskLib_Close(disk)

   sidecarFile = lib.DiskLib_SidecarMakeFileName(volpath, dVolKey)
   fh = open(sidecarFile, "rb+")
   fh.seek(0, 0)
   fh.write("# Persistent volume plugin private data.\n")
   fh.close()
   return True

# Delete the the side car for the given volume
def sidecarDelete(volpath):
   disk = c_uint32(0)
   res = 0

   disk = kvVolPathOpen(volpath)

   if disk == 0:
      return False

   res = lib.DiskLib_SidecarDelete(disk, dVolKey)
   if res != 0:
      print "Side car delete for %s failed - %x" % (volpath, res)
      return False
   
   lib.DiskLib_Close(disk)
   return True

# Return the value string given a key (index).
def sidecarGetKey(volpath, key):
   sidecarFile = lib.DiskLib_SidecarMakeFileName(volpath, dVolKey)

   kvloc = (key + 1) * KV_MAX_LEN

   fh = open(sidecarFile, "rb+")
   # Use the key as index read side car at the
   # offset.
   fh.seek(kvloc, 0)
   kvstr = fh.read(KV_MAX_LEN)
   fh.close()
   val = re.split("=", kvstr)
   return val[1]

# Write the KV pair at the offset for the given key index. Right now
# this uses read/write calls, later will move to ESX ObjLib_Pread/PWrite
# calls.
def sidecarSetKey(volpath, key, val):
   sidecarFile = lib.DiskLib_SidecarMakeFileName(volpath, dVolKey)

   kvloc = (key + 1) * KV_MAX_LEN

   fh = open(sidecarFile, "rb+")
   # Use the key as index into the key name array, append
   # the value passed in and write the side car at the
   # offset (each key-value pair is allowed
   # a max length of KV_MAX_LEN.

   kvstr = dvolKeys.kv_strings[key] + "=" + val
   print "setting %s at %d" % (kvstr, kvloc)

   fh.seek(kvloc, 0)
   print "positioned at :", fh.tell()
   fh.write(kvstr)
   fh.close()
   return True

# No-op for a side car, the key is retained till the side car is deleted
def sidecarDeleteKey(volpath, key):
   return True

