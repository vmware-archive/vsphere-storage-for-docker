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
import logging
import pyVim.connect

# datastores should not change during 'vmdkops_admin' run,
# so using global to avoid multiple scans of /vmfs/volumes
datastores = None

# we assume files smaller that that to be descriptor files
MAX_DESCR_SIZE = 5000


def get_datastores():
    """
    Returns a list of (name, url-name, dockvol_path), with an element per datastore
    where:
    'name' is datastore name (e.g. 'vsanDatastore') , 
    'url-name' is the last element of datastore URL (e.g. 'vsan:572904f8c031435f-3513e0db551fcc82')
    'dockvol-path; is a full path to 'dockvols' folder on datastore 
    """

    global datastores
    if datastores != None:
        return datastores

    si = pyVim.connect.Connect()
    #  We are connected to ESX so childEntity[0] is current DC/Host
    ds_objects = \
      si.content.rootFolder.childEntity[0].datastoreFolder.childEntity
    datastores = [(d.info.name,
                   os.path.split(d.info.url)[1],
                   os.path.join(d.info.url, 'dockvols'))
                  for d in ds_objects]
    pyVim.connect.Disconnect(si)

    return datastores

def get_volumes():
    """ Return dicts of docker volumes, their datastore and their paths """
    volumes = []
    for (datastore, url_name, path) in get_datastores():
        for file_name in list_vmdks(path):
            volumes.append({'path': path,
                            'filename': file_name,
                            'datastore': datastore})
    return volumes


def get_vmdk_path(path, vol_name):
    """ forms full path as <path-to-volumes>/<volname>.vmdk"""
    return os.path.join(path, "{0}.vmdk".format(vol_name))


def list_vmdks(path, volname="", show_snapshots=False):
    """ Return a list of VMDKs in a given path. Filters out non-descriptor
    files.

    Params:
    path -  where the VMDKs are looked for
    volname - if passed, only files related to this VMDKs will be returned. Useful when
            doing volume snapshot inspect
    show_snapshots - if set to True, all VMDKs (including delta files) will be returned
    """

    # dockvols may not exists on a datastore
    if not os.path.exists(path):
        return []

    glob_pattern = "{0}/{1}*.vmdk".format(path, volname)
    vmdks = [os.path.basename(f) for f in glob.glob(glob_pattern)
            if vmdk_is_a_descriptor(os.path.join(path, f))]

    # For hiding snapshots we rely on volume names NOT having a '-' symbol,
    # so all files with '-' must be delta disks, digest and the likes, and
    # need to be hidden from further processing.
    # see vmdk_ops.py:parse_vol_name() which enforces the "no -" rule.
    if not show_snapshots:
        vmdks = [f for f in vmdks if f.find('-') == -1]
    
    return vmdks


def vmdk_is_a_descriptor(filepath):
    """
    Is the file a vmdk descriptor file?  We assume any file that ends in .vmdk
    and has a size less than MAX_DESCR_SIZE is a desciptor file.
    """
    if filepath.endswith('.vmdk') and os.stat(filepath).st_size < MAX_DESCR_SIZE:
       return True

    return False


def strip_vmdk_extension(filename):
    """ Remove the .vmdk file extension from a string """
    return filename.replace(".vmdk", "")
