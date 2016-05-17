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

# Module for VSAN storage policy creation and configuration

import os
import vmdk_utils
import volume_kv as kv


def create(name, content):
    """
    Create a new storage policy and save it as dockvols/policies/name in
    the VSAN datastore. If there are VSAN volumes currently using a policy with
    the same name, creation will fail. Return a string on error and None on
    success.
    """
    if policy_exists(name):
        return 'Error: Policy already exists'

    if not validate_vsan_policy_string(content):
        return 'Error: Invalid policy string'

    return create_policy_file(name, content)


def create_policy_file(name, content):
    """
    Create a storage policy file in {path}/policies/name. If the policy
    already exists return an error string, otherwise return None.
    """
    path = vmdk_utils.get_vsan_datastore()
    policies_dir = os.path.join(path, 'policies')
    try:
        os.mkdir(policies_dir)
    except OSError:
        pass

    filename = os.path.join(policies_dir, name)
    if os.path.isfile(filename):
        return 'Error: {0} already exists'.format(name)

    try:
        with open(filename, 'w') as f:
            f.write(content)
            f.write('\n')
    except:
        return 'Error: Failed to open {0} for writing'.format(filename)

    return None


def delete(name):
    """
    Remove a given policy. If the policy does not exist return an error string,
    otherwise return None
    """
    path = vmdk_utils.get_vsan_datastore()
    vmdk = policy_in_use(path, name)
    if vmdk:
        return 'Error: Cannot remove. Policy is in use by {0}'.format(vmdk)
    try:
        os.remove(policy_path(name))
    except:
        return 'Error: {0} does not exist'.format(name)

    return None


def get_policies():
    """ Return a dict of all policies. """
    policies = {}
    path = os.path.join(vmdk_utils.get_vsan_datastore(), 'policies')
    for name in os.listdir(path):
        with open(os.path.join(path, name)) as f:
            content = f.read()
        policies[name] = content
    return policies


def list_volumes_and_policies():
    """ Return a list of vmdks and the policies in use"""
    vmdks_and_policies = []
    path = vmdk_utils.get_vsan_datastore()
    for vmdk in vmdk_utils.list_vmdks(path):
        policy = kv_get_vsan_policy_name(os.path.join(path, vmdk))
        vmdks_and_policies.append({'volume': vmdk, 'policy': policy})
    return vmdks_and_policies


def policy_exists(name):
    """ Check if the policy file exists """
    return os.path.isfile(policy_path(name))


def policy_path(name):
    """ Return the path to a given policy file """
    return os.path.join(vmdk_utils.get_vsan_datastore(), 'policies', name)


def kv_get_vsan_policy_name(path):
    """
    Take a path for a vmdk and return a policy name if it exists or None if it
    doesn't
    """

    try:
        return kv.getAll(path)[u'volOpts']['vsan-policy-name']
    except:
        return None


def policy_in_use(path, name):
    """
    Check if a policy is in use by a VMDK and return the name of the first VMDK
    using it if it is, None otherwise
    """
    for vmdk in vmdk_utils.list_vmdks(path):
        policy = kv_get_vsan_policy_name(os.path.join(path, vmdk))
        if policy == name:
            return vmdk
    return None


def validate_vsan_policy_string(content):
    """
    Stub for a function that validates the syntax of a vsan policy string
    """
    return True
