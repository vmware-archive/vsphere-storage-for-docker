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
# limitations under the License

""" Provide APIs to tenant management """

import auth
import auth_data_const
import convert
import auth_data
import vmdk_utils
import error_code
import log_config
import logging

def get_auth_mgr():
    """ Get a connection to auth DB. """
    try:
        auth_mgr = auth.get_auth_mgr()
    except auth_data.DbConnectionError as e:
        error_info = "Failed to connect auth DB({0})".format(e)
        return error_info, None
    return None, auth_mgr

def get_tenant_from_db(name):
    """
        Get a tenant object with given name
        Return value:
        -- error_info: return None on success or error info on failure
        -- tenant: return tenant object on success or None on failure 
    """
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info, None

    error_info, tenant = auth_mgr.get_tenant(name)
    return error_info, tenant

def get_tenant_name(tenant_uuid):
    """
        Get tenant name with given tenant_uuid
        Return value:
        -- error_info: return None on success or error info on failure
        -- tenant_name: return tenant name on success or None on failure 
    """
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info, None

    error_info, tenant_name = auth_mgr.get_tenant_name(tenant_uuid)
    return error_info, tenant_name

def create_tenant_in_db(name, description, vms, privileges):
    """
        Create a tenant object in DB
        Return value:
        -- error_info: return None on success or error info on failure
        -- tenant: return tenant object on success or None on failure
    """
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info, None

    error_info, tenant = auth_mgr.create_tenant(name=name, 
                                                description=description,  
                                                vms=vms, 
                                                privileges=privileges)
    return error_info, tenant

def get_tenant_list_from_db(name=None):
    """
        List all tenants or tenant with the name specified
        Params:
        -- name: if "name" is specified, return a list of one tenant with name specified
        if "name" is not specified, return a list of all tenants
        Return value:
        -- error_info: return None on success or error info on failure
        -- tenant_list: return a list of tenant objects on success or None on failure
    """
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info, None

    if not name:
        error_info, tenant_list = auth_mgr.list_tenants()
    else:
        error_info, tenant = auth_mgr.get_tenant(name)
        if error_info:
            tenant_list = []
        else:
            tenant_list = [tenant]

    return error_info, tenant_list

def generate_tuple_from_vm_list(vm_list):
    """ Generate a list of (vm_uuid, vm_name) pair """
    if not vm_list:
        return None, [], []
    vms = []
    error_info = ""
    not_found_vms = []
    for vm_name in vm_list:
        vm_uuid = vmdk_utils.get_vm_uuid_by_name(vm_name)
        if not vm_uuid:
            err =  "Cannot find vm_uuid for vm {0} ".format(vm_name)
            if err:
                error_info = error_info + err
                not_found_vms.append(vm_name) 
        vms.append((vm_uuid, vm_name))
    
    if error_info:
        return error_info, vms, not_found_vms
    
    return None, vms, not_found_vms

def default_privileges():
     """ return a privilege object with default value """
     privileges = [{'datastore': '',
                    'allow_create': 0,
                    'max_volume_size': 0,
                    'usage_quota': 0}]
     return privileges

def set_privileges(allow_create, privileges, value):
    """ set or unset allow_create privileges based on input param  """
    privileges[auth_data_const.COL_ALLOW_CREATE] = value
        
def generate_privileges(datastore, allow_create, volume_maxsize, volume_totalsize):
    """ Generate privileges based on input params """
    logging.debug("generate_privileges: datastore=%s allow_create=%s"
                  "volume_maxsize=%s volume_totalsize=%s",
                  datastore, allow_create, volume_maxsize, volume_totalsize)
    privileges = default_privileges()[0]
    if datastore == auth.DEFAULT_DS:
        datastore_url = auth.DEFAULT_DS_URL
    else:
        datastore_url = vmdk_utils.get_datastore_url(datastore)
    privileges[auth_data_const.COL_DATASTORE_URL] = datastore_url

    if allow_create is True:
        set_privileges(allow_create, privileges, 1)

    if volume_maxsize:
        size_in_MB = convert.convert_to_MB(volume_maxsize)
        privileges[auth_data_const.COL_MAX_VOLUME_SIZE] = size_in_MB

    if volume_totalsize:
        size_in_MB = convert.convert_to_MB(volume_totalsize)
        privileges[auth_data_const.COL_USAGE_QUOTA] = size_in_MB
    
    logging.debug("generate_privileges: privileges=%s", privileges)
    return privileges

def modify_privileges(privileges, allow_create, volume_maxsize, volume_totalsize):
    """ Modify privileges based on input params """
    if allow_create is not None:
        if allow_create is True:
            set_privileges(allow_create, privileges, 1)
        else:
            set_privileges(allow_create, privileges, 0)

    if volume_maxsize:
        size_in_MB = convert.convert_to_MB(volume_maxsize)
        privileges[auth_data_const.COL_MAX_VOLUME_SIZE] = size_in_MB

    if volume_totalsize:
        size_in_MB = convert.convert_to_MB(volume_totalsize)
        privileges[auth_data_const.COL_USAGE_QUOTA] = size_in_MB

    return privileges

def generate_privileges_dict(privileges):
    # privileges is a list which is read from auth DB
    # it has the following format
    # (tenant_uuid, datastore_url, allow_create, max_volume_size, usage_quota)
    privileges_dict = {}
    privileges_dict[auth_data_const.COL_DATASTORE_URL] = privileges[1]
    privileges_dict[auth_data_const.COL_ALLOW_CREATE] = privileges[2]
    privileges_dict[auth_data_const.COL_MAX_VOLUME_SIZE] = privileges[3]
    privileges_dict[auth_data_const.COL_USAGE_QUOTA] = privileges[4]
    return privileges_dict

def get_default_datastore(name):
    """ 
        Get default_datastore for given tenant
        Return value:
        --- error_info: return None on success or error info on failure
        --- default_datastore: return name of default_datastore on success or None on failure 
    """
    logging.debug("get_default_datastore: for tenant with name=%s", name)
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info, None
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info, None
    
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info, None
    
    # if default_datastore is not set for this tenant, default_datastore will be None
    error_info, default_datastore = tenant.get_default_datastore(auth_mgr.conn)
    return error_info, default_datastore 


def _tenant_create(name, description="", vm_list=None, privileges=None):
    """ API to create a tenant """
    logging.debug("_tenant_create: name=%s description=%s vm_list=%s privileges=%s", name, description, vm_list, privileges)
    error_info, vms, not_found_vms = generate_tuple_from_vm_list(vm_list)
    if error_info:
        not_found_vm_list = ",".join(not_found_vms)
        logging.warning(error_code.VM_NOT_FOUND.format(not_found_vm_list))
        return error_info, None
        
    # if param "description" is not set by caller, the default value is empty string
    if not description:
        description = ""

    logging.debug("_tenant_create: vms=%s", vms)
    vms_uuid_list = [(vm_id) for (vm_id, vm_name) in vms]
    error_info, tenant = create_tenant_in_db(
                                             name=name, 
                                             description=description, 
                                             vms=vms_uuid_list, 
                                             privileges=privileges)
    if error_info:
        return error_info, None
    
    return None, tenant

def _tenant_update(name, new_name=None, description=None, default_datastore=None):
    """ API to update a tenant """
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info
    
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info
    if new_name:    
        error_info = tenant.set_name(auth_mgr.conn, name, new_name)
        if error_info:
            return error_info
    if description:
        error_info = tenant.set_description(auth_mgr.conn, description)
        if error_info:
            return error_info
    if default_datastore:
        datastore_url = vmdk_utils.get_datastore_url(default_datastore)
        error_info = tenant.set_default_datastore(auth_mgr.conn, datastore_url)
        if error_info:
            return error_info
    
    return None

def _tenant_rm(name, remove_volumes=False):
    """ API to remove a tenant """
    logging.debug("_tenant_rm: name=%s remove_volumes=%s", name, remove_volumes)
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info
    
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info

    error_info = auth_mgr.remove_tenant(tenant.id, remove_volumes)
    return error_info

def _tenant_ls(name=None):
    """ API to list all tenants """
    logging.debug("_tenant_ls: name=%s", name)
    error_info, tenant_list = get_tenant_list_from_db(name)
    return error_info, tenant_list

def _tenant_vm_add(name, vm_list):
    """ API to add vms for a tenant """
    logging.debug("_tenant_vm_add: name=%s vm_list=%s", name, vm_list)
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info

    error_info, vms, not_found_vms = generate_tuple_from_vm_list(vm_list)
    if error_info:
        return error_info
    
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info

    logging.debug("_tenant_vm_add: vms=%s", vms)
    vms_uuid_list = [(vm_id) for (vm_id, vm_name) in vms]
    error_info = tenant.add_vms(auth_mgr.conn, vms_uuid_list)
    return error_info

def _tenant_vm_rm(name, vm_list):
    """ API to remove vms for a tenant """
    logging.debug("_tenant_vm_rm: name=%s vm_list=%s", name, vm_list)
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info

    error_info, vms, not_found_vms = generate_tuple_from_vm_list(vm_list)
    
    if error_info:
        return error_info

    logging.debug("_tenant_vm_add: vms=%s", vms)
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info
      
    vms_uuid_list = [(vm_id) for (vm_id, vm_name) in vms]
    error_info = tenant.remove_vms(auth_mgr.conn, vms_uuid_list)
    return error_info


def _tenant_vm_ls(name):
    """ API to get vms for a tenant """
    logging.debug("_tenant_vm_ls: name=%s", name)
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info, None
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info, None
    # tenant.vms is a list of vm_uuid of vms which belong to this tenant
    return None, tenant.vms
    
def _tenant_access_add(name, datastore, allow_create=None, default_datastore=False, volume_maxsize=None, volume_totalsize=None):
    """ API to add datastore access for a tenant """
    logging.debug("_tenant_access_add: name=%s datastore=%s, allow_create=%s "
                  "volume_maxsize=%s volume_totalsize=%s")
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info

    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info

    privileges = generate_privileges(datastore=datastore, 
                                     allow_create=allow_create, 
                                     volume_maxsize=volume_maxsize, 
                                     volume_totalsize=volume_totalsize)
    logging.debug("_tenant_access_add: privileges=%s", privileges)
    
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info
    
    error_info = tenant.set_datastore_access_privileges(auth_mgr.conn, [privileges])
    if error_info:
        return error_info
    
    error_info, result = auth.get_row_from_privileges_table(auth_mgr.conn, tenant.id)
    # if len(result) == 1, which means "datastore"" is the first datastore for this tenant,
    # and should set this datastore to the "default_datastore" for this tenant
    if error_info:
        return error_info
    logging.debug("_tenant_access_add: get_row_from_privileges_table for tenant id=%s return %s",
                   tenant.id, result)
    if len(result) == 1 or default_datastore:
        if datastore == auth.DEFAULT_DS:
            datastore_url = auth.DEFAULT_DS_URL
        else:    
            datastore_url = vmdk_utils.get_datastore_url(datastore)
        error_info = tenant.set_default_datastore(auth_mgr.conn, datastore_url)
    
    return error_info

def _tenant_access_set(name, datastore, allow_create=None, volume_maxsize=None, volume_totalsize=None):
    """ API to modify datastore access for a tenant """
    logging.debug("_tenant_access_set: name=%s datastore=%s, allow_create=%s "
                  "volume_maxsize=%s volume_totalsize=%s")
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info

    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info
    
    datastore_url = vmdk_utils.get_datastore_url(datastore)
    logging.debug("_tenant_access_set: datastore_url=%s", datastore_url)
    privileges = [d for d in tenant.privileges if d[auth_data_const.COL_DATASTORE_URL] == datastore_url]
    
    if not privileges:
        error_info = "No privileges exist for ({0}, {1})".format(name, datastore)
        return error_info
    
    privileges_dict = generate_privileges_dict(privileges[0])
    logging.debug("_tenant_access_set: originial privileges_dict=%s", privileges_dict)
    privileges_dict = modify_privileges(privileges=privileges_dict, 
                                        allow_create=allow_create,
                                        volume_maxsize=volume_maxsize,
                                        volume_totalsize=volume_totalsize)
    logging.debug("_tenant_access_set: modified privileges_dict=%s", privileges_dict)

    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info

    error_info = tenant.set_datastore_access_privileges(auth_mgr.conn, [privileges_dict])
    return error_info

def _tenant_access_rm(name, datastore):
    """ API to remove datastore access for a tenant """
    logging.debug("_tenant_access_rm: name=%s datastore=%s", name, datastore)
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info
    
    error_info, auth_mgr = get_auth_mgr()
    if error_info:
        return error_info
    if datastore == auth.DEFAULT_DS:
        datastore_url = auth.DEFAULT_DS_URL
    else:
        datastore_url = vmdk_utils.get_datastore_url(datastore)
    logging.debug("_tenant_access_rm: datastore_url=%s", datastore_url)
    error_info = tenant.remove_datastore_access_privileges(auth_mgr.conn, datastore_url)
    if error_info:
        return error_info

    # get dafault_datastore, if default_datastore is the same as param "datastore"
    # need to set default_datastore_url to "" in tenants table    
    error_info, default_datastore = get_default_datastore(name)
    if error_info:
        return error_info
    
    if default_datastore == datastore:
        error_info  = tenant.set_default_datastore(auth_mgr.conn, "")

    return error_info

def _tenant_access_ls(name):
    """ Handle tenant access ls command """
    logging.debug("_tenant_access_ls: name=%s", name)
    error_info, tenant = get_tenant_from_db(name)
    if error_info:
        return error_info, None
    
    if not tenant:
        error_info = error_code.TENANT_NOT_EXIST.format(name)
        return error_info, None

    return None, tenant.privileges 
 