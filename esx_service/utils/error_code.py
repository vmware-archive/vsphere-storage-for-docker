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
    # Tenant related error code end

    # VM related error code start
    VM_NOT_FOUND = 101
    REPLACE_VM_EMPTY = 102
    VM_ALREADY_IN_TENANT = 103
    VM_NOT_IN_TENANT = 104
    # VM related error code end

    # Privilege related error code start
    PRIVILEGE_NOT_FOUND = 201
    PRIVILEGE_ALREADY_EXIST = 202
    PRIVILEGE_INVALID_VOLUME_SIZE = 203
    # Privilege related error code end

    # DATASTORE related error code start
    DEFAULT_DS_NOT_SET = 301
    DS_NOT_EXIST = 302
    # DATASTORE related error code end

    # VMODL related error code start
    VMODL_TENANT_NAME_EMPTY = 401
    VMODL_TENANT_NAME_TOO_LONG =402
    VMODL_TENANT_DESC_TOO_LONG = 403
    # VMODL related error code end

    INTERNAL_ERROR = 501
    INVALID_ARGUMENT = 502
    VOLUME_NAME_INVALID = 503


error_code_to_message = {
    ErrorCode.VM_NOT_BELONG_TO_TENANT : "VM {0} does not belong to any tenant",
    ErrorCode.TENANT_NOT_EXIST : "Tenant {0} does not exist",
    ErrorCode.TENANT_ALREADY_EXIST : "Tenant {0} already exists",
    ErrorCode.TENANT_NAME_NOT_FOUND : "Cannot find tenant name for tenant with {0}",
    ErrorCode.TENANT_CREATE_FAILED : "Tenant {0} create failed with err: {1}",
    ErrorCode.TENANT_SET_ACCESS_PRIVILEGES_FAILED : "Tenant {0} set access privileges on datastore {1} failed with err: {2}",
    ErrorCode.TENANT_GET_FAILED : "Get Tenant {0} failed",
    ErrorCode.TENANT_NAME_INVALID : "Tenant name {0} is invalid, only {1} is allowed",

    ErrorCode.VM_NOT_FOUND : "Cannot find vm {0}",
    ErrorCode.REPLACE_VM_EMPTY : "Replace VM cannot be empty",
    ErrorCode.VM_ALREADY_IN_TENANT : "Some VMs in {0} have already been associated with tenant '{1}', cannot add them again",
    ErrorCode.VM_NOT_IN_TENANT : "Some VMs in {0} have not been associated with tenant '{1}', cannot remove them",

    ErrorCode.PRIVILEGE_NOT_FOUND : "No privilege exists for ({0}, {1})",
    ErrorCode.PRIVILEGE_ALREADY_EXIST : "Privilege for ({0}, {1}) already exists",
    ErrorCode.PRIVILEGE_INVALID_VOLUME_SIZE : "Volume max size {0}MB exceeds the total size {1}MB",

    ErrorCode.DEFAULT_DS_NOT_SET : "Default datastore is not set",
    ErrorCode.DS_NOT_EXIST : "Datastore {0} does not exist",

    ErrorCode.VMODL_TENANT_NAME_EMPTY : "Tenant name is empty",
    ErrorCode.VMODL_TENANT_NAME_TOO_LONG : "Tenant name exceeds 64 characters: {0}",
    ErrorCode.VMODL_TENANT_DESC_TOO_LONG : "Tenant description exceeds 256 characters: {0}",

    ErrorCode.INTERNAL_ERROR : "Internal Error({0})",
    ErrorCode.INVALID_ARGUMENT : "Invalid Argument({0})",
     ErrorCode.VOLUME_NAME_INVALID : "Volume name {0} is invalid, only {1} is allowed"
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


