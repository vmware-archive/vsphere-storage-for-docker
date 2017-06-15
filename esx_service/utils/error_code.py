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

""" Definiton of error code and error message """
class ErrorCode:
    # Tenant related error code start
    VM_NOT_BELONG_TO_TENANT = 1
    TENANT_NOT_EXIST = 2
    TENANT_ALREADY_EXIST = 3
    TENANT_NAME_NOT_FOUND = 4
    TENANT_CREATE_FAILED = 5
    TENANT_SET_ACCESS_PRIVILEGES_FAILED = 6
    TENANT_GET_FAILED = 7
    TENANT_NAME_INVALID = 8
    TENANT_NOT_EMPTY = 9
    # Tenant related error code end

    # VM related error code start
    VM_NOT_FOUND = 101
    REPLACE_VM_EMPTY = 102
    VM_ALREADY_IN_TENANT = 103
    VM_NOT_IN_TENANT = 104
    VM_IN_ANOTHER_TENANT = 105
    VM_LIST_EMPTY = 106
    VM_DUPLICATE = 107
    VM_WITH_MOUNTED_VOLUMES = 108
    # VM related error code end

    # Privilege related error code start
    PRIVILEGE_NOT_FOUND = 201
    PRIVILEGE_ALREADY_EXIST = 202
    PRIVILEGE_INVALID_VOLUME_SIZE = 203
    PRIVILEGE_INVALID_ALLOW_CREATE_VALUE = 204
    PRIVILEGE_REMOVE_NOT_ALLOWED = 205

    # result returned from check_privileges_for_command function
    PRIVILEGE_NO_PRIVILEGE = 206
    PRIVILEGE_NO_MOUNT_PRIVILEGE = 207
    PRIVILEGE_NO_CREATE_PRIVILEGE = 208
    PRIVILEGE_NO_DELETE_PRIVILEGE = 209
    PRIVILEGE_MAX_VOL_EXCEED = 210
    PRIVILEGE_USAGE_QUOTA_EXCEED = 211
    PRIVILEGE_SET_TOTAL_VOLUME_SIZE_LIMIT_NOT_ALLOWED = 212
    # Privilege related error code end

    # DATASTORE related error code start
    DS_DEFAULT_NOT_SET = 301
    DS_NOT_EXIST = 302
    DS_DEFAULT_NAME_INVALID = 303
    DS_DEFAULT_SET_FAILED = 304
    DS_DEFAULT_CANNOT_USE_ALL_DS = 305
    # DATASTORE related error code end

    # VMODL related error code start
    VMODL_TENANT_NAME_EMPTY = 401
    VMODL_TENANT_NAME_TOO_LONG = 402
    VMODL_TENANT_DESC_TOO_LONG = 403
    # VMODL related error code end

    INTERNAL_ERROR = 501
    INVALID_ARGUMENT = 502
    VOLUME_NAME_INVALID = 503
    FEATURE_NOT_SUPPORTED = 504
    # Indicates that config init needed for a requested operation
    INIT_NEEDED = 505

    # Catch all for SQLite errors. Note that logging() will have extra  info
    SQLITE3_ERROR = 506

    # Volume option related error code start
    OPT_VOLUME_SIZE_INVALID  = 507
    # Volume option related error code end



error_code_to_message = {
    ErrorCode.VM_NOT_BELONG_TO_TENANT : "VM {0} does not belong to any vmgroup",
    ErrorCode.TENANT_NOT_EXIST : "Vmgroup {0} does not exist",
    ErrorCode.TENANT_ALREADY_EXIST : "Vmgroup {0} already exists",
    ErrorCode.TENANT_NAME_NOT_FOUND : "Cannot find vmgroup name for vmgroup with {0}",
    ErrorCode.TENANT_CREATE_FAILED : "Vmgroup {0} create failed with err: {1}",
    ErrorCode.TENANT_SET_ACCESS_PRIVILEGES_FAILED : "Vmgroup {0} set access privileges on datastore {1} failed with err: {2}",
    ErrorCode.TENANT_GET_FAILED : "Get vmgroup {0} failed",
    ErrorCode.TENANT_NAME_INVALID : "Vmgroup name {0} is invalid, only {1} is allowed",
    ErrorCode.TENANT_NOT_EMPTY : "Cannot delete non-empty vmgroup {0}. Remove VMs from the vmgroup before deleting it.",

    ErrorCode.VM_NOT_FOUND : "Cannot find vm {0}",
    ErrorCode.REPLACE_VM_EMPTY : "Replace VM cannot be empty",
    ErrorCode.VM_ALREADY_IN_TENANT : "VM '{0}' has already been associated with tenant '{1}', cannot add it again",
    ErrorCode.VM_NOT_IN_TENANT : "VM '{0}' has not been associated with tenant '{1}', cannot remove it",
    ErrorCode.VM_IN_ANOTHER_TENANT : "VM '{0}' has already been associated with tenant '{1}', can't add it",
    ErrorCode.VM_LIST_EMPTY : "VM list cannot be empty",
    ErrorCode.VM_DUPLICATE : "VMs {0} contain duplicates, they should be unique",
    ErrorCode.VM_WITH_MOUNTED_VOLUMES : "VM '{0}' has volumes mounted.",

    ErrorCode.PRIVILEGE_NOT_FOUND : "No access privilege exists for ({0}, {1})",
    ErrorCode.PRIVILEGE_ALREADY_EXIST : "Access privilege for ({0}, {1}) already exists",
    ErrorCode.PRIVILEGE_INVALID_VOLUME_SIZE : "Volume max size {0}MB exceeds the total size {1}MB",
    ErrorCode.PRIVILEGE_INVALID_ALLOW_CREATE_VALUE : "Invalid value {0} for allow-create option",
    ErrorCode.PRIVILEGE_REMOVE_NOT_ALLOWED : "Removing of access privilege to 'default_datastore' is not supported",
    ErrorCode.PRIVILEGE_NO_PRIVILEGE : "No access privilege exists",
    ErrorCode.PRIVILEGE_NO_MOUNT_PRIVILEGE : "No mount privilege",
    ErrorCode.PRIVILEGE_NO_CREATE_PRIVILEGE : "No create privilege",
    ErrorCode.PRIVILEGE_NO_DELETE_PRIVILEGE : "No delete privilege",
    ErrorCode.PRIVILEGE_MAX_VOL_EXCEED : "Volume size exceeds the max volume size limit",
    ErrorCode.PRIVILEGE_USAGE_QUOTA_EXCEED : "The total volume size exceeds the usage quota",
    ErrorCode.PRIVILEGE_SET_TOTAL_VOLUME_SIZE_LIMIT_NOT_ALLOWED : "Cannot set volume-totalsize for {}",

    ErrorCode.DS_DEFAULT_NOT_SET : "Default datastore is not set for vmgroup {}",
    ErrorCode.DS_NOT_EXIST : "Datastore {0} does not exist",
    ErrorCode.DS_DEFAULT_NAME_INVALID : "Default datastore name {} is invalid",
    ErrorCode.DS_DEFAULT_SET_FAILED : "Set default datastore for vmgroup {} failed ({})",
    ErrorCode.DS_DEFAULT_CANNOT_USE_ALL_DS : "Cannot use _ALL_DS as default datastore. Please use specific datastore name or _VM_DS special datastore",

    ErrorCode.VMODL_TENANT_NAME_EMPTY : "Vmgroup name is empty",
    ErrorCode.VMODL_TENANT_NAME_TOO_LONG : "Vmgroup name exceeds 64 characters: {0}",
    ErrorCode.VMODL_TENANT_DESC_TOO_LONG : "Vmgroup description exceeds 256 characters: {0}",

    ErrorCode.INTERNAL_ERROR : "Internal Error({0})",
    ErrorCode.INVALID_ARGUMENT : "Invalid Argument({0})",
    ErrorCode.VOLUME_NAME_INVALID : "Volume name {0} is invalid, only {1} is allowed",
    ErrorCode.FEATURE_NOT_SUPPORTED : "This feature is not supported for vmgroup {}.",
    ErrorCode.INIT_NEEDED: "Configuration not found. Please init configuration with 'vmdkops_admin.py config init'.",
    ErrorCode.SQLITE3_ERROR: "Sqlite3 error - see log for more info",

    ErrorCode.OPT_VOLUME_SIZE_INVALID : "Invalid volume size specified.",
}

class ErrorInfo:
    """ A class to abstract ErrorInfo object
        @Param code: error_code
        @Param msg: detailed error message
    """

    def __init__(self, code, msg):
        self.code = code
        self.msg = msg

def join_args(fmstr, *args):
    return fmstr.format(*args)

def generate_error_info(err_code, *params):
    """
        Return error_info object with given err_code and params
        @Param err_code: error_code
        @Param *params: varialbe number of params which are
         needed to construct error message
    """
    fmstr = error_code_to_message[err_code]
    err_msg = join_args(fmstr, *params)
    error_info = ErrorInfo(err_code, err_msg)
    return error_info


