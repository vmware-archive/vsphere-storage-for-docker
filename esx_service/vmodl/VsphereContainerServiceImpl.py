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

TENANT_NAME_MAX_LEN = 64
TENANT_DESC_MAX_LEN = 256

class TenantManagerImpl(vim.vcs.TenantManager):
    '''Implementation of VCS TenantManager'''

    def __init__(self, moId):
        vim.vcs.TenantManager.__init__(self, moId)

    def CreateTenant(self, name, description=None):
        logging.info("Creating a tenant: name=%s, description=%s", name, description)

        self.check_create_tenant_parameters(name, description)

        # Create the tenant in the database
        error_info, tenant = auth_api._tenant_create(name, description);
        if error_info:
            logging.error("Failed to create tenant: %s", error_info.msg)

            if error_info.code == ErrorCode.TENANT_ALREADY_EXIST:
                raise vim.fault.AlreadyExists(name="name")
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Successfully created a tenant: name=%s, description=%s", name, description)
        return self.map_tenant(tenant)

    def RemoveTenant(self, name, remove_volumes=False):
        logging.info("Removing a tenant: name=%s, remove_volumes=%s", name, remove_volumes)

        error_info = auth_api._tenant_rm(name, remove_volumes)
        if error_info:
            logging.error("Failed to remove tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Successfully removed tenant: name=%s", name)

    def GetTenants(self, name=None):
        logging.info("Retrieving tenant(s): name=%s", name)

        error_info, tenant_list = auth_api._tenant_ls(name)
        if error_info:
            logging.error("Failed to retrieve tenant(s): %s", error_info.msg)
            raise vim.fault.VcsFault(msg=error_info.msg);

        result = []
        for tenant in tenant_list:
            result.append(self.map_tenant(tenant))

        logging.info("Successfully retrieved tenant(s): name=%s", name)
        return result

    def UpdateTenant(self, name, new_name=None, description=None, default_datastore=None):
        logging.info("Updating tenant: name=%s, new_name=%s, description=%s, default_datastore=%s",
            name, new_name, description, default_datastore)

        self.check_update_tenant_parameters(name, new_name, description)

        error_info = auth_api._tenant_update(name, new_name, description, default_datastore)
        if error_info:
            logging.error("Failed to update tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            elif error_info.code == ErrorCode.TENANT_ALREADY_EXIST:
                raise vim.fault.AlreadyExists(name="new_name")
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Successfully updated tenant: name=%s, new_name=%s, description=%s, default_datastore=%s",
            name, new_name, description, default_datastore)

    def AddVMs(self, tenant, vms):
        if len(vms) == 0:
            logging.error("Adding VMs: the VM list is empty")
            raise vmodl.fault.InvalidArgument("VM list is empty")

        logging.info("Adding VMs: %s to tenant: %s", vms, tenant.name)

        error_info = auth_api._tenant_vm_add(tenant.name, vms)
        if error_info:
            logging.error("Failed to add VMs to tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            elif error_info.code == ErrorCode.VM_ALREADY_IN_TENANT:
                raise vim.fault.AlreadyExists(name="vms")
            elif error_info.code == ErrorCode.VM_NOT_FOUND:
                raise vmodl.fault.InvalidArgument(invalidProperty="vms")
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Succssfully added VMs: %s to tenant: %s", vms, tenant.name)

    def RemoveVMs(self, tenant, vms):
        if len(vms) == 0:
            logging.error("Remove VMs: the VM list is empty")
            raise vmodl.fault.InvalidArgument("VM list is empty")

        logging.info("Removing VMs: %s from tenant: %s", vms, tenant.name)

        error_info = auth_api._tenant_vm_rm(tenant.name, vms)
        if error_info:
            logging.error("Failed to remove VMs from tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            elif error_info.code == ErrorCode.VM_NOT_FOUND or error_info.code == ErrorCode.VM_NOT_IN_TENANT:
                raise vmodl.fault.InvalidArgument(invalidProperty="vms")
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Succssfully removed VMs: %s from tenant: %s", vms, tenant.name)

    def ReplaceVMs(self, tenant, vms):
        if len(vms) == 0:
            logging.error("Replace VMs: the VM list is empty")
            raise vmodl.fault.InvalidArgument("VM list is empty")

        logging.info("Replacing VMs for tenant: %s", tenant.name)
        logging.info("Existing VMs: %s", tenant.vms)
        logging.info("New VMs: %s", vms)

        error_info = auth_api._tenant_vm_replace(tenant.name, vms)
        if error_info:
            logging.error("Failed to replace VMs for tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            elif error_info.code == ErrorCode.VM_NOT_FOUND:
                raise vmodl.fault.InvalidArgument(invalidProperty="vms")
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Succssfully replaced VMs for tenant: %s", tenant.name)

    def AddPrivilege(self, tenant, privilege, default_datastore=False):
        logging.info("Adding privilege: %s to tenant: %s, default_datastore=%s",
            privilege, tenant.name, default_datastore)

        error_info = auth_api._tenant_access_add(name=tenant.name,
                                                 datastore=privilege.datastore,
                                                 allow_create=privilege.allow_create,
                                                 volume_maxsize_in_MB=privilege.volume_max_size,
                                                 volume_totalsize_in_MB=privilege.volume_total_size,
                                                 default_datastore=default_datastore)
        if error_info:
            logging.error("Failed to add privilege to tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            elif error_info.code == ErrorCode.PRIVILEGE_ALREADY_EXIST:
                raise vim.fault.AlreadyExists(name="privilege")
            elif error_info.code == ErrorCode.DS_NOT_EXIST or error_info.code == ErrorCode.PRIVILEGE_INVALID_VOLUME_SIZE:
                raise vmodl.fault.InvalidArgument(msg=error_info.msg)
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Succssfully added privilege to tenant: %s", tenant.name)

    def UpdatePrivilege(self, tenant, datastore, allow_create, volume_max_size, volume_total_size):
        logging.info("Updating privilege (datastore=%s) for tenant: %s", datastore, tenant.name)

        error_info = auth_api._tenant_access_set(name=tenant.name,
                                                 datastore=datastore,
                                                 allow_create=allow_create,
                                                 volume_maxsize_in_MB=volume_max_size,
                                                 volume_totalsize_in_MB=volume_total_size)
        if error_info:
            logging.error("Failed to update privilege for tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            elif error_info.code == ErrorCode.DS_NOT_EXIST or error_info.code == ErrorCode.PRIVILEGE_NOT_FOUND:
                raise vmodl.fault.InvalidArgument(invalidProperty="datastore")
            elif error_info.code == ErrorCode.PRIVILEGE_INVALID_VOLUME_SIZE:
                raise vmodl.fault.InvalidArgument(msg=error_info.msg)
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Succssfully updated privilege for tenant: %s", tenant.name)

    def RemovePrivilege(self, tenant, datastore):
        logging.info("Removing privilege (datastore=%s) from tenant: %s", datastore, tenant.name)

        error_info = auth_api._tenant_access_rm(tenant.name, datastore)
        if error_info:
            logging.error("Failed to remove privilege from tenant: %s", error_info.msg)
            if error_info.code == ErrorCode.TENANT_NOT_EXIST:
                raise vim.fault.NotFound(msg=error_info.msg)
            elif error_info.code == ErrorCode.DS_NOT_EXIST or error_info.code == ErrorCode.PRIVILEGE_NOT_FOUND:
                raise vmodl.fault.InvalidArgument(invalidProperty="datastore")
            else:
                raise vim.fault.VcsFault(msg=error_info.msg)

        logging.info("Succssfully removed privilege (datastore=%s) from tenant: %s", datastore, tenant.name)

    def map_tenant(self, tenant):
        """
        Map a DockerVolumeTenant instance returned by auth_api to VMODL vim.vcs.Tenant instance
        """

        if tenant == None:
            return None

        result = vim.vcs.Tenant()

        # Populate basic tenant info
        result.id = tenant.id
        result.name = tenant.name
        result.description = tenant.description

        # Populate default datastore
        if tenant.default_datastore_url:
            result.default_datastore = vmdk_utils.get_datastore_name(tenant.default_datastore_url)
            if result.default_datastore is None:
                return None

        # Populate associated VMs
        if tenant.vms:
            for vm_id in tenant.vms:
                vm_name = vmdk_utils.get_vm_name_by_uuid(vm_id)
                result.vms.append(vm_name)

        # Populate associated privileges
        if tenant.privileges:
            for privilege in tenant.privileges:
                result.privileges.append(self.map_privilege(privilege))

        return result

    def map_privilege(self, privilege):
        """
        Map a DatastoreAccessPrivilege instance returned by auth_api to VMODL vim.vcs.storage.DatastoreAccessPrivilege instance
        """

        if privilege == None:
            return None

        result = vim.vcs.storage.DatastoreAccessPrivilege()
        result.datastore =  vmdk_utils.get_datastore_name(privilege.datastore_url)
        if result.datastore is None:
            return None

        result.allow_create = privilege.allow_create
        result.volume_max_size = privilege.max_volume_size
        result.volume_total_size = privilege.usage_quota

        return result

    def check_create_tenant_parameters(self, name, description):
        if len(name) == 0:
            error_msg = error_code.error_code_to_message[ErrorCode.VMODL_TENANT_NAME_EMPTY]
            logging.error("Failed to create tenant: %s", error_msg)
            raise vmodl.fault.InvalidArgument(invalidProperty="name")

        if len(name) > TENANT_NAME_MAX_LEN:
            error_msg = error_code.error_code_to_message[ErrorCode.VMODL_TENANT_NAME_TOO_LONG].format(name);
            logging.error("Failed to create tenant: %s", error_msg)
            raise vmodl.fault.InvalidArgument(invalidProperty="name")

        if description and len(description) > TENANT_DESC_MAX_LEN:
            error_msg = error_code.error_code_to_message[ErrorCode.VMODL_TENANT_DESC_TOO_LONG].format(name);
            logging.error("Failed to create tenant: %s", error_msg)
            raise vmodl.fault.InvalidArgument(invalidProperty="description")

    def check_update_tenant_parameters(self, name, new_name, description):
        if new_name == name:
            logging.error("Failed to update tenant: new_name is the same as name")
            raise vmodl.fault.InvalidArgument(invalidProperty="new_name")

        if len(new_name) == 0:
            error_msg = error_code.error_code_to_message[ErrorCode.VMODL_TENANT_NAME_EMPTY]
            logging.error("Failed to update tenant: %s", error_msg)
            raise vmodl.fault.InvalidArgument(invalidProperty="new_name")

        if len(new_name) > TENANT_NAME_MAX_LEN:
            error_msg = error_code.error_code_to_message[ErrorCode.VMODL_TENANT_NAME_TOO_LONG].format(name);
            logging.error("Failed to update tenant: %s", error_msg)
            raise vmodl.fault.InvalidArgument(invalidProperty="new_name")

        if description and len(description) > TENANT_DESC_MAX_LEN:
            error_msg = error_code.error_code_to_message[ErrorCode.VMODL_TENANT_DESC_TOO_LONG].format(name);
            logging.error("Failed to update tenant: %s", error_msg)
            raise vmodl.fault.InvalidArgument(invalidProperty="description")

class VsphereContainerServiceImpl(vim.vcs.VsphereContainerService):
    '''Implementation of Vsphere Container Serivce'''

    def __init__(self, moId):
        vim.vcs.VsphereContainerService.__init__(self, moId)

    def GetTenantManager(self):
        return GetMoManager().LookupObject("vcs-tenant-manager")

GetMoManager().RegisterObjects([VsphereContainerServiceImpl("vsphere-container-service"),
                                TenantManagerImpl("vcs-tenant-manager")])
