---
title: Multi-Tenancy using VMGroups
---

For the vSphere Docker Volume Service, Multi-tenancy is implemented by assigning a Datastore and VMs to a vmgroup. 
1.	By default _DEFAULT vmgroup is created as part of installation and new VMs get assigned to this vmgroup and will get privileges associated with this vmgroup.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup
```

2.	Let’s consider a sample use case. There are 2 teams - Dev and Test - working on Product1.
We create separate vmgroups (namely Product1Dev and Product2Test) for each of the teams where we can put restrictions on datastore consumption for each team.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=Product1Dev
vmgroup 'Product1Dev' is created.  Do not forget to run 'vmgroup vm add' and 'vmgroup access add' commands to enable access control.
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=Product1Test
vmgroup 'Product1Test' is created.  Do not forget to run 'vmgroup vm add' and 'vmgroup access add' commands to enable access control.
```
3. Now we remove _DEFAULT vmgroup to lock down vmdk_ops to serve only explicitly configured VMs.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup rm --name=_DEFAULT
vmgroup rm succeeded
```

4. Verify that vmgroups have got created.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name          Description  Default_datastore  VM_list
------------------------------------  ------------  -----------  -----------------  -------
ac4b7167-94b3-470e-b932-5b32f2bfa273  Product1Dev
f15c1f6d-5df5-4a00-8f20-77c8e7a7af11  Product1Test
```

5.	Let’s add VMs (docker hosts) in respective VM groups.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm add --vm-list=Photon1,Photon2,Photon3 --name=Product1Dev
vmgroup vm add succeeded
 [root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm add --vm-list=Photon4,Photon5,Photon6 --name=Product1Test
vmgroup vm add succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name          Description  Default_datastore  VM_list
------------------------------------  ------------  -----------  -----------------  -----------------------
ac4b7167-94b3-470e-b932-5b32f2bfa273  Product1Dev                                   Photon1,Photon2,Photon3
f15c1f6d-5df5-4a00-8f20-77c8e7a7af11  Product1Test                                  Photon4,Photon5,Photon6
```

6.	Put limit on VMs (docker hosts) of dev team to create volumes of total 20 Gb each not exceeding size of 1 GB. Similarly for QA team, the storage consuption should not exceed total of 40 GB with each volume of maximum 1GB size.

```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=Product1Dev --datastore=datastore3  --allow-create --default-datastore --volume-maxsize=1GB --volume-totalsize=20GB
vmgroup access add succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=Product1Test --datastore=datastore3  --allow-create --default-datastore --volume-maxsize=1GB --volume-totalsize=40GB
vmgroup access add succeeded
```

7. Verify that storage restrictions has been set properly.
```
[root@localhost:~] usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name Product1Dev
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore3  True          1.00GB           20.00GB

[root@localhost:~] usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name Product1Test
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore3  True          1.00GB           40.00GB
```

6. Now try to create a volume from one of the QA machines with size = 2 GB

```
root@photon4 [ ~ ]# docker volume create --name=MyVolume --driver=vsphere -o size=2GB
Error response from daemon: create MyVolume: VolumeDriver.Create: volume size exceeds the max volume size limit
```

The vDVS has restricted user from creating a volume of size > 1 GB.

7.	Try to create volume on datastore other than the one which is set as default

```
root@photon4 [ ~ ]# docker volume create --name=MyVolume@datastore1 --driver=vsphere -o size=2GB
Error response from daemon: create MyVolume@datastore1: VolumeDriver.Create: No create privilege
```

Remember we have set default datastore as datastore3 which as “--allow-create” permissions.

This way, vSphere admin can put storage restrictions on different groups withing organization.
