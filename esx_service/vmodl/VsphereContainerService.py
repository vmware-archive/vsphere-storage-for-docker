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

class VsphereContainerService:
   _name = "vim.vcs.VsphereContainerService"

   @JavaDocs(parent=_name, docs =
   """
   This is the bootstrapping class for vSphere Container Service (VCS).
   VCS enables customers to run stateful containerized applications on top
   of VMware vSphere platform. This service addresses persistent storage
   requirements for Docker containers in vSphere environments, and enables
   multitenancy, security and access control from a single location.
   """
   )
   @Internal(parent=_name)
   @ManagedType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Get the TenantManager instance.
   @return TenantManager instance.
   """
   )
   @Method(parent=_name, wsdlName="GetTenantManager")
   @Return(typ="vim.vcs.TenantManager")
   def GetTenantManager(self):
       pass

class TenantManager:
   _name = "vim.vcs.TenantManager"

   @JavaDocs(parent=_name, docs =
   """
   This class manages the lifecycle of tenants for vSphere Container Service.
   """
   )
   @Internal(parent=_name)
   @ManagedType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Create a Tenant instance.
   @param name Name of the tenant.
   @param description Detailed description of the tenant. Optional.
   @throws vmodl.fault.InvalidArgument If the given name or description exceeds maximum length.
   @throws vim.fault.AlreadyExists If the given tenant name already exists.
   """
   )
   @Method(parent=_name, wsdlName="CreateTenant",
           faults=["vmodl.fault.InvalidArgument", "vim.fault.AlreadyExists"])
   @Param(name="name", typ="string")
   @Param(name="description", typ="string", flags=F_OPTIONAL)
   @Return(typ="vim.vcs.Tenant")
   def CreateTenant(self, name, description=None):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove a Tenant instance. Remove all volumes created by this tenant
   if remove_volumes flag is set to True.
   @param name Tenant name
   @param remove_volumes Default value is False. If set to True,
          remove all volumes created by this Tenant.
   @throws vim.fault.NotFound If the Tenant name does not exist.
   """
   )
   @Method(parent=_name, wsdlName="RemoveTenant", faults=["vim.fault.NotFound"])
   @Param(name="name", typ="string")
   @Param(name="remove_volumes", typ="boolean", flags=F_OPTIONAL)
   @Return(typ="void")
   def RemoveTenant(self, name, remove_volumes=False):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Query Tenant for the given name.
   @param name Tenant name.
   @return Tenant for the given name, or all tenants if name is not set.
   @throws vim.fault.NotFound If the Tenant name does not exist.
   """
   )
   @Method(parent=_name, wsdlName="GetTenants", faults=["vim.fault.NotFound"])
   @Param(name="name", typ="string", flags=F_OPTIONAL)
   @Return(typ="vim.vcs.Tenant[]")
   def GetTenants(self, name=None):
       pass 

   @JavaDocs(parent=_name, docs=
   """
   Modify the attributes of a Tenant instance.
   @param name Current name of the tenant
   @param new_name New name of the tenant
   @param new_description New description of the tenant
   @throws vim.fault.NotFound If the name does not exist.
   @throws vim.fault.AlreadyExists If the new_name already exists.
   """
   )
   @Method(parent=_name, wsdlName="ModifyTenant",
           faults=["vim.fault.NotFound", "vim.fault.AlreadyExists"])
   @Param(name="name", typ="string")
   @Param(name="new_name", typ="string", flags=F_OPTIONAL)
   @Param(name="new_description", typ="string", flags=F_OPTIONAL)
   @Return(typ="void")
   def ModifyTenant(self, name, new_name=None, new_description=None):
       pass

class Tenant:
   _name = "vim.vcs.Tenant"

   @JavaDocs(parent=_name, docs =
   """
   Tenant is an abstraction of a group of VMs. Each tenant may have different
   privileges on different datastores. This class provides operations to manage
   VMs belonging to a tenant, and the privileges assigned to this tenant.
   """
   )
   @Internal(parent=_name)
   @ManagedType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Tenant uuid. This is a global unique id generated by the system. 
   """
   )
   @Attribute(parent=_name, typ="string")
   def id(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Tenant name, with maximum 64 characters. Tenant name must be unique.
   """
   )
   @Attribute(parent=_name, typ="string")
   def name(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Tenant description, with maximum 256 characters.
   """
   )
   @Attribute(parent=_name, typ="string", flags=F_OPTIONAL)
   def description(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Default datastore. A tenant may have different privileges on different datastores.
   Default datastore will be used when a VM addresses the volume using short notation
   (i.e. volume_name without @datastore suffix). 
   """
   )
   @Attribute(parent=_name, typ="vim.Datastore", flags=F_OPTIONAL)
   def default_datastore(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   VMs belonging to this tenant.
   """
   )
   @Attribute(parent=_name, typ="vim.VirtualMachine[]", flags=F_OPTIONAL)
   def vms(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Privileges assigned to this tenant.
   """
   )
   @Attribute(parent=_name, typ="vim.vcs.storage.DatastoreAccessPrivilege[]", flags=F_OPTIONAL)
   def privileges(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Add VMs to this tenant. Adding a VM already belonging to this Tenant has no effect.
   @param vms VMs to be added to this tenant.
   """
   )
   @Method(parent=_name, wsdlName="AddVMs")
   @Param(name="vms", typ="vim.VirtualMachine[]")
   @Return(typ='void')
   def AddVMs(self, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove VMs from this tenant.
   @param vms List of VMs to be removed from this tenant.
   @throws vim.fault.NotFound If any VM does not exist or does not belong to this tenant.

   """
   )
   @Method(parent=_name, wsdlName="RemoveVMs", faults=["vim.fault.NotFound"])
   @Param(name="vms", typ="vim.VirtualMachine[]")
   @Return(typ='void')
   def RemoveVMs(self, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Get all VMs belonging to this tenant.
   @return VMs belonging to this tenant.
   """
   )
   @Method(parent=_name, wsdlName="GetVMs")
   @Return(typ='vim.VirtualMachine[]')
   def GetVMs(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Replace VMs for this tenant. All existing VMs will be removed and replaced with
   new ones.
   @param vms VMs to be added to this tenant replacing existing VMs.
   """
   )
   @Method(parent=_name, wsdlName="ReplaceVMs")
   @Param(name="vms", typ="vim.VirtualMachine[]")
   @Return(typ='void')
   def ReplaceVMs(self, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Add a datastore access privilege for this tenant.
   @param privilege privilege to be assigned to this tenant.
   @param switch_default_datastore If set to true, switch the default datastore
          for this tenant. False by default.
   """
   )
   @Method(parent=_name, wsdlName="AddPrivilege")
   @Param(name="privilege", typ="vim.vcs.storage.DatastoreAccessPrivilege")
   @Param(name="switch_default_datastore", typ="boolean", flags=F_OPTIONAL)
   @Return(typ='void')
   def AddPrivilege(self, privilege, switch_default_datastore=False):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Modify datastore access privilege for this tenant. Datastore itself cannot be changed.
   @param datastore Privilege settings on this datastore will be changed.
   @param allow_create If set to true, this tenant is allowed to create and delete volumes on
          the datastore. If false, the tenant is allowed to mount and unmount volumes only.
   @param volume_max_size Maximum size (in MiB) of one single volume. Zero means unlimited size.
   @param volume_total_size Total storage usage quota (in MiB) allowed on this datastore.
          Zero means unlimited quota.
   @throws vim.fault.NotFound If the given datastore does not exist for this tenant.
   @throws vmodl.fault.InvalidArgument If the specified parameters are invalid,
           e.g. volume total size exceeds the datastore capacity.
   """
   )
   @Method(parent=_name, wsdlName="ModifyPrivilege",
           faults=["vim.fault.NotFound", "vmodl.fault.InvalidArgument"])
   @Param(name="datastore", typ="vim.Datastore")
   @Param(name="allow_create", typ="boolean", flags=F_OPTIONAL)
   @Param(name="volume_max_size", typ="int", flags=F_OPTIONAL)
   @Param(name="volume_total_size", typ="int", flags=F_OPTIONAL)
   @Return(typ='void')
   def ModifyPrivilege(self, datastore, allow_create, volume_max_size, volume_total_size):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove access privilege on the given datastore from this tenant.
   @param datastore Privilege on this datastore will be removed from this tenant.
   """
   )
   @Method(parent=_name, wsdlName="RemovePrivilege")
   @Param(name="datastore", typ="vim.Datastore")
   @Return(typ='void')
   def RemovePrivilege(self, datastore):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Get privilege of this tenant on the given datastore.
   @datastore If set, return privilege for this datastore only; otherwise return all privileges.
   @return Privilege of this tenant on the given datastore.
   @throw NotFound If this tenant has no privilege on the given datastore.
   """
   )
   @Method(parent=_name, wsdlName="GetPrivileges",
        faults=["vim.fault.NotFound"])
   @Param(name="datastore", typ="vim.Datastore", flags=F_OPTIONAL)
   @Return(typ='vim.vcs.storage.DatastoreAccessPrivilege[]')
   def GetPrivileges(self, datastore):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Modify default datastore for a tenant.
   @param datastore New default datastore for this tenant.
   """
   )
   @Method(parent=_name, wsdlName="ModifyDatastore")
   @Param(name="datastore", typ="vim.Datastore")
   @Return(typ='void')
   def ModifyDefaultDatastore(self, datastore):
       pass

class DatastoreAccessPrivilege:
   _name = "vim.vcs.storage.DatastoreAccessPrivilege"

   @JavaDocs(parent=_name, docs =
   """
   A privilege defines operations against a datastore and limits on those operations.
   """
   )
   @Internal(parent=_name)
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