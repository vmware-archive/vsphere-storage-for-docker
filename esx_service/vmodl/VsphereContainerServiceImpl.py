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

"""
Copyright 2016 VMware, Inc.  All rights reserved. 
Licensed under the Apache License, Version 2.0 
http://www.apache.org/licenses/LICENSE-2.0
"""

import logging
import os
import os.path
import sys

from pyVmomi import Vim, vim, vmodl
from MoManager import GetMoManager

# Location of utils used by the plugin
TOP_DIR = "/usr/lib/vmware/vmdkops"
PY_LOC  = os.path.join(TOP_DIR, "Python")
PY_BIN  = os.path.join(TOP_DIR, "bin")

# vmdkops python utils are in PY_LOC, so insert to path ahead of other stuff
sys.path.insert(0, PY_LOC)
sys.path.insert(0, PY_BIN)

import error_code
from error_code import ErrorCode
import auth_api
import vmdk_utils

class TenantManagerImpl(vim.vcs.TenantManager):
    '''Implementation of VCS TenantManager'''

    def __init__(self, moId):
        vim.vcs.TenantManager.__init__(self, moId)

    def CreateTenant(self, name, description=None):
        logging.info("Creating a tenant: name=%s, description=%s", name, description)

        if len(name) == 0:
            error_info = ErrorCode.VMODL_TENANT_NAME_EMPTY.format(name);
            return error_info

        if len(name) > 64:
            error_info = ErrorCode.VMODL_TENANT_NAME_TOO_LONG.format(name);
            return error_info

        if description and len(description) > 256:
            error_info = ErrorCode.VMODL_TENANT_NAME_TOO_LONG.format(name);
            return error_info

        # Create the tenant in the database
        error_info, tenant = auth_api._tenant_create(name, description);
        if error_info:
            logging.error("Failed to create tenant: name=%s, description=%s", name, description)
            raise vim.fault.VcsFault(msg = error_info.msg)

        result = vim.vcs.Tenant()
        result.id = tenant.id
        result.name = name
        result.description = description

        logging.info("Successfully created a tenant: name=%s, description=%s", name, description)
        return result

    def RemoveTenant(self, name, remove_volumes=False):
        logging.info("Removing a tenant: name=%s, remove_volumes=%s", name, remove_volumes)

        error_info = auth_api._tenant_rm(name, remove_volumes)
        if error_info == ErrorCode.TENANT_NOT_EXIST:
            raise vim.fault.NotFound(error_info.format(name))

        if error_info:
            logging.error("Failed to remove tenant: name=%s", name)
            raise vim.fault.VcsFault(msg = error_info.msg)
        
        logging.info("Successfully removed tenant: name=%s", name)

    def GetTenants(self, name=None):
        logging.info("Retrieving tenant(s): name=%s", name)

        error_info, tenant_list = auth_api._tenant_ls(name)
        if error_info:
            logging.error("Failed to retrieve tenant(s): name=%s", name)
            raise vim.fault.VcsFault(msg = error_info.msg);

        result = []
        for t in tenant_list:
            tenant = vim.vcs.Tenant()
            tenant.id = t.id
            tenant.name = t.name
            tenant.description = t.description

            # Populate default datastore 
            if t.default_datastore_url:
                tenant.default_datastore = vmdk_utils.get_datastore_name(t.default_datastore_url)

            # Populate associated VMs
            if t.vms:
                for vm_id in t.vms:
                    vm_name = vmdk_utils.get_vm_name_by_uuid(vm_id)
                    tenant.vms.append(vm_name)

            # Populate associated privileges
            if t.privileges:
                for p in t.privileges:
                    dap = vim.vcs.storage.DatastoreAccessPrivilege()
                    dap.datastore =  vmdk_utils.get_datastore_name(p.datastore_url)
                    dap.allow_create = p.allow_create
                    dap.volume_max_size = p.max_volume_size
                    dap.volume_total_size = p.usage_quota
                    tenant.privileges.append(dap)

            result.append(tenant)

        logging.info("Successfully retrieved tenant(s): name=%s", name)
        return result
            
    def UpdateTenant(self, name, new_name=None, description=None, default_datastore=None):
        logging.info("Updating tenant: name=%s, new_name=%s, description=%s, default_datastore=%s",
            name, new_name, description, default_datastore)

        error_info = auth_api._tenant_update(name, new_name, description, default_datastore)
        if error_info == ErrorCode.TENANT_NOT_EXIST:
            raise vim.fault.NotFound(error_info.format(name))

        if error_info == ErrorCode.TENANT_ALREADY_EXIST:
            raise vim.fault.AlreadyExists(error_info.format(new_name))

        if error_info:
            logging.error("Failed to update tenant: name=%s, new_name=%s, description=%s, default_datastore=%s",
            name, new_name, description, default_datastore)
            raise vim.fault.VcsFault(msg = error_info.msg)
        
        logging.info("Successfully updated tenant: name=%s, new_name=%s, description=%s, default_datastore=%s",
            name, new_name, description, default_datastore)

    def AddVMs(self, tenant, vms):
        if len(vms) == 0:
            logging.warn("Adding VMs: the VM list is empty")
            return

        logging.info("Adding VMs: %s to tenant: %s", vms, tenant.name)

        error_info = auth_api._tenant_vm_add(tenant.name, vms)
        if error_info:
            logging.error("Failed to add VMs: %s to tenant: ", vms, tenant.name)
            raise vim.fault.VcsFault(msg = error_info.msg)

        logging.info("Succssfully added VMs: %s to tenant: %s", vms, tenant.name)
    
    def RemoveVMs(self, tenant, vms):
        if len(vms) == 0:
            logging.warn("Remove VMs: the VM list is empty")
            return

        logging.info("Removing VMs: %s from tenant: %s", vms, tenant.name)

        error_info = auth_api._tenant_vm_rm(tenant.name, vms)
        if error_info:
            logging.error("Failed to remove VMs: %s from tenant: ", vms, tenant.name)
            raise vim.fault.VcsFault(msg = error_info.msg)

        logging.info("Succssfully removed VMs: %s from tenant: %s", vms, tenant.name)
    
    def ReplaceVMs(self, tenant, vms):
        if len(vms) == 0:
            logging.warn("Replace VMs: the VM list is empty")
            return

        logging.info("Replacing VMs for tenant: %s", tenant.name)
        logging.info("Existing VMs: %s", tenant.vms)
        logging.info("New VMs: %s", vms)

        error_info = auth_api._tenant_vm_replace(tenant.name, vms)
        if error_info:
            logging.error("Failed to replace VMs for tenant: ", tenant.name)
            raise vim.fault.VcsFault(msg = error_info.msg)

        logging.info("Succssfully replaced VMs for tenant: %s", tenant.name)

    def AddPrivilege(self, tenant, privilege, default_datastore=False):
        logging.info("Adding privilege: %s to tenant: %s, default_datastore=%s",
            privilege, tenant.name, default_datastore)

        error_info = auth_api._tenant_access_add(name = tenant.name,
                                                 datastore = privilege.datastore,
                                                 allow_create = privilege.allow_create,
                                                 volume_maxsize_in_MB = privilege.volume_max_size,
                                                 volume_totalsize_in_MB = privilege.volume_total_size)
        if error_info:
            logging.error("Failed to add privilege to tenant: %s", tenant.name)
            raise vim.fault.VcsFault(msg = error_info.msg)

        logging.info("Succssfully added privilege to tenant: %s", tenant.name)

    def UpdatePrivilege(self, tenant, datastore, allow_create, volume_max_size, volume_total_size):
        logging.info("Updating privilege (datastore=%s) for tenant: %s", datastore, tenant.name)

        error_info = auth_api._tenant_access_set(name = tenant.name,
                                                 datastore = datastore,
                                                 allow_create = allow_create,
                                                 volume_maxsize_in_MB = volume_max_size,
                                                 volume_totalsize_in_MB = volume_total_size)
        if error_info:
            if error_info.code == ErrorCode.PRIVILEGE_NOT_FOUND:
                logging.error(error_info.msg)
                raise vim.fault.VcsFault(msg = error_info.msg)
            else:
                logging.error("Failed to update privilege for tenant: %s", tenant.name)
                raise vim.fault.VcsFault(msg = error_info.msg)

        logging.info("Succssfully updated privilege for tenant: %s", tenant.name)

    def RemovePrivilege(self, tenant, datastore):
        logging.info("Removing privilege (datastore=%s) from tenant: %s", datastore, tenant.name)

        error_info = auth_api._tenant_access_rm(tenant.name, datastore)
        if error_info:
            logging.error("Failed to remove privilege (datastore=%s) from tenant: %s", datastore, tenant.name)
            raise vim.fault.VcsFault(msg = error_info.msg)

        logging.info("Succssfully removed privilege (datastore=%s) from tenant: %s", datastore, tenant.name)

class VsphereContainerServiceImpl(vim.vcs.VsphereContainerService):
    '''Implementation of Vsphere Container Serivce'''

    def __init__(self, moId):
        vim.vcs.VsphereContainerService.__init__(self, moId)

    def GetTenantManager(self):
        return GetMoManager().LookupObject("vcs-tenant-manager")

GetMoManager().RegisterObjects([VsphereContainerServiceImpl("vsphere-container-service"),
                                TenantManagerImpl("vcs-tenant-manager")])
