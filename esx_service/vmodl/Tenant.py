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

from VmodlDecorators import ManagedType, EnumType, Method, \
   Return, RegisterVmodlTypes, F_OPTIONAL, Param, DataType, Attribute
from pyVmomi import Vmodl
from pyVmomi.VmomiSupport import newestVersions

try:
   from asyncVmodlEmitterLib import JavaDocs, Internal
except ImportError:
   pass
   def JavaDocs(parent, docs):
      def Decorate(f):
         return f
      return Decorate
   def Internal(parent):
      def Decorate(f):
         return f
      return Decorate


# _VERSION = newestVersions.Get("vim")
_VERSION = 'vim.version.version10'

# Vmodl Names

class Tenant:
   ''' 
   Tenant is an abstraction of a group of VMs and the associated privileges on specific datastores.
   This class provides operations to manage VMs and privileges associated with a tenant.
   '''

   _name = "vim.vcs.TenantManager"

   @Internal(parent=_name)
   @ManagedType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Tenant uuid
   """
   )
   @Attribute(parent=_name, typ="string")
   def id(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Tenant name
   """
   )
   @Attribute(parent=_name, typ="string")
   def name(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Tenant description
   """
   )
   @Attribute(parent=_name, typ="string", flags=F_OPTIONAL)
   def description(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Default datastore name
   """
   )
   @Attribute(parent=_name, typ="string", flags=F_OPTIONAL)
   def default_datastore(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   VMs belonging to this tenant
   """
   )
   @Attribute(parent=_name, typ="string[]", flags=F_OPTIONAL)
   def vms(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Privileges allowed on this datastore
   """
   )
   @Attribute(parent=_name, typ="vim.vcs.storage.DatastoreAccessPrivilege[]", flags=F_OPTIONAL)
   def privileges(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Add VMs to a tenant
   """
   )
   @Method(parent=_name, wsdlName="AddVMs")
   @Param(name="vms", typ="string[]")
   @Return(typ='void')
   def AddVMs(self, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove VMs from a tenant
   """
   )
   @Method(parent=_name, wsdlName="RemoveVMs")
   @Param(name="vms", typ="string[]")
   @Return(typ='void')
   def RemoveVMsFromTenant(self, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Get VMs for a tenant
   """
   )
   @Method(parent=_name, wsdlName="GetVMs")
   @Return(typ='string[]')
   def GetVMs(self, name):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Replace VMs for a tenant
   """
   )
   @Method(parent=_name, wsdlName="ReplaceVMs")
   @Param(name="vms", typ="string[]")
   @Return(typ='void')
   def ReplaceVMs(self, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Add a datastore access privilege for a tenant
   """
   )
   @Method(parent=_name, wsdlName="AddPrivilege")
   @Param(name="privilege", typ="vim.vcs.storage.DatastoreAccessPrivilege")
   @Return(typ='void')
   def AddPrivilege(self, privilege):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Modify datastore access privilege for a tenant
   """
   )
   @Method(parent=_name, wsdlName="ModifyPrivilege")
   @Param(name="privilege", typ="vim.vcs.storage.DatastoreAccessPrivilege")
   @Return(typ='void')
   def ModifyPrivilege(self, privilege):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove datastore access privilege for a tenant
   """
   )
   @Method(parent=_name, wsdlName="RemovePrivilege")
   @Param(name="datastore", typ="string")
   @Return(typ='void')
   def RemovePrivilege(self, datastore):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Get all datastore access privileges for a tenant
   """
   )
   @Method(parent=_name, wsdlName="GetPrivileges")
   @Param(name="datastore", typ="string", flags=F_OPTIONAL)
   @Return(typ='vim.vcs.storage.DatastoreAccessPrivilege[]')
   def GetPrivilege(self, datastore):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Modify default datastore for a tenant
   """
   )
   @Method(parent=_name, wsdlName="ModifyDatastore")
   @Param(name="datastore", typ="string")
   @Return(typ='void')
   def ModifyPrivilege(self, privilege):
       pass

RegisterVmodlTypes()
