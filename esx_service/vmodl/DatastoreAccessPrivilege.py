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
   Datastore of this privilege. A tenant who has this privilege will have access
   on this datastore.
   """
   )
   @Attribute(parent=_name, typ="vim.Datastore")
   def datastore(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates whether the tenant is allowed to create and delete volumes on
   this datastore. If it's false, the tenant is allowed to mount and unmount
   volumes only.
   """
   )
   @Attribute(parent=_name, typ="boolean")
   def allow_create(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Maximum size (in MiB) of one single volume. Zero means unlimited size.
   """
   )
   @Attribute(parent=_name, typ="int")
   def volume_max_size(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Total storage usage quota (in MiB) allowed on this datastore. Zero means unlimited quota.
   """
   )
   @Attribute(parent=_name, typ="int")
   def volume_total_size(self):
       pass

RegisterVmodlTypes()
