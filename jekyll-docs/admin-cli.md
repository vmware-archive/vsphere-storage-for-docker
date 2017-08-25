---
title: Admin CLI Reference
---

## Introduction

With the vSphere Docker Volume Service, each ESXi host manages multiple VMs, with
each of them acting as a Docker host. The Docker engine on these hosts communicates with the Docker
volume service to create and delete virtual disks (VMDKs), as well as mounts them as Docker
volumes. These virtual disks may live on any datastore accessible to the ESXi host and are managed
by the Docker user via the Docker CLI. However, the Docker CLI is limited in what visibility it can
provide to the user. Furthermore, it is desirable that an administrator be able to get a global view
of all virtual disks created and in use on the host. For these reasons, an admin CLI is provided through
esxcli that runs on the ESXi host and that provides access to information not visible from the
Docker CLI.

The admin CLI also enables ESX admins to implement access control and basic storage quotas.

Admin CLI commandset as documented below is located under namespace `esxcli storage guestvol`. It supports `--help`  at
every command and sub-command.

The entire commandset is also available for use with `/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py `.
You need to make use of `/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py` instead of `esxcli storage guestvol`.

**NOTE** Access control is not supported for Stateless ESXi , as the code relies on Authorization Config DB (aka "Config DB"), or symlink to the Confg DB, being in /etc/vmware/vmdksops/auth-db.

```
[root@localhost:~] esxcli storage guestvol --help
Usage: esxcli storage guestvol {cmd} [cmd options]

Available Namespaces:
  vmgroup               Administer and monitor volume access control
  config                Init and manage Config DB to enable quotas and access control [EXPERIMENTAL]
  policy                Configure and display storage policy information
  volume                Manage vDVS volumes

Available Commands:
  status                Status of vdvs service
```

The remainder of this document will describe each admin CLI command and provide examples
of their usage.

## Vmgroup

vmgroups allow placing access control restrictions on all Docker storage requests issued from a group of VMs. Administrator can create a vmgroup, place a set of VMs in it (`create` and ``vm add`` subcommands, and then associate this group with a specific set of Datastores and access privileges (`access` and `update` subcommands).

### Help
```bash
[root@localhost:~] esxcli storage guestvol vmgroup --help
Usage: esxcli storage guestvol vmgroup {cmd} [cmd options]

Available Namespaces:
  access                Add or remove Datastore access and quotas for a vmgroup
  vm                    Add, removes, replaces and lists VMs in a vmgroup

Available Commands:
  create                Create a new vmgroup
  ls                    List vmgroups and the VMs they are applied to
  rm                    Delete a vmgroup
  update                Update a vmgroup
```

### Create
A vmgroup named "_DEFAULT" will be created automatically post install.
```
[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
```

The "Default_datastore" field is set to "_VM_DS" for "_DEFAULT" vmgroup. Any volume create from VM which belongs to "_DEFAULT" vmgroup will be created on the datastore where VM resides.

When configuration is initialized with 'config init', the access to _ALL_DS and _VM_DS for all VMs is automatically enabled.
```
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=_DEFAULT
Datastore  Allow_create  Max_volume_size  Total_size
---------  ------------  ---------------  ----------
_ALL_DS    True          Unset            Unset
_VM_DS     True          Unset            Unset
```

Creates a new named vmgroup and optionally assigns VMs. Valid vmgroup name is only allowed to be "[a-zA-Z0-9_][a-zA-Z0-9_.-]*"

"Default_datastore" is a required parameter. The value is either a valid datastore name, or special string "_VM_DS.
After setting the "default_datastore" of a named vmgroup, a full access privilege to the "default_datastore" will be added automatically
and the volume will be created on the "default_datastore" if using short name.
After default_datastore is set, all VMs in the group have full access to it. Also, all volumes created with [short names](/features/tenancy/#Default datastore)
will be placed on this datastore.
Users can modify this privilege using `vmgroup access` subcommands.

Sample:
```
[root@localhost:~] esxcli storage guestvol vmgroup create --name=vmgroup1 --default-datastore=vsanDatastore
vmgroup 'vmgroup1' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.
[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
5d43cc7b-1e34-4b86-af2e-d595e86a1cfa  vmgroup1                             vsanDatastore

[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset
```

The "default_datastore" can be also set to a special value "_VM_DS" during vmgroup create.
```
[root@localhost:~] esxcli storage guestvol vmgroup create --name=vmgroup2 --default-datastore="_VM_DS"
vmgroup 'vmgroup2' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.
[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
4aaee5c9-9778-4299-9eb9-4c59f20a519b  vmgroup2                             _VM_DS

[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup2
Datastore  Allow_create  Max_volume_size  Total_size
---------  ------------  ---------------  ----------
_VM_DS     True          Unset            Unset

```

"Default_datastore" cannot be set to "_ALL_DS". An attempt to do so will generate an error"
```
[root@localhost:~] esxcli storage guestvol vmgroup create --name=vmgroup3 --default-datastore="_ALL_DS"
Cannot use _ALL_DS as default datastore. Please use specific datastore name or _VM_DS special datastore
```

The vmgroup to VM association can be done at create time.

Sample:
```
[root@localhost:~] esxcli storage guestvol vmgroup create --name=vmgroup1 --default-datastore=vsanDatastore --vm-list=ubuntu-VM0.0
vmgroup 'vmgroup1' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.

[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  ------------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
f68c315d-2690-4b63-94c4-f0db09ad458f  vmgroup1                             vsanDatastore      ubuntu-VM0.0

```

#### Help
```
[root@localhost:~] esxcli storage guestvol vmgroup create --help
Usage: esxcli storage guestvol vmgroup create [cmd options]

Description:
  create                Create a new vmgroup

Cmd options:
  --default-datastore=<str>
                        Datastore to be used by default for volumes placement (required)
  --description=<str>   The description of the vmgroup
  --name=<str>          The name of the vmgroup (required)
  --vm-list=<str>       A list of VM names to place in this vmgroup

```
### List
List existing vmgroups, the datastores vmgroups have access to and the VMs assigned.
```
[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  ------------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
f68c315d-2690-4b63-94c4-f0db09ad458f  vmgroup1                             vsanDatastore      ubuntu-VM0.0

```

#### Help
```
[root@localhost:~] esxcli storage guestvol vmgroup ls --help
Usage: esxcli storage guestvol vmgroup ls [cmd options]

Description:
  ls                    List vmgroups and the VMs they are applied to
```

### Update
Update existing vmgroup. This command allows to update "Description" and "Default_datastore" fields, or rename an existing vmgroup.
"Default_datastore" is either a valid datastore name or a special value "_VM_DS".
After changing the "default_datastore" for a vmgroup, a full access privilege to the new "default_datastore" will be created automatically, and the existing access privilege to old "default_datastore" will remain. User can remove the access privilege to old "default_datastore" if not needed using `vmgroup access rm` subcommands.
Sample:
```
[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  ------------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
f68c315d-2690-4b63-94c4-f0db09ad458f  vmgroup1                             vsanDatastore      ubuntu-VM0.0

[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset

[root@localhost:~] esxcli storage guestvol vmgroup update --name=vmgroup1 --description="New description of vmgroup1" --new-name=new-vmgroup1 --defau
lt-datastore=sharedVmfs-0
vmgroup modify succeeded

[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name          Description                  Default_datastore  VM_list
------------------------------------  ------------  ---------------------------  -----------------  ------------
11111111-1111-1111-1111-111111111111  _DEFAULT      This is a default vmgroup    _VM_DS
f68c315d-2690-4b63-94c4-f0db09ad458f  new-vmgroup1  New description of vmgroup1  sharedVmfs-0       ubuntu-VM0.0
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=new-vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
sharedVmfs-0   True          Unset            Unset
vsanDatastore  True          Unset            Unset
```
Please use the test suggested above, for "create".

#### Help
```
[root@localhost:~] esxcli storage guestvol vmgroup update --help
Usage: esxcli storage guestvol vmgroup update [cmd options]

Description:
  update                Update a vmgroup

Cmd options:
  --default-datastore=<str>
                        Datastore to be used by default for volumes placement
  --description=<str>   The new description of the vmgroup
  --name=<str>          The name of the vmgroup (required)
  --new-name=<str>      The new name of the vmgroup

```

### Remove
Remove a vmgroup, optionally all volumes for a vmgroup can be removed as well.

Sample:
```
[root@localhost:~] esxcli storage guestvol vmgroup rm --name=new-vmgroup1 --remove-volumes
All Volumes will be removed. vmgroup rm succeeded

[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
```

In MultiNode mode, VMs from different hosts can be a part of a vmgroup.
When in this mode, vmgroup which has member VMs in it cannot be directly deleted.
First remove the VMs individually from the vmgroup using admin cli
on the same host on which the VM resides.
Then remove the vmgroup.
#### Help
```
[root@localhost:~] esxcli storage guestvol vmgroup rm --help
Usage: esxcli storage guestvol vmgroup rm [cmd options]

Description:
  rm                    Delete a vmgroup

Cmd options:
  --name=<str>          The name of the vmgroup (required)
  --remove-volumes      BE CAREFUL: Removes this vmgroup volumes when removing a vmgroup
```

### Virtual Machine

#### Add
Add a VM to a vmgroup. A VM can only access the datastores for the vmgroup it is assigned to.
VMs can be assigned to only one vmgroup at a time.
```
[root@localhost:~] esxcli storage guestvol vmgroup vm add --name=vmgroup1 --vm-list=ubuntu-VM1.0
vmgroup vm add succeeded
[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------------------------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
443583f6-afd0-4fa7-bfa7-9116384431f3  vmgroup1                             vsanDatastore      ubuntu-VM0.0,ubuntu-VM1.0

```

#### List
```
[root@localhost:~] esxcli storage guestvol vmgroup vm ls --name vmgroup1
Uuid                                  Name
------------------------------------  ------------
564d9bc4-bfd6-b648-935b-6353fb6dc1e7  ubuntu-VM0.0
564d171f-2de0-95b7-f9b8-9d9c42841b7a  ubuntu-VM1.0
```

#### Remove
Remove a VM from a vmgroup's list of VMs. VM will no longer be able to access the volumes created for the vmgroup.
```
[root@localhost:~] esxcli storage guestvol vmgroup vm rm --name=vmgroup1 --vm-list=ubuntu-VM1.0
vmgroup vm rm succeeded
[root@localhost:~] esxcli storage guestvol vmgroup vm ls --name vmgroup1
Uuid                                  Name
------------------------------------  ------------
564d9bc4-bfd6-b648-935b-6353fb6dc1e7  ubuntu-VM0.0

```

### Replace
Replace VMs from a vmgroup's list of VMs. VMs which are replaced will no longer be able to access the volumes created for the vmgroup.
```
[root@localhost:~] esxcli storage guestvol vmgroup vm ls --name vmgroup1
Uuid                                  Name
------------------------------------  ------------
564d9bc4-bfd6-b648-935b-6353fb6dc1e7  ubuntu-VM0.0
[root@localhost:~] esxcli storage guestvol vmgroup vm replace --name=vmgroup1 --vm-list=ubuntu-VM1.0
vmgroup vm replace succeeded
[root@localhost:~] esxcli storage guestvol vmgroup vm ls --name vmgroup1
Uuid                                  Name
------------------------------------  ------------
564d171f-2de0-95b7-f9b8-9d9c42841b7a  ubuntu-VM1.0
```

Note: If the VMs have volumes attached (containers running), their membership change i.e. changing the vmgroup to which
they belong is not permitted. Make sure no volumes are attached.
To do so:
1. Get the list of containers running. (docker ps)
2. If the container has any vDVS volume mounted (docker inspect container_name), stop the container.
3. Ensure that the dvs volumes have status detached (docker volume inspect)

#### Help
```
[root@localhost:~] esxcli storage guestvol vmgroup vm --help
Usage: esxcli storage guestvol vmgroup vm {cmd} [cmd options]

Available Commands:
  add                   Add VM(s) to a vmgroup
  ls                    list VMs in a vmgroup
  replace               Replace VM(s) for a vmgroup
  rm                    Remove VM(s) from a vmgroup

```

### Access
Change the access control for a vmgroup.
This includes ability to grant privileges & set resource consumption limits for a datastore.

#### Help
```bash
[root@localhost:~] esxcli storage guestvol vmgroup access --help
Usage: esxcli storage guestvol vmgroup access {cmd} [cmd options]

Available Commands:
  add                   Add a datastore access for a vmgroup
  ls                    List access for a vmgroup
  rm                    Remove datastore access for a vmgroup
  set                   Modify datastore access for a vmgroup

```

#### Add
Grants datastore access to a vmgroup.
Valid value for "datastore" includes the name of valid datastores in the ESX host , special value "_VM_DS" or "_ALL_DS".
When DS is set to _VM_DS, access to datastore where vm lives is allowed for vms in vmgroup.
When DS is set to _ALL_DS, access to any datastore which has not been granted access to explicitly is allowed for vms in vmgroup.
Sample:

```bash
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset

[root@localhost:~] esxcli storage guestvol vmgroup access add --name=vmgroup1 --datastore=sharedVmfs-0  --volume-maxsize=500MB --volume-totalsize=1GB
vmgroup access add succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
sharedVmfs-0   False         500.00MB         1.00GB
vsanDatastore  True          Unset            Unset

```

By default no "allow_create" right is given

```bash
[root@localhost:~] esxcli storage guestvol vmgroup access add --name=vmgroup1 --datastore=sharedVmfs-0  --volume-maxsize=500MB --volume-totalsize=1GB
vmgroup access add succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
sharedVmfs-0   False         500.00MB         1.00GB
vsanDatastore  True          Unset            Unset
```

"allow_create" right is given when you run the command with "--allow-create" flag.
```bash
[root@localhost:~] esxcli storage guestvol vmgroup access add --name=vmgroup1 --datastore=sharedVmfs-0  --volume-maxsize=500MB --volume-totalsize=1GB
 --allow-create
vmgroup access add succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
sharedVmfs-0   True          500.00MB         1.00GB
vsanDatastore  True          Unset            Unset
```
For _VM_DS and _ALL_DS special DS names, "--volume-totalsize" can also be set.

When "--volume-totalsize" is set for "_VM_DS", it means the total volume size on datastore where VM lives cannot exceed the value specified by "--volume-totalsize". For example, if "--volume-totalsize" is set to "1GB" for "_VM_DS" and VM lives in "datastore2", where the total volume size is already 800MB and user tries to create another volume with 500MB will not be allowed. However, if the VM is moved to "datastore3", where the total volume size is 500MB and user tries to create another volume with 500MB will be allowed.

```
[root@localhost:~] esxcli storage guestvol vmgroup access add --name=vmgroup1 --datastore=_VM_DS  --volume-maxsize=500MB --volume-totalsize=1GB --all
ow-create
vmgroup access add succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset
_VM_DS         True          500.00MB         1.00GB
```

When "--volume-totalsize" is set for "_ALL_DS", it means the total volume size on any datastore which has not been given access to explicitly cannot exceed the value specified by "--volume-totalsize". In the following example, "vmgroup1" has been given access to "datastore1" and "_ALL_DS". The "Total_size" for "datastore1" is "Unset", which means no limit. The "Total_size" for "_ALL_DS" is "1GB". So for "datastore1", there is no limit on the total volume size. However, for any datastore other than "datastore1", the total volume size on that datastore cannot exceed 1GB.

```
[root@localhost:~] esxcli storage guestvol vmgroup access add --name=vmgroup1 --datastore=_ALL_DS  --volume-maxsize=500MB --volume-totalsize=1GB --al
low-create
vmgroup access add succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset
_ALL_DS        True          500.00MB         1.00GB
```


##### Help
```bash
[root@localhost:~] esxcli storage guestvol vmgroup access add --help
Usage: esxcli storage guestvol vmgroup access add [cmd options]

Description:
  add                   Add a datastore access for a vmgroup

Cmd options:
  --allow-create        Allow create and delete on datastore if set
  --datastore=<str>     Datastore name (required)
  --name=<str>          The name of the vmgroup (required)
  --volume-maxsize=<str>
                        Maximum size of the volume that can be created
  --volume-totalsize=<str>
                        Maximum total size of all volume that can be created on the datastore for this vmgroup
```

#### List
List the current access control granted to a vmgroup.

When displaying the result keep in mind:

- For capacity Unset indicates no limits

```bash
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset
_ALL_DS        True          500.00MB         1.00GB
```

##### Help
```bash
[root@localhost:~] esxcli storage guestvol vmgroup access ls --help
Usage: esxcli storage guestvol vmgroup access ls [cmd options]

Description:
  ls                    List access for a vmgroup

Cmd options:
  --name=<str>          Vmgroup name (required)
```

#### Remove
Remove access to a datastore for a vmgroup.
Removing of access privilege to "default_datastore" is not suported
```bash
[root@localhost:~] esxcli storage guestvol vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  ------------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
443583f6-afd0-4fa7-bfa7-9116384431f3  vmgroup1                             vsanDatastore      ubuntu-VM1.0

[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset
_ALL_DS        True          500.00MB         1.00GB

[root@localhost:~]
[root@localhost:~] esxcli storage guestvol vmgroup access rm --name=vmgroup1 --datastore=vsanDatastore
Removing of access privilege to 'default_datastore' is not supported
[root@localhost:~] esxcli storage guestvol vmgroup access rm --name=vmgroup1 --datastore=_ALL_DS
vmgroup access rm succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset

```

##### Help
```bash
[root@localhost:~] esxcli storage guestvol vmgroup  access rm --help
Usage: esxcli storage guestvol vmgroup access rm [cmd options]

Description:
  rm                    Remove datastore access for a vmgroup

Cmd options:
  --datastore=<str>     Datastore name (required)
  --name=<str>          The name of the vmgroup (required)

```

#### Set
Set command allows to change the existing access control in place for a vmgroup.

Sample:

```shell
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          Unset            Unset
[root@localhost:~] esxcli storage guestvol vmgroup access set --name=vmgroup1 --datastore=vsanDatastore --allow-create=True  --volume-maxsize=1000MB
vmgroup access set succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          1000.00MB        Unset
```

"--volume-totalsize" can also be set to the value other than unlimit when add privilege for special value "_VM_DS" and "_ALL_DS".
```
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          1000.00MB        2.00GB
_ALL_DS     True          500.00MB         Unset
_VM_DS      True          Unset            Unset


[root@localhost:~] esxcli storage guestvol vmgroup access set --name=vmgroup1 --datastore=_VM_DS  --volume-maxsize=1GB --volume-totalsize=2GB
vmgroup access set succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access set --name=vmgroup1 --datastore=_ALL_DS  --volume-maxsize=500MB --volume-totalsize=1GB
vmgroup access set succeeded
[root@localhost:~] esxcli storage guestvol vmgroup access ls --name=vmgroup1
Datastore      Allow_create  Max_volume_size  Total_size
-------------  ------------  ---------------  ----------
vsanDatastore  True          1000.00MB        Unset
_ALL_DS        True          500.00MB         1.00GB
_VM_DS         True          1.00GB           2.00GB

```

##### Help
```
[root@localhost:~] esxcli storage guestvol vmgroup access set --help
Usage: esxcli storage guestvol vmgroup access set [cmd options]

Description:
  set                   Modify datastore access for a vmgroup

Cmd options:
  --allow-create        Allow create and delete on datastore if set to True. This value can also be set to False by user.
  --datastore=<str>     Datastore name (required)
  --name=<str>          The name of the vmgroup (required)
  --volume-maxsize=<str>
                        Maximum size of the volume that can be created
  --volume-totalsize=<str>
                        Maximum total size of all volume that can be created on the datastore for this vmgroup

```

## Volume

#### Help
```bash
[root@localhost:~] esxcli storage guestvol volume ls --help
Usage: esxcli storage guestvol volume ls [cmd options]

Description:
  ls                    List volumes

Cmd options:
  --vmgroup=<str>       Displays volumes for a given vmgroup
```

#### List All
List all properties for all Docker volumes that exist on datastores accessible to the host.

```bash
[root@localhost:~] esxcli storage guestvol volume ls
Volume  Datastore     VMGroup   Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By    Created Date
------  ------------  --------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ------------  ------------------------
vol1    sharedVmfs-0  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:08 2017
vol2    sharedVmfs-0  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:12 2017
```

Note that the `Policy` column shows the named VSAN storage policy created with the same tool
(vmdkops_admin.py).  Since these example virtual disks live on a VMFS datastore they do not have a storage
policy and show up as `N/A'.

Note that the `VMGroup` column shows the vmgroup by which the volume was created. If the vmgroup which created the volume has been removed, the `VMGroup` column shows up as 'N/A'. See the following example:

```bash
[root@localhost:~] esxcli storage guestvol volume ls
Volume  Datastore      VMGroup   Capacity  Used  Filesystem  Policy          Disk Format  Attached-to  Access      Attach-as               Created By    Created Date
------  -------------  --------  --------  ----  ----------  --------------  -----------  -----------  ----------  ----------------------  ------------  ------------------------
vol1    sharedVmfs-0   _DEFAULT  100MB     13MB  ext4        N/A             thin         detached     read-write  independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:08 2017
vol2    sharedVmfs-0   _DEFAULT  100MB     13MB  ext4        N/A             thin         detached     read-write  independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:12 2017
vol3    vsanDatastore  N/A       100MB     80MB  ext4        [VSAN default]  thin         detached     read-write  independent_persistent  ubuntu-VM1.0  Mon Aug 21 04:38:06 2017
vol4    vsanDatastore  N/A       100MB     80MB  ext4        [VSAN default]  thin         detached     read-write  independent_persistent  ubuntu-VM1.0  Mon Aug 21 04:38:10 2017
```

#### List limited information about volumes

Only shows Volume, Capacity, Disk Format and Attached-to fields
```bash
[root@localhost:~] esxcli storage guestvol volume shortls
Volume  Capacity  Disk Format  Attached-to
------  --------  -----------  -----------
vol1    100MB     thin         detached
vol2    100MB     thin         detached
vol3    100MB     thin         detached
vol4    100MB     thin         detached
```

### Set
Modify attribute settings on a given volume. The volume is identified by its name, vmgroup_name which the volume belongs to and datastore,
for example if the volume name is `container-vol` then the volume is specified as "container-vol@datastore-name".
The attributes to set/modify are specified as a comma separated list as "<attr1>=<value>, <attr2>=<value>....". For example,
a command line would look like this.

```bash
$ esxcli storage guestvol volume set --volume=<volume@datastore> --vmgroup=<vmgroup_name> --options="<attr1>=<value>, <attr2>=<value>, ..."
```

The volume attributes are set and take effect only the next time the volume attached to a VM. The changes do not impact any VM
thats currently using the volume. For the present, only the "access" attribute is supported to be modified via this command, and
can be set to either of the allowed values "read-only" or "read-write".

Set command allows the admin to enforce a volume to be read-only.
This removes the need to depend on [Docker's run command options for volume access](https://docs.docker.com/engine/tutorials/dockervolumes/) (``` docker run -v /vol:/vol:ro```).

A sample use case:

1. Create a volume, attach to a container (default is read-write).
2. Master the volume with libraries commonly used by the target application (or a cluster of apps that form a docker app bundle).
3. Use admin CLI to flip the access attribute to read-only.
4. Make those libraries available to the containers in the app bundle and they can all share the same libraries.

The container images themselves can be smaller as they share the libs and possibly binaries from read-only volumes.

Example:
```bash
[root@localhost:~] esxcli storage guestvol volume ls
Volume  Datastore      VMGroup   Capacity  Used  Filesystem  Policy          Disk Format  Attached-to  Access      Attach-as               Created By    Created Date
------  -------------  --------  --------  ----  ----------  --------------  -----------  -----------  ----------  ----------------------  ------------  ------------------------
vol1    sharedVmfs-0   _DEFAULT  100MB     13MB  ext4        N/A             thin         detached     read-write  independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:08 2017
vol2    sharedVmfs-0   _DEFAULT  100MB     13MB  ext4        N/A             thin         detached     read-write  independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:12 2017
vol3    vsanDatastore  N/A       100MB     80MB  ext4        [VSAN default]  thin         detached     read-write  independent_persistent  ubuntu-VM1.0  Mon Aug 21 04:38:06 2017
vol4    vsanDatastore  N/A       100MB     80MB  ext4        [VSAN default]  thin         detached     read-write  independent_persistent  ubuntu-VM1.0  Mon Aug 21 04:38:10 2017
[root@localhost:~] esxcli storage guestvol volume set --volume=vol1@sharedVmfs-0 --vmgroup=_DEFAULT --options="access=read-only"
Successfully updated settings for vol1@sharedVmfs-0
[root@localhost:~] esxcli storage guestvol volume ls
Volume  Datastore      VMGroup   Capacity  Used  Filesystem  Policy          Disk Format  Attached-to  Access      Attach-as               Created By    Created Date
------  -------------  --------  --------  ----  ----------  --------------  -----------  -----------  ----------  ----------------------  ------------  ------------------------
vol1    sharedVmfs-0   _DEFAULT  100MB     13MB  ext4        N/A             thin         detached     read-only   independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:08 2017
vol2    sharedVmfs-0   _DEFAULT  100MB     13MB  ext4        N/A             thin         detached     read-write  independent_persistent  ubuntu-VM0.0  Mon Aug 21 04:36:12 2017
vol3    vsanDatastore  N/A       100MB     80MB  ext4        [VSAN default]  thin         detached     read-write  independent_persistent  ubuntu-VM1.0  Mon Aug 21 04:38:06 2017
vol4    vsanDatastore  N/A       100MB     80MB  ext4        [VSAN default]  thin         detached     read-write  independent_persistent  ubuntu-VM1.0  Mon Aug 21 04:38:10 2017

```


## Policy

Create, configure and show the VSAN policy names and their corresponding VSAN policy strings. Also show whether or not they are in use.

#### Help
```bash
[root@localhost:~] esxcli storage guestvol policy --help
Usage: esxcli storage guestvol policy {cmd} [cmd options]

Available Commands:
  create                Create a storage policy
  ls                    List storage policies and volumes using those policies
  rm                    Remove a storage policy
  update                Update the definition of a storage policy and all VSAN objects using that policy
```

#### Create

Create a VSAN storage policy.

```bash
[root@localhost:~] esxcli storage guestvol policy create --name some-policy --content '(("proportionalCapacity" i0)("hostFailuresToTolerate" i0))'
Successfully created policy: some-policy
```

Note that the VSAN storage policy string given with `--content` is a standard VSAN storage policy
string.  Please refer to the [VSAN documentation](https://pubs.vmware.com/vsphere-60/index.jsp?topic=%2Fcom.vmware.vcli.ref.doc%2Fesxcli_vsan.html)
for storage policy options.

#### List

List all VSAN storage policies.

```bash
[root@localhost:~] esxcli storage guestvol policy ls
Policy Name  Policy Content                                              Active
-----------  ----------------------------------------------------------  ------
some-policy  (("proportionalCapacity" i0)("hostFailuresToTolerate" i0))  Unused
```

When creating a virtual disk using `docker volume create`, the policy name should be given with the `-o`
option such as `docker volume create --driver=vsphere --name=some-vol -o vsan-policy-name=some-policy`.
The number of virtual disks using the policy will then show up in the `Active` column.

#### Update

Update a VSAN storage policy.

This command will update a VSAN storage policy for all virtual disks currently using this policy. If
the command fails, the number of virtual disks that were successfully updated and the number that
failed to update will be shown. The names of the virtual disks that failed to update will be logged
so that manual action can be taken.

```bash
[root@localhost:~] esxcli storage guestvol policy update --name some-policy --content '(("proportionalCapacity" i1))'
Successfully updated policy some-policy
```

#### Remove  (`rm`)

Remove a VSAN storage policy. Note that a storage policy cannot be removed if it is currently in use
by one or more virtual disks.

The ability to list which virtual disks are using a specific storage policy, change storage policies
for a virtual disk, and reset virtual disks to the default storage policy is a necessary
enhancement tracked [here](https://github.com/vmware/docker-volume-vsphere/issues/577).

```bash
[root@localhost:~] esxcli storage guestvol policy rm --name=some-policy
Successfully removed policy: some-policy
```

## `Config` (Authorization DB configuration)

**THIS FEATURE IS EXPERIMENTAL**

Creates, removes, moves and reports on status of Authorization config DB (referred to as `Config DB`). Config DB keeps authorization information - vmgroups, datastore access control, quota information -  and without initializing it no access control is supported. Also, before Config DB is initialized, any attempt to configure access control will fail, e.g.
```
[root@localhost:~] vmdkops_admin vmgroup create --name MY
Internal Error(Error: Please init configuration in vmdkops_admin before trying to change it)
```

If the Config DB is not initialized, Docker Volume Service will use  "Config DB NotConfigured" Mode, when any request to create, remove, mount or unmount Docker volume is accepted.

After initialization the service can use SingleNode mode - when the DB itself is located on the local ESXi node in `/etc/vmware/vmdkps/auth-db` file, or MultiNode mode - when the above location is a symlin to a shared datastore location.

In SingleNode mode all vmgroups and authorization control is local for each ESXi node, and node do not share this information.

In MultiNode mode, VSphere Docker Volume Service Authorization Config DB needs to be initialized on each ESXi host (`config init --datastore=<ds>`, and the nodes will share the authoration control.

#### Init (`config init`)

Initializing the config is optional. If the config is not initialized, there will be no access control and all `vmgroup` commands will fail with appropriate messages.

Before configuring access control or quotas, the config needs to be inited to either SingleNode (`init --local`) or MultiNode (`init --datastore=ds_name`) mode.

```
[root@localhost:~] esxcli storage guestvol config init --help
Usage: esxcli storage guestvol config init [cmd options]

Description:
  init                  Init and manage Config DB to enable quotas and access control [EXPERIMENTAL]

Cmd options:
  --datastore=<str>     Config file will be placed on a datastore
  --force               Force operation, ignore warnings
  --local               Allows local (SingleNode) Init
```

Example:

```
[root@localhost:~] vmdkops_admin status
Version: 0.12.fea683a-0.0.1
Status: Running
DB_LocalPath: /etc/vmware/vmdkops/auth-db
DB_SharedLocation: n/a
DB_Mode: NotConfigured (no local DB or symlink)  <===== NOT CONFIGURED
Pid: 5979199
Port: 1019
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: DEBUG

[root@localhost:~] vmdkops_admin config init --local
Creating new DB at /etc/vmware/vmdkops/auth-db
Restarting the vmdkops service to pick up new configuration
Stopping vmdkops-opsd with PID=5979199
vmdkops-opsd is not running
Starting vmdkops-opsd
vmdkops-opsd is running pid=5979684

[root@localhost:~] esxcli storage guestvol status
=== Service:
Version: 0.14.ff1d8d4-0.0.1
Status: Running
Pid: 607737
Port: 1019
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: INFO
=== Authorization Config DB:
DB_LocalPath: /etc/vmware/vmdkops/auth-db
DB_Mode: SingleNode (local DB exists)     <==== LOCAL configuration
DB_SharedLocation: N/A

```


#### Remove (`config rm`)

Allows to remove local configuration DB. Since this is a destructive operation, admin needs to type both `--local` flag (to confirm it's local only operation and does not impact shared database, if any) , and `--confirm` flag to confirm that she actually wants to delete the local Config DB.

Running `esxcli storage guestvol config rm` with no flags prints an explanation on how to remove the shared config DB, if any.

```
[root@localhost:~] esxcli storage guestvol config rm --help
Usage: esxcli storage guestvol config rm [cmd options]

Description:
  rm                    Init and manage Config DB to enable quotas and access control [EXPERIMENTAL]

Cmd options:
  --confirm             Explicitly confirm the operation
  --local               Remove only local link or local DB
  --no-backup           Do not create DB backup before removing
  --unlink              Remove the local link to shared DB
```


#### Status

To get config DB  status, use  `esxcli storage guestvol status` command.

#### Move (`config mv`)

[Not implemented yet] Allows to relocate config DB between datastores.

## Status

Show config and run-time information about the service.

```
[root@localhost:~] esxcli storage guestvol status --help
Usage: esxcli storage guestvol status [cmd options]

Description:
  status                Status of vdvs service

Cmd options:
  --fast                Skip some of the data collection (port, version)
```


```bash
[root@localhost:~] time esxcli storage guestvol status
=== Service:
Version: 0.14.ff1d8d4-0.0.1
Status: Running
Pid: 607737
Port: 1019
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: INFO
=== Authorization Config DB:
DB_Mode: SingleNode (local DB exists)
DB_SharedLocation: N/A
DB_LocalPath: /etc/vmware/vmdkops/auth-db
real	0m 2.67s
user	0m 0.31s
sys	0m 0.00s
```

 Some of the information retrieval may be slow (e.g. VIB version (`Version` field) # or VMCI port number (`Port` field). `--fast` flag skips slow data collection and prints `?` for fields with no information.

```bash
[root@localhost:~] time esxcli storage guestvol status --fast
=== Service:
Version: ?
Status: Running
Pid: 607737
Port: ?
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: INFO
=== Authorization Config DB:
DB_LocalPath: /etc/vmware/vmdkops/auth-db
DB_SharedLocation: N/A
DB_Mode: SingleNode (local DB exists)
real	0m 0.95s
user	0m 0.34s
sys	0m 0.00s
```