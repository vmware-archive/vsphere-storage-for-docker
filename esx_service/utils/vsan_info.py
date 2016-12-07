#!/usr/bin/env python
#
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
# limitations under the License.#

# Simple API to access VSAN policy information.
# Uses objtool to extract and set policy in VSAN objects
#
# To obtain a connection to the local SI use vmdk_ops.get_si()
#

import logging
import json
import os.path
import vmdk_ops

OBJTOOL = '/usr/lib/vmware/osfs/bin/objtool '
OBJTOOL_SET_POLICY = OBJTOOL + "setPolicy -u {0} -p '{1}'"
OBJTOOL_GET_ATTR = OBJTOOL + "getAttr -u {0} --format=json"


def get_vsan_datastore():
    """Returns Datastore management object for vsanDatastore, or None"""
    si = vmdk_ops.get_si()
    stores = si.content.rootFolder.childEntity[0].datastore
    try:
        return [d for d in stores if d.summary.type == "vsan"][0]
    except:
        return None


def get_vsan_dockvols_path():
    """
    Return the VSAN datastore dockvols path for a given cluster. Default to the
    first datastore for now, so we can test without VSAN.
    """
    datastore = get_vsan_datastore()
    if datastore:
        path, err = vmdk_ops.get_vol_path(datastore.info.name)
        if not err:
            return path
    else:
        return None


def is_on_vsan(vmdk_path):
    """Returns True if path is on VSAN datastore, False otherwise"""
    ds = get_vsan_datastore()
    if not ds:
        return False
    return os.path.realpath(vmdk_path).startswith(os.path.realpath(
        ds.info.url))


def set_policy(vmdk_path, policy_string):
    """
    Sets policy for an object backing <vmdk_path> to <policy>.
    Returns True on success
    """
    uuid = vmdk_ops.get_vsan_uuid(vmdk_path)
    rc, out = vmdk_ops.RunCommand(OBJTOOL_SET_POLICY.format(uuid,
                                                            policy_string))
    if rc != 0:
        logging.warning("Failed to set policy for %s : %s", vmdk_path, out)
        return False
    return True


def same_policy(vmdk_path, policy_string):
    """"
    Returns True if VSAN object backing <vmdk_path>  has <policy_string>
    """
    existing_policy_string = get_policy(vmdk_path).expandtabs(1).replace(" ",
                                                                         "")
    return existing_policy_string == policy_string.expandtabs(1).replace(" ",
                                                                         "")


def get_policy(vmdk_path):
    """
    Returns VSAN policy string for VSAN object backing <vmdk_path>
    Throws exception if the path is not found or it is not a VSAN object
    """
    uuid = vmdk_ops.get_vsan_uuid(vmdk_path)
    rc, out = vmdk_ops.RunCommand(OBJTOOL_GET_ATTR.format(uuid))
    if rc != 0:
        logging.warning("Failed to get policy for %s : %s", vmdk_path, out)
        return None
    policy = json.loads(out)['Policy']
    return policy
