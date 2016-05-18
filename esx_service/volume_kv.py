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

# Default meta-data for a volume created by the plugin, keys can be
# added or removed during the life of a volume. The below list
# is whats included by default when a kv store is created.
#
# 1. status - the status of the volume, whether its attached or
#             not to a VM.
# 2. created - timestamp of creation
# 3. created-by - The VM that created the volume
# 4. volOpts - the string of options with which the volume was
#              created.


# Create a kv store object for this volume identified by vol_path
# Create the side car or open if it exists.
def init():
    kvESX.kvESXInit()


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
