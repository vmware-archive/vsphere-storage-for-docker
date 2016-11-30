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

from VmodlModules import *

class DatastoreAccessPrivilege:
   _name = "vim.vcs.storage.DatastoreAccessPrivilege"

   @JavaDocs(parent=_name, docs =
   """
   This class encapsulates the datastore access privilege for a tenant.
   """
   )
   @DataType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Datastore URL
   """
   )
   @Attribute(parent=_name, typ="string")
   def datastore_url(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates whether the tenant is allowed to create (or delete) a volume on this datastore
   """
   )
   @Attribute(parent=_name, typ="boolean")
   def allow_create(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Max volume size allowed on this datastore
   """
   )
   @Attribute(parent=_name, typ="string")
   def volume_max_size_in_MiB(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Total storage usage allowed on this datastore
   """
   )
   @Attribute(parent=_name, typ="string")
   def volume_total_size_in_MiB(self):
       pass

RegisterVmodlTypes()
