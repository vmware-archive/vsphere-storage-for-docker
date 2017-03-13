## VM based authorization for docker volumes
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

""" Define string constant for column name for table in authorization DB"""

# column name in tenants table
COL_ID = 'id'
COL_NAME = 'name'
COL_DESCRIPTION = 'description'
COL_DEFAULT_DATASTORE_URL = 'default_datastore_url'

# column name in vms table
COL_VM_ID = 'vm_id'
COL_TENANT_ID = 'tenant_id'

# column name in privileges table
COL_DATASTORE_URL = 'datastore_url'
COL_ALLOW_CREATE = 'allow_create'
COL_MAX_VOLUME_SIZE = 'max_volume_size'
COL_USAGE_QUOTA = 'usage_quota'

# column name in volume table
COL_VOLUME_NAME = 'volume_name'
COL_VOLUME_SIZE = 'volume_size'

# default tenant constants
DEFAULT_TENANT = '_DEFAULT'
DEFAULT_TENANT_UUID = '11111111-1111-1111-1111-111111111111'
DEFAULT_TENANT_DESCR = "This is a default vm-group"
DEFAULT_DS  = '_DEFAULT'
DEFAULT_DS_URL = DEFAULT_DS + "_URL"
ORPHAN_TENANT = "_ORPHAN"
