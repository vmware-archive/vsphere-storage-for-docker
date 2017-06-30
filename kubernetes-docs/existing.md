---
title: Configurations on Existing Kubernetes Cluster
---

If a Kubernetes cluster has not been deployed using Kubernetes-Anywhere, follow the instructions below to use the vSphere Cloud Provider. These steps are not needed when using Kubernetes-Anywhere, they will be done as part of the deployment.

**Enable UUID for a VM**

This can be done via govc tool

```
export GOVC_URL=<IP/URL>
export GOVC_USERNAME=<vCenter User>
export GOVC_PASSWORD=<vCenter Password>
export GOVC_INSECURE=1
govc vm.change -e="disk.enableUUID=1" -vm=<VMNAME>
```

To enable disk UUID, vCenter user requires following  privileges.

```
$ govc role.ls change-uuid
System.Anonymous
System.Read
System.View
VirtualMachine.Config.AdvancedConfig
VirtualMachine.Config.Settings
```
 
 
**Create Role and User with Required Privileges for vSphere Cloud Provider**

vSphere Cloud Provider requires the following minimal set of privileges to interact with vCenter:
Please refer vSphere Documentation Center to know about steps for creating a Custom Role, User and Role Assignment.
Note: Assign Permissions at the vCenter Level and make sure to check Propagate.

```
Datastore > Allocate space
Datastore > Low level file Operations
Virtual Machine > Configuration > Add existing disk
Virtual Machine > Configuration > Add or remove device
Virtual Machine > Configuration > Remove disk
Network > Assign network
Virtual machine > Configuration > Add new disk
Virtual Machine > Inventory > Create new
Virtual machine > Configuration > Add new disk
Resource > Assign virtual machine to resource pool
Profile-driven storage -> Profile-driven storage view
```


Provide the cloud config file to each instance of kubelet, apiserver and controller manager via --cloud-config=<path to file> flag. Cloud config template can be found at Kubernetes-Anywhere
Sample Config:

```
[Global]
        user = <User name for vCenter>
        password = <Password for vCenter>
        server = <IP/URL for vCenter>
        port = <Default 443 for vCenter>
        insecure-flag = <set to 1 if the host above uses a self-signed cert>
        datacenter = <Datacenter to be used>
        datastore = <Datastore to use for provisioning volumes using storage 
                     classes/dynamic provisioning>
        working-dir = <Folder in which VMs are provisioned, can be null. It should 
                      be full path to the folder in which Kubernetes nodes are 
                      provisioned. Not just a folder name. If deployed in root, 
                      no need to specify the folder name>
        vm-uuid = <VM Instance UUID of virtual machine which can be retrieved 
                    from instanceUuid property in VmConfigInfo, or also set as 
                    vc.uuid in VMX file. If empty, will be retrieved from sysfs
                    (requires root)>
[Disk]
    scsicontrollertype = pvscsi
```

 
VM UUID can be retrieved using following script. Execute this script on each node and update vsphere.conf file mentioned above.

```
cat /sys/class/dmi/id/product_serial | sed -e 's/^VMware-//' -e 's/-/ /' | awk '{ print toupper($1$2$3$4 "-" $5$6 "-" $7$8 "-" $9$10 "-" $11$12$13$14$15$16) }'
```

Set the cloud provider via --cloud-provider=vsphere flag for each instance of kubelet, apiserver and controller manager.

When upgrading to 1.6 install the default storage class addons.

Using resource pool and maintaining the VMs from each cluster in their respective cluster folder you can run multiple Kubernetes cluster on vCenter.

vSphere supports shared storage across multiple vCenters. User can use shared storage in multiple Kubernetes Clusters.
 
