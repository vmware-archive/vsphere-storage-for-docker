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

# Utility functions for dealing with VMDKs and datastores

import os
import os.path
import glob
import re
import logging
import fnmatch
import vmdk_ops

# datastores should not change during 'vmdkops_admin' run,
# so using global to avoid multiple scans of /vmfs/volumes
datastores = None

# we assume files smaller that that to be descriptor files
MAX_DESCR_SIZE = 5000

# regexp for finding "snapshot" (aka delta disk) descriptor names
SNAP_NAME_REGEXP = r"^.*-[0-9]{6}$"        # used for names without .vmdk suffix
SNAP_VMDK_REGEXP = r"^.*-[0-9]{6}\.vmdk$"  # used for file names

# regexp for finding 'special' vmdk files (they are created by ESXi) 
SPECIAL_FILES_REGEXP = r"\A.*-(delta|ctk|digest|flat)\.vmdk$"

# glob expression to match end of 'delta' (aka snapshots) file names.
SNAP_SUFFIX_GLOB = "-[0-9][0-9][0-9][0-9][0-9][0-9].vmdk"

# regexp for finding datastore path "[datastore] path/to/file.vmdk" from full vmdk path
DATASTORE_PATH_REGEXP = r"^/vmfs/volumes/([^/]+)/(.*\.vmdk)$"

def init_datastoreCache():
    """
    Initializes the datastore cache with the list of datastores accessible from local ESX host.
    """
    global datastores
    logging.debug("init_datastoreCache: %s", datastores)

    si = vmdk_ops.get_si()

    #  We are connected to ESX so childEntity[0] is current DC/Host
    ds_objects = \
      si.content.rootFolder.childEntity[0].datastoreFolder.childEntity
    datastores = [(d.info.name,
                   os.path.split(d.info.url)[1],
                   os.path.join(d.info.url, 'dockvols'))
                  for d in ds_objects]

def validate_datastore(datastore):
    """
    Checks if the datastore is part of datastoreCache. 
    If not it will update the datastore cache and checks if datastore is part of the updated cache.
    """
    global datastores
    if datastores == None:
        init_datastoreCache()
    if datastore in [i[0] for i in datastores]:
        return True
    else:
        init_datastoreCache()
        if datastore in [i[0] for i in datastores]:
            return True
    return False

def get_datastores():
    """
    Returns a list of (name, url-name, dockvol_path), with an element per datastore
    where:
    'name' is datastore name (e.g. 'vsanDatastore') ,
    'url-name' is the last element of datastore URL (e.g. 'vsan:572904f8c031435f-3513e0db551fcc82')
    'dockvol-path; is a full path to 'dockvols' folder on datastore 
    """
    if datastores == None:
        init_datastoreCache()
    return datastores

def get_volumes(tenant_re):
    """ Return dicts of docker volumes, their datastore and their paths 
        
        
    """
    # Assume we have two tenants "tenant1" and "tenant2"
    # volumes belongs to "tenant1" are under /vmfs/volumes/datastore1/dockervol/tenant1 
    # volumes belongs to "tenant2" are under /vmfs/volumes/datastore1/dockervol/tenant2
    # volumes does not belongs to any tenants are under /vmfs/volumes/dockervol
    # tenant_re = None : only return volumes which does not belong to any tenant
    # tenant_re = "tenant1" : only return volumes which belongs to tenant1
    # tenant_re = "tenant*" : return volumes which belongs to tenant1 or tenant2
    # tenant_re = "*" : return all volumes under /vmfs/volumes/datastore1/dockervol
    logging.debug("get_volumes: tenant_pattern(%s)", tenant_re)
    volumes = []
    for (datastore, url_name, path) in get_datastores():
        logging.debug("get_volumes: %s %s %s", datastore, url_name, path)
        if not tenant_re:
            for file_name in list_vmdks(path):
                # path : docker_vol path
                volumes.append({'path': path,
                                'filename': file_name,
                                'datastore': datastore})
        else:
            for root, dirs, files in os.walk(path):
                # walkthough all files under docker_vol path
                # root is the current directory which is traversing
                #  root = /vmfs/volumes/datastore1/dockervol/tenant1
                #  path = /vmfs/volumes/datastore1/dockervol
                #  sub_dir get the string "/tenant1"
                #  sub_dir_name is "tenant1" which will be used to match
                #  pattern specified by tenant_re
                sub_dir = root.replace(path, "")
                sub_dir_name = sub_dir[1:]
                if fnmatch.fnmatch(sub_dir_name, tenant_re):
                    for file_name in list_vmdks(root):
                        volumes.append({'path': root,
                                        'filename': file_name,
                                        'datastore': datastore})
    logging.debug("volumes %s", volumes)
    return volumes


def get_vmdk_path(path, vol_name):
    """If the volume-related VMDK exists, returns full path to the latest
    VMDK disk in the disk chain, be it volume-NNNNNN.vmdk or volume.vmdk.
    If the disk does not exists, returns full path to the disk for create().
    """

    # Get a delta disk list, and if it's empty - return the full path for volume
    # VMDK base file.
    # Note: we rely on NEVER allowing '-NNNNNN' in end of a volume name and on
    # the fact that ESXi always creates deltadisks as <name>-NNNNNN.vmdk (N is a
    # digit, and there are exactly 6 digits there) for delta disks
    #
    # see vmdk_ops.py:parse_vol_name() which enforces the volume name rules.
    delta_disks = glob.glob("{0}/{1}{2}".format(path, vol_name, SNAP_SUFFIX_GLOB))
    if not delta_disks:
        return os.path.join(path, "{0}.vmdk".format(vol_name))

    # this funky code gets the name of the latest delta disk:
    latest = sorted([(vmdk, os.stat(vmdk).st_ctime) for vmdk in delta_disks], key=lambda d: d[1], reverse=True)[0][0]
    logging.debug("The latest delta disk is %s. All delta disks: %s", latest, delta_disks)
    return latest


def get_datastore_path(vmdk_path):
    """Returns a string datastore path "[datastore] path/to/file.vmdk"
    from a full vmdk path.
    """
    match = re.search(DATASTORE_PATH_REGEXP, vmdk_path)
    datastore, path = match.groups()
    return "[{0}] {1}".format(datastore, path)

def get_datastore_from_vmdk_path(vmdk_path):
    """Returns a string representing the datastore from a full vmdk path.
    """
    match = re.search(DATASTORE_PATH_REGEXP, vmdk_path)
    datastore, path = match.groups()
    return datastore

def list_vmdks(path, volname="", show_snapshots=False):
    """ Return a list of VMDKs in a given path. Filters out non-descriptor
    files and delta disks.

    Params:
    path -  where the VMDKs are looked for
    volname - if passed, only files related to this VMDKs will be returned. Useful when
            doing volume snapshot inspect
    show_snapshots - if set to True, all VMDKs (including delta files) will be returned
    """

    # dockvols may not exists on a datastore - this is normal.
    if not os.path.exists(path):
        return []
    logging.debug("list_vmdks: dockvol existed on datastore")
    vmdks = [f for f in os.listdir(path) if vmdk_is_a_descriptor(path, f)]
    if volname:
        vmdks = [f for f in vmdks if f.startswith(volname)]

    if not show_snapshots:
        expr =  re.compile(SNAP_VMDK_REGEXP)
        vmdks = [f for f in vmdks if not expr.match(f)]
    logging.debug("vmdks %s", vmdks)
    return vmdks


def vmdk_is_a_descriptor(path, file_name):
    """
    Is the file a vmdk descriptor file?  We assume any file that ends in .vmdk,
    does not have -delta or -flat or he likes at the end of filename, 
    and has a size less than MAX_DESCR_SIZE is a descriptor file.
    """

    name = file_name.lower()

    # filter out all files with wrong extention
    # also filter out -delta, -flat, -digest and -ctk VMDK files
    if not name.endswith('.vmdk') or re.match(SPECIAL_FILES_REGEXP, name):
        return False

    # Check the size. It's a cheap(ish) way to check for descriptor, 
    # without actually checking the file content and risking lock conflicts
    try:
        if os.stat(os.path.join(path, file_name)).st_size > MAX_DESCR_SIZE:
            return False
    except OSError:
        pass  # if file does not exist, assume it's small enough

    return True


def strip_vmdk_extension(filename):
    """ Remove the .vmdk file extension from a string """
    return filename.replace(".vmdk", "")

def get_vm_uuid_by_name(vm_name):
    """ Returns vm_uuid for given vm_name, or None """
    si = vmdk_ops.get_si()
    try:
        vm = [d for d in si.content.rootFolder.childEntity[0].vmFolder.childEntity if d.config.name == vm_name]
        return vm[0].config.uuid
    except:
        return None

def get_vm_config_path(vm_name):
    """Returns vm_uuid for given vm_name, or None """
    si = vmdk_ops.get_si()
    try:
        vm = [d for d in si.content.rootFolder.childEntity[0].vmFolder.childEntity if d.config.name == vm_name]
        config_path = vm[0].summary.config.vmPathName   
    except:
        return None
    
     # config path has the format like this "[datastore1] test_vm1/test_vm1/test_vm1.vmx"
    datastore, path = config_path.split()
    datastore = datastore[1:-1]
    datastore_path = os.path.join("/vmfs/volumes/", datastore)
    # datastore_path has the format like this /vmfs/volumes/datastore_name
    vm_config_path = os.path.join(datastore_path, path)
    return vm_config_path

def find_vm_by_name(vm_name):
    """ Return vm for given vm_name, or None """
    si = vmdk_ops.get_si()
    try:
        vm = [d for d in si.content.rootFolder.childEntity[0].vmFolder.childEntity 
                if d.config.name == vm_name]
        return vm[0]

    except:
        return None

