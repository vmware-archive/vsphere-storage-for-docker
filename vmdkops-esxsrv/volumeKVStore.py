##
## This module handles creating and managing a kv store for volumes
## (vmdks) created by the docker volume plugin on an ESX host. The
## module exposes a set of functions that allow creat/delete/get/set
## on the kv store. Currently uses side cars to keep KV pairs for 
## a given volume.

import kvESX
import dvolKeys

kvPrefix = "DVOL"

# Create a kv store object for this volume identified by volPath
# Create the side car or open if it exists.
def kvStoreInit():
   kvESX.kvESXInit(kvPrefix)
   return None


# Create a side car KV store for given volpath
def kvStoreCreate(volPath, vmID, daemonID, volStatus):
   res = kvESX.sidecarCreate(volPath)

   if res != True:
      print "KV store create failed"
      return False
   kvESX.sidecarSetKey(volPath, dvolKeys.DVOL_VM_ID, vmID)
   kvESX.sidecarSetKey(volPath, dvolKeys.DVOL_DAEMON_ID, daemonID)
   kvESX.sidecarSetKey(volPath, dvolKeys.DVOL_STATUS, volStatus)

   return True

# Delete a kv store object for this volume identified by volPath
def kvStoreDelete(volPath):
   return kvESX.sidecarDelete(volPath)

# Set a string value for a given key(index)
def kvStoreSet(volPath, key, val):
   return kvESX.sidecarSetKey(volPath, key, val)


# Get value for a given key (index), returns a string thats the value
# for the key
def kvStoreGet(volPath, key):
   return kvESX.sidecarGetKey(volPath, key)

# No-op for side car based KV pairs, once added KV pairs live till
# the side car is deleted.
def kvStoreRemove(volPath, key):
   return kvESX.sidecarDeleteKey(volPath)
