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

from pyVim import vmconfig
from pyVmomi import vim
import pyVim
from pyVim.invt import GetVmFolder, FindChild
from error_code import *

import threadutils
import vmdk_ops
import auth_data_const
import auth
import auth_api
import log_config
from error_code import *


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

# lsof command
LSOF_CMD = "/bin/vmkvsitools lsof"

# Number of times and sleep time to retry on IOError EBUSY
VMDK_RETRY_COUNT = 5
VMDK_RETRY_SLEEP = 1

# root for all the volumes
VOLUME_ROOT = "/vmfs/volumes/"

# For managing resource locks.
lockManager = threadutils.LockManager()


def init_datastoreCache(force=False):
    """
    Initializes the datastore cache with the list of datastores accessible
    from local ESX host. force=True will force it to ignore current cache
    and force init
    """
    with lockManager.get_lock("init_datastoreCache"):
        global datastores
        logging.debug("init_datastoreCache:  %s", datastores)
        if datastores and not force:
            return

        si = vmdk_ops.get_si()

        #  We are connected to ESX so childEntity[0] is current DC/Host
        ds_objects = si.content.rootFolder.childEntity[0].datastoreFolder.childEntity
        tmp_ds = []

        for datastore in ds_objects:
            dockvols_path, err = vmdk_ops.get_vol_path(datastore.info.name)
            if err:
                logging.error(" datastore %s is being ignored as the dockvol path can't be created on it", datastore.info.name)
                continue
            tmp_ds.append((datastore.info.name,
                           datastore.info.url,
                           dockvols_path))
        datastores = tmp_ds


def validate_datastore(datastore):
    """
    Checks if the datastore is part of datastoreCache.
    If not it will update the datastore cache and check if datastore
    is a part of the updated cache.
    """
    init_datastoreCache()
    if datastore in [i[0] for i in datastores]:
        return True
    else:
        init_datastoreCache(force=True)
        if datastore in [i[0] for i in datastores]:
            return True
    return False


def get_datastores():
    """
    Returns a list of (name, url, dockvol_path), with an element per datastore
    where:
    'name' is datastore name (e.g. 'vsanDatastore') ,
    'url' is datastore URL (e.g. '/vmfs/volumes/vsan:572904f8c031435f-3513e0db551fcc82')
    'dockvol-path; is a full path to 'dockvols' folder on datastore
    """
    init_datastoreCache()
    return datastores


def get_volumes(tenant_re):
    """ Return dicts of docker volumes, their datastore and their paths
    """
    # Assume we have two tenants "tenant1" and "tenant2"
    # volumes for "tenant1" are in /vmfs/volumes/datastore1/dockervol/tenant1
    # volumes for "tenant2" are in /vmfs/volumes/datastore1/dockervol/tenant2
    # volumes does not belongs to any tenants are under /vmfs/volumes/dockervol
    # tenant_re = None : only return volumes which do not belong to a tenant
    # tenant_re = "tenant1" : only return volumes which belongs to tenant1
    # tenant_re = "tenant*" : return volumes which belong to tenant1 or tenant2
    # tenant_re = "*" : return all volumes under /vmfs/volumes/datastore1/dockervol
    logging.debug("get_volumes: tenant_pattern(%s)", tenant_re)
    volumes = []
    for (datastore, url, path) in get_datastores():
        logging.debug("get_volumes: %s %s %s", datastore, url, path)
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
                #  root = /vmfs/volumes/datastore1/dockervol/tenant1_uuid
                #  path = /vmfs/volumes/datastore1/dockervol
                #  sub_dir get the string "/tenant1_uuid"
                #  sub_dir_name is "tenant1_uuid"
                #  call get_tenant_name with "tenant1_uuid" to find corresponding
                #  tenant_name which will be used to match
                #  pattern specified by tenant_re
                logging.debug("get_volumes: path=%s root=%s", path, root)
                sub_dir = root.replace(path, "")
                sub_dir_name = sub_dir[1:]
                # sub_dir_name is the tenant uuid
                error_info, tenant_name = auth_api.get_tenant_name(sub_dir_name)
                if not error_info:
                    logging.debug("get_volumes: path=%s root=%s sub_dir_name=%s tenant_name=%s",
                                  path, root, sub_dir_name, tenant_name)
                    if fnmatch.fnmatch(tenant_name, tenant_re):
                        for file_name in list_vmdks(root):
                            volumes.append({'path': root,
                                            'filename': file_name,
                                            'datastore': datastore,
                                            'tenant': tenant_name})
                else:
                    # cannot find this tenant, this tenant was removed
                    # mark those volumes created by "orphan" tenant
                    logging.debug("get_volumes: cannot find tenant_name for tenant_uuid=%s", sub_dir_name)
                    logging.debug("get_volumes: path=%s root=%s sub_dir_name=%s",
                                path, root, sub_dir_name)

                    # return orphan volumes only in case when volumes from any tenants are asked
                    if tenant_re == "*":
                        for file_name in list_vmdks(root):
                            volumes.append({'path': root,
                                            'filename': file_name,
                                            'datastore': datastore,
                                            'tenant' : auth_data_const.ORPHAN_TENANT})
    logging.debug("volumes %s", volumes)
    return volumes


def get_vmdk_path(path, vol_name):
    """
    If the volume-related VMDK exists, returns full path to the latest
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


def get_volname_from_vmdk_path(vmdk_path):
    """Returns the volume name from a full vmdk path.
    """
    match = re.search(DATASTORE_PATH_REGEXP, vmdk_path)
    _, path = match.groups()
    vmdk = path.split("/")[-1]
    return strip_vmdk_extension(vmdk)


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
        expr = re.compile(SNAP_VMDK_REGEXP)
        vmdks = [f for f in vmdks if not expr.match(f)]
    logging.debug("vmdks %s", vmdks)
    return vmdks


def vmdk_is_a_descriptor(path, file_name):
    """
    Is the file a vmdk descriptor file?  We assume any file that ends in .vmdk,
    does not have -delta or -flat or -digest or -ctk at the end of filename,
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
        vm = FindChild(GetVmFolder(), vm_name)
        return vm.config.uuid
    except:
        return None


def get_vm_name_by_uuid(vm_uuid):
    """ Returns vm_name for given vm_uuid, or None """
    si = vmdk_ops.get_si()
    try:
        return vmdk_ops.vm_uuid2name(vm_uuid)
    except:
        return None


def get_vm_config_path(vm_name):
    """Returns vm_uuid for given vm_name, or None """
    si = vmdk_ops.get_si()
    try:
        vm = FindChild(GetVmFolder(), vm_name)
        config_path = vm.summary.config.vmPathName
    except:
        return None

    # config path has the format like this "[datastore1] test_vm1/test_vm1/test_vm1.vmx"
    datastore, path = config_path.split()
    datastore = datastore[1:-1]
    datastore_path = os.path.join("/vmfs/volumes/", datastore)
    # datastore_path has the format like this /vmfs/volumes/datastore_name
    vm_config_path = os.path.join(datastore_path, path)
    return vm_config_path

def get_attached_volume_path(vm, volname, datastore):
    """
    Returns full path for docker volume "volname", residing on "datastore" and attached to "VM"
    Files a warning and returns None if the volume is not attached
    """

    # Find the attached disk with backing matching "[datastore] dockvols/[.*]/volname[-ddddddd]?.vmdk"
    # SInce we don't know the vmgroup (the path after dockvols), we'll just pick the first match (and yell if
    # there is more that one match)
    # yes, it is super redundant - we will find VM, scan disks and find a matching one here, then return the path
    # and it will likely be used to do the same steps - find VM, scan the disks, etc.. It's a hack and it's a corner
    # case, so we'll live with this

    # Note that if VM is moved to a different vmgroup in flight, we may fail here and it's fine.
    # Note that if ther is a volume with the same name in 2 different vmgroup folders and both are attached
    # and VM is moved between the groups we may end up returning  the wrong volume but it's less

    if not datastore:
        # we rely on datastore always being a part of volume name passed to detach.
        # if this contract breaks, or we are called from somewhere else - bail out
        logging.error("get_attached_volume_path internal error - empty datastore")
        return None

    # look for '[datastore] dockvols/tenant/volume.vmdk' name
    # and account for delta disks (e.g. volume-000001.vmdk)
    prog = re.compile('\[%s\] %s/[^/]+/%s(-[0-9]{6})?\.vmdk$' %
                      (datastore, vmdk_ops.DOCK_VOLS_DIR, volname))
    attached = [d for d in vm.config.hardware.device  \
                    if isinstance(d, vim.VirtualDisk) and \
                       isinstance(d.backing, vim.VirtualDisk.FlatVer2BackingInfo) and \
                       prog.match(d.backing.fileName)]
    if len(attached) == 0:
        logging.error("Can't find device attached to '%s' for volume '%s' on [%s].",
                      vm.config.name, volname, datastore)
        return None
    if len(attached) > 1:
        logging.warning("More than 1  device attached to '%s' for volume '%s' on [%s].",
                        vm.config.name, volname, datastore)
    path = find_dvs_volume(attached[0])
    logging.warning("Found path: %s", path)
    return path

def find_dvs_volume(dev):
    """
    If the @param dev (type is vim.vm.device) a vDVS managed volume, return its vmdk path
    """
    # if device is not a virtual disk, skip this device
    if type(dev) != vim.vm.device.VirtualDisk:
        return False

    # Filename format is as follows:
    # "[<datastore name>] <parent-directory>/tenant/<vmdk-descriptor-name>"
    # Trim the datastore name and keep disk path.
    datastore_name, disk_path = dev.backing.fileName.rsplit("]", 1)
    logging.info("backing disk name is %s", disk_path)
    # name formatting to remove unwanted characters
    datastore_name = datastore_name[1:]
    disk_path = disk_path.lstrip()

    # find the dockvols dir on current datastore and resolve symlinks if any
    dvol_dir_path = os.path.realpath(os.path.join(VOLUME_ROOT,
                                                  datastore_name, vmdk_ops.DOCK_VOLS_DIR))
    dvol_dir = os.path.basename(dvol_dir_path)

    if disk_path.startswith(dvol_dir):
        # returning the vmdk path for vDVS volume
        return os.path.join(VOLUME_ROOT, datastore_name, disk_path)

    return None

def check_volumes_mounted(vm_list):
    """
    Return error_info if any vm in @param vm_list have docker volume mounted
    """
    for vm_id, _ in vm_list:
        vm = vmdk_ops.findVmByUuid(vm_id)
        if vm:
            for d in vm.config.hardware.device:
                if find_dvs_volume(d):
                    error_info = generate_error_info(ErrorCode.VM_WITH_MOUNTED_VOLUMES,
                                                     vm.config.name)
                    return error_info
        else:
            error_info = generate_error_info(ErrorCode.VM_NOT_FOUND, vm_id)
            return error_info
    return None


def log_volume_lsof(vol_name):
    """Log volume open file descriptors"""
    rc, out = vmdk_ops.RunCommand(LSOF_CMD)
    if rc != 0:
        logging.error("Error running lsof for %s: %s", vol_name, out)
        return
    for line in out.splitlines():
        # Make sure we only match the lines pertaining to that volume files.
        if re.search(r".*/vmfs/volumes/.*{0}.*".format(vol_name), line):
            cartel, name, ftype, fd, desc = line.split()
            msg = "cartel={0}, name={1}, type={2}, fd={3}, desc={4}".format(
                cartel, name, ftype, fd, desc)
            logging.info("Volume open descriptor: %s", msg)


def get_datastore_objects():
    """ return all datastore objects """
    si = vmdk_ops.get_si()
    return si.content.rootFolder.childEntity[0].datastore


def get_datastore_url(datastore_name):
    """ return datastore url for given datastore name """

    # Return datastore url for datastore name "_VM_DS""
    if datastore_name == auth_data_const.VM_DS:
        return auth_data_const.VM_DS_URL

    # Return datastore url for datastore name "_ALL_DS""
    if datastore_name == auth_data_const.ALL_DS:
        return auth_data_const.ALL_DS_URL

    # validate_datastore will refresh the cache if datastore_name is not in cache
    if not validate_datastore(datastore_name):
        return None

    # Query datastore URL from VIM API
    # get_datastores() return a list of tuple
    # each tuple has format like (datastore_name, datastore_url, dockvol_path)
    res = [d[1] for d in get_datastores() if d[0] == datastore_name]
    return res[0]


def get_datastore_name(datastore_url):
    """ return datastore name for given datastore url """
    # Return datastore name for datastore url "_VM_DS_URL""
    if datastore_url == auth_data_const.VM_DS_URL:
        return auth_data_const.VM_DS

    # Return datastore name for datastore url "_ALL_DS_URL""
    if datastore_url == auth_data_const.ALL_DS_URL:
        return auth_data_const.ALL_DS

    # Query datastore name from VIM API
    # get_datastores() return a list of tuple
    # each tuple has format like (datastore_name, datastore_url, dockvol_path)
    res = [d[0] for d in get_datastores() if d[1] == datastore_url]
    logging.debug("get_datastore_name: res=%s", res)
    return res[0] if res else None

def get_datastore_url_from_config_path(config_path):
    """Returns datastore url in config_path """
    # path can be /vmfs/volumes/<datastore_url_name>/...
    # or /vmfs/volumes/datastore_name/...
    # so extract datastore_url_name:
    config_ds_url = os.path.join("/vmfs/volumes/",
                                 os.path.realpath(config_path).split("/")[3])
    logging.debug("get_datastore_url_from_config_path: config_path=%s config_ds_url=%s"
                  % (config_path, config_ds_url))
    return config_ds_url


def main():
    log_config.configure()

if __name__ == "__main__":
    main()
