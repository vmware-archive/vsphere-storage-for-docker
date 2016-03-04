##
## This module handles creating and managing a kv store for volumes
## (vmdks) created by the docker volume plugin on an ESX host. The
## module exposes a set of functions that allow creat/delete/get/set
## on the kv store. Currently uses side cars to keep KV pairs for 
## a given volume.

import kvESX

# Default meta-data for a plug-vol
defMeta = {'name':'plugvol',
           'controller':0,
           'slot':0,
           'vmID':1,
           'daemonID':1,
           'status':'detached',
           'volOpts':'None',
           'cbrcEnabled':False,
           'ioFilters':'None'};

# Create a kv store object for this volume identified by volPath
# Create the side car or open if it exists.
def init():
   kvESX.kvESXInit()
   return None


# Create a side car KV store for given volpath
def create(volPath, name, vm, daemon, status):
   plugVolMeta = defMeta.copy()

   plugVolMeta['name'] = name
   plugVolMeta['vmID'] = vm
   plugVolMeta['daemonID'] = daemon
   plugVolMeta['status'] = status

   res = kvESX.create(volPath, plugVolMeta)

   if res != True:
      print "KV store create failed"
      return False

   return True

# Delete a kv store object for this volume identified by volPath
def delete(volPath):
   return kvESX.delete(volPath)

# Set a string value for a given key(index)
def set(volPath, key, val):
   plugvolMeta = kvESX.load(volPath)

   plugvolMeta[key] = val

   return kvESX.save(volPath, plugvolMeta)


# Get value for a given key (index), returns a string thats the value
# for the key
def get(volPath, key):
   plugvolMeta = kvESX.load(volPath)

   return plugvolMeta[key]

# No-op for side car based KV pairs, once added KV pairs live till
# the side car is deleted.
def remove(volPath, key):
   plugvolMeta = kvESX.load(volPath)
   del plugvolMeta[key]

   return kvESX.save(volPath, plugvolMeta)
