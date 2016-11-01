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

class VsanDockerPersistentVolumeTenantList:
   _name = "vim.vsan.VsanDockerPersistentVolumeTenantList"
   @JavaDocs(parent=_name, docs =
   """
   This class encapsulates the Docker Volume Tenant List Result.
   """
   )
   
   @DataType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Defines a list of tenants
   """
   )
   @Attribute(parent=_name, typ="string[]")
   def tenants(self):
       pass

class VsanDockerPersistentVolumeDatastoreAccessPrivileges:
   _name = "vim.vsan.VsanDockerPersistentVolumeDatastoreAccessPrivileges"
   @JavaDocs(parent=_name, docs =
   """
   This class encapsulates the Docker Volume Datastore Access Privileges.
   """
   )
   
   @DataType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Datastore name
   """
   )
   @Attribute(parent=_name, typ="string")
   def datastore(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates whether the privilege has create volume privilege
   """
   )
   @Attribute(parent=_name, typ="boolean")
   def create_volumes(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates whether the privilege has delete volume privilege
   """
   )
   @Attribute(parent=_name, typ="boolean")
   def delete_volumes(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates whether the privilege has mount volume privilege
   """
   )
   @Attribute(parent=_name, typ="boolean")
   def mount_volumes(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates max volume size allowed on this datastore
   """
   )
   @Attribute(parent=_name, typ="string")
   def max_volume_size(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates total storage usage allowed on this datastore
   """
   )
   @Attribute(parent=_name, typ="string")
   def usage_quota(self):
       pass

class VsanDockerPersistentVolumeTenant:
   _name = "vim.vsan.VsanDockerPersistentVolumeTenant"
   @JavaDocs(parent=_name, docs =
   """
   This class encapsulates the Docker Volume Tenant.
   """
   )
      
   @DataType(name=_name, version=_VERSION)
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
   @Attribute(parent=_name, typ="string")
   def description(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Default datastore name
   """
   )
   @Attribute(parent=_name, typ="string")
   def default_datastore(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates whether the privilege has mount volume privilege
   """
   )
   @Attribute(parent=_name, typ="vim.vsan.VsanDockerPersistentVolumeDatastoreAccessPrivileges")
   def default_privileges(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates max volume size allowed on this datastore
   """
   )
   @Attribute(parent=_name, typ="string[]", flags=F_OPTIONAL)
   def vms(self):
       pass

   @JavaDocs(parent=_name, docs =
   """
   Indicates total storage usage allowed on this datastore
   """
   )
   @Attribute(parent=_name, typ="vim.vsan.VsanDockerPersistentVolumeDatastoreAccessPrivileges[]", flags=F_OPTIONAL)
   def privileges(self):
       pass     

class VsanDockerPersistentVolumeSystem:
   ''' This is the API to Docker Persistent Volumes on VSAN'''

   _name = "vim.host.VsanDockerPersistentVolumeSystem"

   @Internal(parent=_name)
   @ManagedType(name=_name, version=_VERSION)
   def __init__(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Query the tenant list. This method returns a list of tenants, as strings. 
   The method is blocking on requires DB IO. 
   It SHOULD be converted to async task, eventually.
   """
   )
   @Method(parent=_name, wsdlName="GetTenantList")
   @Return(typ="vim.vsan.VsanDockerPersistentVolumeTenantList")
   def GetTenantList(self):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Create DatastoreAccessPrivileges object
   """
   )
   @Method(parent=_name, wsdlName="CreateDatastoreAccessPrivileges")
   @Param(name="datastore", typ="string")
   @Param(name="create_volumes", typ="boolean")
   @Param(name="delete_volumes", typ="boolean")
   @Param(name="mount_volumes", typ="boolean")
   @Param(name="max_volume_size", typ="string")
   @Param(name="usage_quota", typ="string")
   @Return(typ="vim.vsan.VsanDockerPersistentVolumeDatastoreAccessPrivileges")
   def CreateDatastoreAccessPrivileges(self, datastore, create_volumes, delete_volumes, mount_volumes, max_volume_size, usage_quota):
       pass
   
   @JavaDocs(parent=_name, docs=
   """
   Get DatastoreAccessPrivileges object
   """
   )
   @Method(parent=_name, wsdlName="GetDatastoreAccessPrivileges")
   @Return(typ="vim.vsan.VsanDockerPersistentVolumeDatastoreAccessPrivileges")
   def GetDatastoreAccessPrivileges(self):
       pass  

   @JavaDocs(parent=_name, docs=
   """
   Create a VsanDockerPersistentVolumeTenant object
   """
   )
   @Method(parent=_name, wsdlName="CreateTenant")
   @Param(name="name", typ="string")
   @Param(name="description", typ="string")
   @Param(name="default_datastore", typ="string")
   @Param(name="default_privileges", typ="vim.vsan.VsanDockerPersistentVolumeDatastoreAccessPrivileges")
   @Param(name="vms", typ="string[]", flags=F_OPTIONAL)
   @Param(name="privileges", typ="vim.vsan.VsanDockerPersistentVolumeDatastoreAccessPrivileges[]", flags=F_OPTIONAL)
   @Return(typ="vim.vsan.VsanDockerPersistentVolumeTenant")
   def CreateTenant(self, name, description, default_datastore, 
                    default_privileges, vms=None, privileges=None):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove a VsanDockerPersistentVolumeTenant object
   """
   )
   @Method(parent=_name, wsdlName="RemoveTenant")
   @Param(name="name", typ="string")
   @Return(typ="void")
   def RemoveTenant(self, name):
       pass

   @JavaDocs(parent=_name, docs=
   """
   List all VsanDockerPersistentVolumeTenant objects
   """
   )
   @Method(parent=_name, wsdlName="ListTenants")
   @Return(typ="vim.vsan.VsanDockerPersistentVolumeTenant[]")
   def ListTenants(self): 
       pass 

   @JavaDocs(parent=_name, docs=
   """
   Add VMs to a tenant
   """
   )
   @Method(parent=_name, wsdlName="AddVMsToTenant")
   @Param(name="name", typ="string")
   @Param(name="vms", typ="string[]")
   @Return(typ='void')
   def AddVMsToTenant(self, name, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Remove VMs from a tenant
   """
   )
   @Method(parent=_name, wsdlName="RemoveVMsFromTenant")
   @Param(name="name", typ="string")
   @Param(name="vms", typ="string[]")
   @Return(typ='void')
   def RemoveVMsFromTenant(self, name, vms):
       pass

   @JavaDocs(parent=_name, docs=
   """
   List VMs for a tenant
   """
   )
   @Method(parent=_name, wsdlName="ListVMsForTenant")
   @Param(name="name", typ="string")
   @Return(typ='string[]')
   def ListVMsForTenant(self, name):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Add DatastoreAccessPrivileges for a tenant
   The fomat of param "rights" is a list of strings, and the valid choices are "create", "delete", "mount" and "all"
   The format of param "volume-maxsize" is Num{MB,GB,TB} - e.g. 2TB
   The format of param "volume-totalsize" is Num{MB,GB,TB} - e.g. 2TB
   """
   )
   @Method(parent=_name, wsdlName="AddDatastoreAccessForTenant")
   @Param(name="name", typ="string")
   @Param(name="datastore", typ="string")
   @Param(name="rights", typ="string[]", flags=F_OPTIONAL)
   @Param(name="volume_maxsize", typ="string", flags=F_OPTIONAL)
   @Param(name="volume_totalsize", typ="string", flags=F_OPTIONAL)
   @Return(typ='void')
   def AddDatastoreAccessForTenant(self, name, datastore, rights=None, volume_maxsize=None, volume_totalsize=None):
       pass

   @JavaDocs(parent=_name, docs=
   """
   Modify DatastoreAccessPrivileges for a tenant
   Similar rules apply to param "delete_volumes" and "mount_volumes" too 
   """
   )
   @Method(parent=_name, wsdlName="ModifyDatastoreAccessForTenant")
   @Param(name="name", typ="string")
   @Param(name="datastore", typ="string")
   @Param(name="add_rights", typ="string", flags=F_OPTIONAL)
   @Param(name="delete_volumes", typ="boolean", flags=F_OPTIONAL)
   @Param(name="mount_volumes", typ="boolean", flags=F_OPTIONAL)
   @Param(name="max_volume_size", typ="string", flags=F_OPTIONAL)
   @Param(name="usage_quota", typ="string", flags=F_OPTIONAL)
   @Return(typ='void')
   def ModifyDatastoreAccessForTenant(self, name, datastore, create_volumes=None, delete_volumes=None, 
                                      mount_volumes=None, max_volume_size=None, usage_quota=None):
       pass                

RegisterVmodlTypes()
