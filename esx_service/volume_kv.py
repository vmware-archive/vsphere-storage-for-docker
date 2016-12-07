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
## This module handles creating and managing a kv store for volumes
## (vmdks) created by the docker volume plugin on an ESX host. The
## module exposes a set of functions that allow creat/delete/get/set
## on the kv store. Currently uses side cars to keep KV pairs for
## a given volume.

import kvESX

# All possible metadata keys for the volume. New keys should be added here as
# constants pointing to strings.

# The status of the volume, whether its attached or not to a VM.
STATUS = 'status'

## Values for STATUS
ATTACHED = 'attached'
DETACHED = 'detached'

# Timestamp of volume creation
CREATED = 'created'
# Name of the VM that created the volume
CREATED_BY = 'created-by'
# The UUID of the VM that the volume is attached to
ATTACHED_VM_UUID = 'attachedVMUuid'

# Dictionary of options passed in by the user
VOL_OPTS = 'volOpts'
# Options below this line are keys in the VOL_OPTS dict.

# The name of the VSAN policy applied to the VSAN volume. Invalid for Non-VSAN
# volumes.
VSAN_POLICY_NAME = 'vsan-policy-name'

## Value for VSAN_POLICY_NAME
DEFAULT_VSAN_POLICY = '[VSAN default]'

# The size of the volume
SIZE = 'size'
DEFAULT_DISK_SIZE = "100mb"

# The disk allocation format for vmdk
DISK_ALLOCATION_FORMAT = 'diskformat'
VALID_ALLOCATION_FORMATS = {"zeroedthick": "preallocated",
                            "thin": "thin",
                            "eagerzeroedthick": "eagerZeroedThick"}
DEFAULT_ALLOCATION_FORMAT = 'thin'

# attach type. Default is independent.
ATTACH_AS = 'attach-as'

# actual values to pass to VM reconfig events. The values are defined in
# vSphere SDK , and for pythin can be found here:
# https://github.com/vmware/pyvmomi/blob/master/docs/vim/vm/device/VirtualDiskOption/DiskMode.rst
# We support the following ones:
INDEPENDENT = 'independent_persistent'  # does not participate in vm snashot
DEPENDENT   = 'persistent' # does participated in VM snapshot
DEFAULT_ATTACH_AS = INDEPENDENT
ATTACH_AS_TYPES = [INDEPENDENT, DEPENDENT]

# Access types
ACCESS = 'access'
ACCESS_READONLY = 'read-only'
ACCESS_READWRITE = 'read-write'
DEFAULT_ACCESS = ACCESS_READWRITE
ACCESS_TYPES = [ACCESS_READWRITE, ACCESS_READONLY]

# Filesystem type
# This option is handled in the volume-plugin at the docker host, and tracked in volume metadata. 
FILESYSTEM_TYPE = 'fstype'
DEFAULT_FILESYSTEM_TYPE = 'ext4'

# Clone references
CLONE_FROM = 'clone-from' # clone volume parent
DEFAULT_CLONE_FROM = 'None'

# Create a kv store object for this volume identified by vol_path
# Create the side car or open if it exists.
def init():
    kvESX.kv_esx_init()

def create(vol_path, vol_meta):
    """
    Create a side car KV store for given vol_path.
    Return true if successful, false otherwise
    """
    return kvESX.create(vol_path, vol_meta)


def delete(vol_path):
    """
    Delete a kv store object for this volume identified by vol_path.
    Return true if successful, false otherwise
    """
    return kvESX.delete(vol_path)


def getAll(vol_path):
    """
    Return the entire meta-data for the given vol_path.
    Return true if successful, false otherwise
    """
    return kvESX.load(vol_path)


def setAll(vol_path, vol_meta):
    """
    Store the meta-data for a given vol-path
    Return true if successful, false otherwise
    """
    if vol_meta:
        return kvESX.save(vol_path, vol_meta)
    # No data to save
    return True


# Set a string value for a given key(index)
def set_kv(vol_path, key, val):
    vol_meta = kvESX.load(vol_path)

    if not vol_meta:
        return False

    vol_meta[key] = val

    return kvESX.save(vol_path, vol_meta)


def get_kv(vol_path, key):
    """
    Return a string value for the given key, or None if the key is not present.
    """
    vol_meta = kvESX.load(vol_path)

    if not vol_meta:
        return None

    if key in vol_meta:
        return vol_meta[key]
    else:
        return None


def remove(vol_path, key):
    """
    Remove a key/value pair from the store. Return true on success, false on
    error.
    """
    vol_meta = kvESX.load(vol_path)

    if not vol_meta:
        return False

    if key in vol_meta:
        del vol_meta[key]

    return kvESX.save(vol_path, vol_meta)

def get_vol_info(vol_path):
   return kvESX.get_info(vol_path)
