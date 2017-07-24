---
title: Tenancy
---

Multi-tenancy is an architecture in which a single instance of a software application serves multiple customers or "tenants." Tenants can be used to provide isolation between independent groups in shared environments, where multiple groups are using some common infrastructure i.e. compute, storage, network, etc. With multi tenancy, you can achieve isolation of resources of one tenant from other tenants.


For the vSphere Docker Volume Service, Multi-tenancy is implemented by assigning a Datastore and VMs to a vmgroup.  A vmgroup can be granted access to create, delete or mount volumes on a specific datastore. VMs assigned to a vmgroup can then execute Docker volume APIs on an assigned datastores. Within a datastore multiple vmgroups can store their Docker volumes. A vmgroup cannot access volumes created by a different vmgroup i.e. vmgroups have their own independent namespace, even if vmgroups share datastores. VMs cannot be shared between vmgroups.

Key attributes of tenancy:

- vSphere Administrator can define group of one or more Docker Host (VM) as
vmgroup
- Docker Host (VM) can be a member of one and only one vmgroup.
- vSphere Administrator can grant vmgroup privileges & set resource consumption
- Vmgroups can share the same underlying storage but preserve volume namespace isolation limits at granularity of datastore.

## Admin CLI

Vmgroups can be created and managed via the [Admin CLI](/docs/user-guide/admin-cli.md/#vmgroup)

## Multitenancy concepts

### Default vmgroup

When a VM which does not belong to any vmgroup issues a request to vmdk_ops, this VM will be assumed to be in _DEFAULT vmgroup, and will get privileges associated with this vmgroup. \_DEFAULT vmgroup will be automatically created by system post install, so by default vmdk_ops will support request from any VM , thus maintaining backward compatibility and simplicity of installation.An admin can remove this vmgroup or modify privileges, thus locking down vmdk_ops to serve only explicitly configured VMs.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=_DEFAULT
Datastore  Allow_create  Max_volume_size  Total_size
---------  ------------  ---------------  ----------
_ALL_DS   True          Unset            Unset
_VM_DS    True          Unset            Unset
```

### Default datastore
When a VM addresses the volume using short notation (volume_name, without @datastore), all VMs in this vmgroup will use default datastore to resolve short volume reference (volume_name will actually mean volume_name@default_datastore).

"Default_datastore" is a required parameter when creating a vmgroup. The value is either a valid datastore name, or special string "_VM_DS".  The "Default_datastore" field is set to "_VM_DS" for "_DEFAULT" vmgroup post install and any volume created from VM which belongs to "_DEFAULT" vmgroup will be created on the datastore where VM resides.


## Example

Let's consider a sample use case where there are 2 teams – Dev and Test working on Product1. Lets create separate vmgroups (namely Product1Dev and Product1Test) for each of the teams where we can put restriction on datastore consumption.

```
[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=Product1Dev --default-datastore=datastore3
vmgroup 'Product1Dev' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.

[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=Product1Test --default-datastore=datastore3
vmgroup 'Product1Test' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.
```
The full access privilege to default datastore("datastore3" in this example) has been created after creating vmgroup.

```
[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=Product1Dev
Datastore     Allow_create  Max_volume_size  Total_size
------------  ------------  ---------------  ----------
datastore3    True          Unset            Unset

[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=Product1Test
Datastore     Allow_create  Max_volume_size  Total_size
------------  ------------  ---------------  ----------
datastore3    True          Unset            Unset

```
Then we removed _DEFAULT vmgroup to lock down vmdk_ops to serve only explicitly configured VMs.

```
[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup rm --name=_DEFAULT
vmgroup rm succeeded

[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name          Description  Default_datastore  VM_list
------------------------------------  ------------  -----------  -----------------  ------------
ac4b7167-94b3-470e-b932-5b32f2bfa273  Product1Dev                datastore3
f15c1f6d-5df5-4a00-8f20-77c8e7a7af11  Product1Test               datastore3
```

Lets add VMs (docker hosts) in respective VM groups.

```
#/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm add --vm-list=Photon1,Photon2,Photon3 --name=Product1Dev
vmgroup vm add succeeded

#/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm add --vm-list=Photon4,Photon5,Photon6 --name=Product1Test
vmgroup vm add succeeded

#/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name          Description  Default_datastore  VM_list
------------------------------------  ------------  -----------  -----------------  -----------------------
ac4b7167-94b3-470e-b932-5b32f2bfa273  Product1Dev                datastore3         Photon1,Photon2,Photon3
f15c1f6d-5df5-4a00-8f20-77c8e7a7af11  Product1Test               datastore3         Photon4,Photon5,Photon6

```

Lets limit dev team to create volumes of total 20 Gb each not exceeding size of 1 GB. Similarly for QA team, lets put restriction of total 40 GB consumption with each volume size not exceeding 1 GB.

```
[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access set --name=Product1Dev --datastore=datastore3  --volume-maxsize=1GB --volume-totalsize=20GB
vmgroup access set succeeded

[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access set --name=Product1Test --datastore=datastore3  --volume-maxsize=1GB --volume-totalsize=40GB
vmgroup access set succeeded

```
Lets verify that storage restrictions have been set properly.

```
[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=Product1Dev
Datastore     Allow_create  Max_volume_size  Total_size
------------  ------------  ---------------  ----------
datastore3    True          1.00GB           20.00GB

[root@sc-rdops-vm17-dhcp-3-236:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=Product1Test
Datastore     Allow_create  Max_volume_size  Total_size
------------  ------------  ---------------  ----------
datastore3    True          1.00GB           40.00GB

```
Lets try to create a volume from one of the QA machines with size = 2 GB

```
#  docker volume create --name=MyVolume --driver=vsphere -o size=2GB
Error response from daemon: create MyVolume: VolumeDriver.Create: Volume size exceeds the max volume size limit
```
vDVS has restricted user from creating a volume of size > 1 GB. Lets try to create volume on datastore other than the one which is set as default

```
root@sc-rdops-vm02-dhcp-52-237:~# docker volume create --name=MyVolume@datastore1 --driver=vsphere -o size=2GB
Error response from daemon: create MyVolume@datastore1: VolumeDriver.Create: No access privilege exists
```
Remember we have set default datastore as "datastore3" which as “--allow-create” permissions and we don't have access privilege to datastore "datastore1".



## References

- [Design Spec for tenancy](https://github.com/vmware/docker-volume-vsphere/blob/master/docs/misc/docker-volume-auth-proposal.v1_2.md)
