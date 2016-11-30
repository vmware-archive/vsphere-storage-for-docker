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
   """
   )
   @Method(parent=_name, wsdlName="GetTenantManager")
   @Return(typ="vim.vcs.TenantManager")
   def GetTenantManager(self):
       pass

RegisterVmodlTypes()