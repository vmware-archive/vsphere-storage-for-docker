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

class TenantManager:
   ''' This class manages the lifecycle of tenants for vSphere Container Service '''

   _name = "vim.vcs.TenantManager"

   @Internal(parent=_name)
   @ManagedType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Create a Tenant instance
   """
   )
   @Method(parent=_name, wsdlName="CreateTenant")
   @Param(name="name", typ="string")
   @Param(name="description", typ="string", flags=F_OPTIONAL)
   @Return(typ="vim.vcs.Tenant")
   def CreateTenant(self, name, description=None):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove a Tenant instance
   """
   )
   @Method(parent=_name, wsdlName="RemoveTenant")
   @Param(name="name", typ="string")
   @Param(name="remove_volumes", typ="boolean")
   @Return(typ="void")
   def RemoveTenant(self, name):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Query Tenants for the given name. Return all Tenants if name is null.
   """
   )
   @Method(parent=_name, wsdlName="GetTenants")
   @Param(name="name", typ="string", flags=F_OPTIONAL)
   @Return(typ="vim.vcs.Tenant[]")
   def GetTenants(self): 
       pass 

   @JavaDocs(parent=_name, docs=
   """
   Modify the attributes of a Tenant instance
   """
   )
   @Method(parent=_name, wsdlName="RemoveTenant")
   @Param(name="name", typ="string")
   @Param(name="new_name", typ="string", flags=F_OPTIONAL)
   @Param(name="new_description", typ="string", flags=F_OPTIONAL)
   @Return(typ="void")
   def RemoveTenant(self, name):
       pass

RegisterVmodlTypes()
