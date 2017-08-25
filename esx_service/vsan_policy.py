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
import logging
import shutil
import vmdk_utils
import vsan_info
import volume_kv as kv
import vsan_info

ERROR_NO_VSAN_DATASTORE = 'Error: VSAN datastore does not exist'

def create(name, content):
    """
    Create a new storage policy and save it as dockvols/policies/name in
    the VSAN datastore. If there are VSAN volumes currently using a policy with
    the same name, creation will fail. Return a string on error and None on
    success.
    """
    datastore_path = vsan_info.get_vsan_dockvols_path()
    if not datastore_path:
        return ERROR_NO_VSAN_DATASTORE

    policies_dir = make_policies_dir(datastore_path)
    filename = os.path.join(policies_dir, name)
    if os.path.isfile(filename):
        return 'Error: Policy already exists'

    if not validate_vsan_policy_string(content):
        return 'Error: Invalid policy string'

    return create_policy_file(filename, content)


def update(name, content):
    """
    Update the content of an existing VSAN policy in the VSAN datastore.
    Update the policy content in each VSAN object currently using the policy. If
    a VSAN policy of the given name does not exist return an error string.
    Return None on success.
    """
    path = policy_path(name)
    if not path:
        return ERROR_NO_VSAN_DATASTORE

    err = update_policy_file_content(path, content)
    if err:
        return err

    return update_vsan_objects_with_policy(name, content)


def update_policy_file_content(path, content):
    """
    Update the VSAN policy file content. Return an error msg or None on success.
    """
    try:
        with open(path) as f:
            existing_content = f.read()
    except OSError as e:
        if not os.path.isfile(path):
            return 'Error: Policy {0} does not exist'.format(
                    os.path.basename(path))
        else:
            return 'Error opening existing policy file {0}: {1}'.format(path, e)

    if existing_content.strip() == content.strip():
            return 'Error: New policy is identical to old policy. Ignoring.'

    # Create a temporary file so we don't corrupt an existing policy file
    tmpfile = '{0}.tmp'.format(path)
    err = create_policy_file(tmpfile, content)
    if err:
        return err

    # Copy the original policy file to a backup file (.old)
    # The backup will be maintained in case the policy content is invalid, when
    # attempting to apply it to existing volumes.
    # Do an atomic rename of the tmpfile to the real policy file name
    try:
        shutil.copy(path, backup_policy_filename(path))
        os.rename(tmpfile, path)
    except OSError:
        print('Internal Error: Failed to update policy file contents: '
                '{0}').format(path)
        raise

    return None


def update_vsan_objects_with_policy(name, content):
    """
    Find all VSAN objects using the policy given by `name` and update the policy
    contents in their objects. Returns an error string containing the list of
    volumes that failed to update, or a msg if there were no volumes to update.
    Returns None if all volumes were updated successfully.

    Note: This function assumes datastore_path exists.
    """
    update_count = 0
    failed_updates = []
    dockvols_path = vsan_info.get_vsan_dockvols_path()
    for v in list_volumes_and_policies():
        if v['policy'] == name:
            volume_name = v['volume']
            vmdk_path = os.path.join(v['path'], volume_name)
            if vsan_info.set_policy(vmdk_path, content):
                failed_updates.append(volume_name)
            else:
                update_count = 1

    if len(failed_updates) != 0:
        if update_count == 0:
            # All volumes failed to update, so reset the original policy
            os.rename(policy_path(backup_policy_filename(name)),
                      policy_path(name))
        else:
            log_failed_updates(failed_updates, name)

        return ('Successfully updated: {0} volumes.\n'
                'Failed to update:     {1} volumes'.format(update_count,
                                                           failed_updates))

    # Remove old policy file on success
    os.remove(policy_path(backup_policy_filename(name)))
    return None

def backup_policy_filename(name):
    """ Generate a .old file from a policy name or path """
    return '{0}.old'.format(name)

def log_failed_updates(volumes, policy_name):
    """
    During policy update, some volumes may fail to have their VSAN policies
    updated. We create a file containing these volumes for debugging purposes.
    """
    filename = policy_path('{0}.failed_volume_updates'.format(policy_name))
    try:
        with open(filename, 'w') as f:
            f.write(volumes)
            f.write('\n')
    except:
        print("Failed to save volume names that failed to update to file."
                "Please record them for future use.")


def make_policies_dir(datastore_path):
    """
    Create the policies dir if it doesn't exist and return the path.
    This function assumes that datastore_path is a VSAN datastore,
    although it won't fail if it isn't.
    """
    policies_dir = os.path.join(datastore_path, 'policies')
    try:
        os.mkdir(policies_dir)
    except OSError:
        pass
    return policies_dir


def create_policy_file(filename, content):
    """
    Create a storage policy file in filename. Returns an error string on
    failure and None on success.
    """
    try:
        with open(filename, 'w') as f:
            f.write(content)
            f.write('\n')
    except:
        msg = 'Error: Failed to open {0} for writing'.format(filename)
        logging.exception(msg)
        if os.path.isfile(filename):
            os.remove(filename)
        return msg

    return None


def delete(name):
    """
    Remove a given policy. If the policy does not exist return an error string,
    otherwise return None
    """
    path = vsan_info.get_vsan_dockvols_path()
    if not path:
        return ERROR_NO_VSAN_DATASTORE

    vmdk = policy_in_use(path, name)
    if vmdk:
        return 'Cannot remove. Policy is in use by {0}'.format(vmdk)
    try:
        os.remove(policy_path(name))
    except:
        logging.exception("Failed to remove %s policy file", name)
        return 'Policy {0} does not exist'.format(name)

    return None


def get_policies():
    """ Return a dict of all VSAN policy names to policy content. """
    policies = {}
    path = vsan_info.get_vsan_dockvols_path()
    if not path:
        return {}

    path = make_policies_dir(path)
    for name in os.listdir(path):
        with open(os.path.join(path, name)) as f:
            content = f.read()
        policies[name] = content
    return policies

def get_policy_content(policy_name):
    """ Return the content for a given policy. """
    if not policy_exists(policy_name):
        logging.warning("Policy %s does not exist", policy_name)
        return None
    with open(policy_path(policy_name)) as f:
        return f.read()

def set_policy_by_name(vmdk_path, policy_name):
    """ Set policy for a given volume. """
    content = get_policy_content(policy_name)
    if not content:
        return 'Error: {0} does not exist'.format(policy_name)
    return vsan_info.set_policy(vmdk_path, content)

def list_volumes_and_policies():
    """ Return a list of vmdks and the policies in use"""
    vmdks_and_policies = []
    path = vsan_info.get_vsan_dockvols_path()
    if not path:
        return []

    for volume in vmdk_utils.get_volumes("*"):
        logging.debug("volume data is %s", volume)
        policy = kv_get_vsan_policy_name(os.path.join(volume['path'], volume['filename']))
        vmdks_and_policies.append({'volume': volume['filename'], 'policy': policy,
                                    'path': volume['path']})
    return vmdks_and_policies


def policy_exists(name):
    """ Check if the policy file exists """
    return os.path.isfile(policy_path(name))


def policy_path(name):
    """
    Return the path to a given policy file or None if VSAN datastore doesn't
    exist
    """
    path = vsan_info.get_vsan_dockvols_path()
    if not path:
        return None

    return os.path.join(path, 'policies', name)


def kv_get_vsan_policy_name(path):
    """
    Take a path for a vmdk and return a policy name if it exists or None if it
    doesn't
    """
    try:
        return kv.getAll(path)[kv.VOL_OPTS][kv.VSAN_POLICY_NAME]
    except:
        return None


def policy_in_use(path, name):
    """
    Check if a policy is in use by a VMDK and return the name of the first VMDK
    using it if it is, None otherwise
    """
    for vmdk in list_volumes_and_policies():
        policy = vmdk['policy']
        if policy == name:
            return vmdk
    return None


def validate_vsan_policy_string(content):
    """
    Stub for a function that validates the syntax of a vsan policy string
    """
    return True
