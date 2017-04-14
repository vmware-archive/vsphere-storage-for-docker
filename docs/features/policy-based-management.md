
# Storage Policy Based Management
Storage Policies capture storage requirements, such as performance and availability, for persistent volumes. These policies determine how the container volume storage objects are provisioned and allocated within the datastore to guarantee the requested Quality of Service. Storage policies are composed of storage capabilities, typically represented by a key-value pair. The key is a specific property that the datastore can offer and the value is a metric, or a range, that the datastore guarantees for a provisioned object, such as a container volume backed by a virtual disk. 

As you might be already aware, vSAN is a distributed layer of software that runs natively as a part of the ESXi hypervisor. vSAN aggregates local or direct-attached capacity devices of a host cluster and creates a single storage pool shared across all hosts in the vSAN cluster. While supporting VMware features that require shared storage, such as HA, vMotion, and DRS, vSAN eliminates the need for external shared storage and simplifies storage configuration and virtual machine provisioning activities. vSAN works with virtual machine storage policies to support a virtual machine-centric storage approach. When provisioning a virtual machine, if there is no explicit assignment of a storage policy to the virtual machine, a generic system defined storage policy, called the vSAN Default Storage Policy is automatically applied to the virtual machine.

We will now illustrate how this policy based management approach can be applied to container volumes as well.

As described in [official documentation](https://pubs.vmware.com/vsphere-65/index.jsp?topic=%2Fcom.vmware.vsphere.virtualsan.doc%2FGUID-08911FD3-2462-4C1C-AE81-0D4DBC8F7990.html), vSAN exposes multiple storage capabilities.


# Using Storage Policies in vDVS
With vDVS, vSphere administrators will have the ability to create custom VSAN Policies that can then be specified during Docker Volume creations. When VSAN policies are assigned to the Docker Volumes, the back-end storage objects get created on the VSAN datastore in the form of virtual disks. This is why the VSAN Policies can be applied seamlessly to the volumes created by vDVS.

To create a new VSAN Policy, you will need to specify the name of the policy and provide the set of VSAN capabilities formatted using the same syntax found in esxcli vsan policy getdefault command. For example: (("hostFailuresToTolerate" i1) ("forceProvisioning" i1)). Here is a list of storage capabilities supported by VSAN (for more detailed information, please refer to official Virtual SAN documentation):


|VSAN Capability|Description|
|------|------|
|hostFailuresToTolerate|Number of failures to tolerate|
|stripeWidth|Number of disk stripes per object|
|forceProvisioning| Force provisioning|
|proportionalCapacity| Object space reservation|
|cacheReservation|Flash read cache reservation|
|checksumDisabled|Disable object checksum|
|replicaPreference|Failure tolerance method|
|iopsLimit|IOPS limit for object|

Here's a step-by-step guide to create and use VSAN Storage Policies for Docker Volumes:

- Create a "gold" and a "silver" storage policies


```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy create --name gold --content '(("hostFailuresToTolarate" i0)("forceProvisioning" i1))'
Successfully created policy: gold
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy create --name silver --content '(("hostFailuresToTolarate" i1)("forceProvisioning" i1))'
Successfully created policy: silver
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy ls
Policy Name  Policy Content                                           Active
-----------  -------------------------------------------------------  ------
gold         (("hostFailuresToTolarate" i0)("forceProvisioning" i1))  Unused
silver       (("hostFailuresToTolarate" i1)("forceProvisioning" i1))  Unused
```

- Create 2 Docker Volume using above policies

```
#docker volume create -d vsphere --name gold_volume@vsanDatastore -o size=1Gb -o vsan-policy-name=go
ld
gold_volume@vsanDatastore
#docker volume create -d vsphere --name silver_volume@vsanDatastore -o size=1Gb -o vsan-policy-name=
silver
silver_volume@vsanDatastore
#docker volume ls
vsphere:latest      gold_volume@vsanDatastore
vsphere:latest      silver_volume@vsanDatastore
```

- Verify if the policies are being used by the volumes

```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy ls
Policy Name  Policy Content                                           Active
-----------  -------------------------------------------------------  -------------------
silver       (("hostFailuresToTolarate" i1)("forceProvisioning" i1))  In use by 1 volumes
gold         (("hostFailuresToTolarate" i0)("forceProvisioning" i1))  In use by 1 volumes
```

- Update one policy content

```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy update --name silver --content '(("hostFailuresToTolarate" i2)("forceProvisioning" i1))'
This operation may take a while. Please be patient.
Successfully updated policy: silver
```

- Remove the volume before removing policy

```
docker volume rm gold_volume@vsanDatastore
gold_volume@vsanDatastore
```

- Remove the policy

```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy rm --name gold
Successfully removed policy: gold
```
