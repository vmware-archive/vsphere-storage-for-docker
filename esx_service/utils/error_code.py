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

""" Definiton of error message """

# Tenant related error message
VM_NOT_BELONG_TO_TENANT = "VM {0} does not belong to any tenant"
TENANT_NOT_EXIST = "Tenant {0} does not exist"
TENANT_ALREADY_EXIST = "Tenant {0} already exists"
TENANT_NAME_NOT_FOUND = "Cannot find tenant name for tenant with {0}"
TENANT_CREATE_FAILED = "Tenant {0} create failed with err: {1}"
TENANT_SET_ACCESS_PRIVILEGES_FAILED = "Tenant {0} set access privileges on datastore {1} failed with err: {2}"

# VM related error message 
VM_NOT_FOUND = "Cannot find vm {0}"
