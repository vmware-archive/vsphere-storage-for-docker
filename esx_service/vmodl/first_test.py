#!/usr/bin/env python
# Copyright 2016 VMWare, Inc. All Rights Reserved.
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

#
# An example of Connecting to (local) VSAN SIMS and fetching simple data 
# from VSAN and VsanDockerPersistentVolumeSystem
#
# Usage: 
#
# 1. drop VsanDockerPersistentVolumeSystem*py files to 
#        /lib/python2.7/site-packages/pyMo/vim/vsan or 
#       /lib64/python3.5/site-packages/pyMo/vim/vsan
#   Do not forget /etc/init.d/vsanmgmtd restart
# 
# 1a. drop the file below (fist_test.py) in local folder or wherever import works from 
#
# 2. in python , run the following:
# import fist_test
# stub = fist_test.connect_to_vsanmgmt()
# fist_test.get_vsan_network_info(stub) # prints vsan info 
# fist_test.get_dockvol_obj(stub)       # prints tenant list (fake string)

import sys
sys.path.append('/lib64/python3.5/site-packages/pyMo/vim/vsan')
sys.path.append('/lib/python2.7/site-packages/pyMo/vim/vsan')

import pyVim
import pyVim.connect
import  pyVim.host

import pyVmomi
import pyVmomi.VmomiSupport
from pyVmomi import vim

from vsanPerfPyMo import VsanPerformanceManager

si = None

def connect_to_vsanmgmt(host = "localhost", port = 443):

    """
    Connects to VSAN mgmt service on ESX (/vsan) and returns SOAP stub
    to use
    """

    version = 'vim.version.version10'

    global si
    if si:
        pyVim.connect.Disconnect(si)

    si = pyVim.connect.Connect(host=host, version=version)
    hostSystem = pyVim.host.GetHostSystem(si)

    token = hostSystem.configManager.vsanSystem.FetchVsanSharedSecret()

    version = pyVmomi.VmomiSupport.newestVersions.Get("vim")
    stub = pyVmomi.SoapStubAdapter(host=host,
                                    port=443,
                                    version=version,
                                    path="/vsan",
                                    poolSize=0)
    vpm = vim.cluster.VsanPerformanceManager("vsan-performance-manager", stub)
    logged_in = vpm.Login(token)

    if not logged_in:
        print("Failed to get sims stub for host %s" % host.name)
        raise OSError("can't login'")

    print("Connected to VSAN mgmt on " + host)
    return stub

def get_vsan_network_info(stub):
    vhs = vim.HostVsanHealthSystem('ha-vsan-health-system', stub)
    return vhs.VsanHostQueryVerifyNetworkSettings()

def print_datastore_access_privileges_obj(privileges):
    content = privileges.__dict__
    for i in content.keys():
        print ("{0}: value={1}".format(i, content[i]))

def print_tenant_obj(tenant):
    print("name: ", tenant.name)
    print("description: ", tenant.description)
    print("default_datastore: ", tenant.default_datastore)
    print("default_privileges: ")
    print_datastore_access_privileges_obj(tenant.default_privileges)
    print("vms: ", tenant.vms)
    if tenant.privileges:
        print("privileges: ")
        for p in tenant.privileges:
            print_datastore_access_privileges_obj(p)
    else:
        print("privileges: ", tenant.privileges)    



if __name__ == "__main__":
    stub = connect_to_vsanmgmt()
    print("\n**** Getting VSAN network info: \n", get_vsan_network_info(stub))

    import VsanDockerPersistentVolumeSystem
    pv = vim.host.VsanDockerPersistentVolumeSystem("vsan-docker-persistent-volumes", stub)

    print("\n**** Getting DOCKVOL TENANTS:")
    tenantsList = pv.GetTenantList()
    print("  total: ", len(tenantsList.tenants),
          "list: ", tenantsList.tenants, 
          "First tenant: ", tenantsList.tenants[0]
    )

    print("\n**** Get Datastore Access Privileges Object:")
    privileges = pv.GetDatastoreAccessPrivileges()
    content = privileges.__dict__
    for i in content.keys():
        print ("{0}: value={1}".format(i, content[i]))
    

    print("\n**** Create Datastore Access Privileges Object:")
    datastore = "datastore2"
    create_volumes = True
    delete_volumes = False
    mount_volumes = True
    max_volume_size = '600MB'
    usage_quota = '2TB'
    privileges = pv.CreateDatastoreAccessPrivileges(datastore = datastore,
                                                    create_volumes = create_volumes,
                                                    delete_volumes = delete_volumes,
                                                    mount_volumes = mount_volumes,
                                                    max_volume_size = max_volume_size,
                                                    usage_quota = usage_quota)
    content = privileges.__dict__
    for i in content.keys():
        print ("{0}: value={1}".format(i, content[i]))
    
    print("\n**** Create Tenant Object:")
    name = "tenant1"
    description = "My first tenant"
    default_datastore = "default_ds"
    default_privileges = pv.CreateDatastoreAccessPrivileges(datastore = "default_ds",
                                                            create_volumes = True,
                                                            delete_volumes = True,
                                                            mount_volumes = True,
                                                            max_volume_size = "No_limit",
                                                            usage_quota = "No_limit")
    vms = ["vm1", "vm2"]
    tenant = pv.CreateTenant(name = name,
                             description = description,
                             default_datastore = default_datastore,
                             default_privileges = default_privileges)
    print_tenant_obj(tenant)

    print("\n**** Remove Tenant Object:")
    name = "tenant1"
    tenant = pv.RemoveTenant(name = name)
    print("\n**** Remove Tenant done:")

    print("\n**** List Tenant Objects:")
    tenant_list = pv.ListTenants()
    print("Tenant lists len = {0}\n\n".format(len(tenant_list)))
    id = 0
    for tenant in tenant_list:
        id = id + 1
        print("Tenant Info for tenant {0} Start".format(id))
        print_tenant_obj(tenant)
        print("Tenant Info for tenant {0} End".format(id))
        print("\n\n")
                          
    print("\n**** Add VMs to tenant:")
    name = "tenant1"
    vms = ["vm1", "vm2"]
    pv.AddVMsToTenant(name = name,
                      vms = vms)
    
    print("\n**** Add VMs to tenant done:")

    print("\n**** Remove VMs from tenant:")
    name = "tenant1"
    vms = ["vm1", "vm2"]
    pv.RemoveVMsFromTenant(name = name,
                           vms = vms)
    print("\n**** Remove VMs from tenant done:")

    print("\n**** List VMs for tenant:")
    name = "tenant1"
    vms = pv.ListVMsForTenant(name = name)
    print("vms={0}".format(vms))          

    print("\n**** Add Datastore Access for tenant:")
    name = "tenant1"
    pv.AddDatastoreAccessForTenant(name = name,
                                   datastore = "datastore1",
                                   rights =["create", "mount"],
                                   volume_maxsize = "550MB",
                                   volume_totalsize = "3TB")
    print("\n**** Add Datastore Access for tenant done:")

    print("\n**** Modify Datastore Access for tenant:")
    name = "tenant1"
    pv.ModifyDatastoreAccessForTenant(name = name,
                                   datastore = "datastore1",
                                   add_rights =["create", "mount"],
                                   remove_rights =["delete"],
                                   volume_maxsize = "650MB",
                                   volume_totalsize = "4TB")
    print("\n**** Modify Datastore Access for tenant done:")

    print("\n**** Remove Datastore Access for tenant:")
    
    pv.RemoveDatastoreAccessForTenant(name = "tenant1",
                                      datastore = "datastore1")
                                   
    print("\n**** Remove Datastore Access for tenant done:")


    print("\n**** List Datastore Access for Tenant:")
    privileges_list = pv.ListDatastoreAccessForTenant(name = "tenant1")
    print("Privileges lists len = {0}\n\n".format(len(tenant_list)))
    id = 0
    for privileges in privileges_list:
        id = id + 1
        print("Privileges {0}: ".format(id))
        print_datastore_access_privileges_obj(privileges)
        print("\n\n")
                                 
    