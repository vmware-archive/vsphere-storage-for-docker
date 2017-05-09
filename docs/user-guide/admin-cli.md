
# Introduction

With the vSphere Docker Volume Service, each ESXi host manages multiple VMs, with
each of them acting as a Docker host. The Docker engine on these hosts communicates with the Docker
volume service to create and delete virtual disks (VMDKs), as well as mounts them as Docker
volumes. These virtual disks may live on any datastore accessible to the ESXi host and are managed
by the Docker user via the Docker CLI. However, the Docker CLI is limited in what visibility it can
provide to the user. Furthermore, it is desirable that an administrator be able to get a global view
of all virtual disks created and in use on the host.  For these reasons, an admin CLI has been
created that runs on the ESXi host and that provides access to information not visible from the
Docker CLI.

The admin CLI also enables ESX admins to implement access control and basic storage quotas.

Admin CLI is located at `/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py`. It supports `--help`  at
every command and sub-command. e.g.

**NOTE** Access control is not supported for Stateless ESXi , as the code relies on Authorization Config DB (aka "Config DB"), or symlink to the Confg DB, being in /etc/vmware/vmdksops/auth-db.

```
[root@localhost:~] alias vmdkops_admin=/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py
[root@localhost:~] vmdkops_admin --help
usage: vmdkops_admin.py [-h] {volume,policy,status,config,vmgroup} ...

vSphere Docker Volume Service admin CLI

optional arguments:
  -h, --help            show this help message and exit

Manage VMDK-based Volumes for Docker:

  {volume,policy,status,config,vmgroup}
                        action
    volume              Manipulate volumes
    policy              Configure and display storage policy information
    status              Show the status of the vmdk_ops service
    config              Init and manage Config DB which enables quotas and
                        access control
    vmgroup            Administer and monitor volume access control
[root@localhost:~] vmdkops_admin volume --help
usage: vmdkops_admin.py volume [-h] {set,ls} ...

optional arguments:
  -h, --help  show this help message and exit

Manipulate volumes:

  {set,ls}    action
    set       Edit settings for a given volume
    ls        List volumes
```

It also prompts for available choices, e.g.
```
[root@localhost:~] vmdkops_admin volume hm
usage: vmdkops_admin.py volume [-h] {set,ls} ...
vmdkops_admin.py volume: error: invalid choice: 'hm' (choose from 'set', 'ls')
```


The remainder of this document will describe each admin CLI command and provide examples
of their usage.

## Vmgroup

vmgroups allow placing access control restrictions on all Docker storage requests issued from a group of VMs. Administrator can create a vmgroup, place a set of VMs in it (`create` and ``vm add`` subcommands, and then associate this group with a specific set of Datastores and access privileges (`access` and `update` subcommands).

### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup -h
usage: vmdkops_admin.py vmgroup [-h] {create,vm,update,access,ls,rm} ...

positional arguments:
  {create,vm,update,access,ls,rm}
    create              Create a new vmgroup
    vm                  Add, removes and lists VMs in a vmgroup
    update              Update an existing vmgroup
    access              Add or remove Datastore access and quotas for a vm-
                        group
    ls                  List vmgroups and the VMs they are applied to
    rm                  Delete a vmgroup

optional arguments:
  -h, --help            show this help message and exit
```

### Create
A vmgroup named "_DEFAULT" will be created automatically post install.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
```

The "Default_datastore" field is set to "_VM_DS" for "_DEFAULT" vmgroup. Any volume create from VM which belongs to "_DEFAULT" vmgroup will be created on the datastore where VM resides.

When configuration is initialized with 'config init', the access to _ALL_DS and _VM_DS for all VMs is automatically enabled.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=_DEFAULT
Datastore  Allow_create  Max_volume_size  Total_size
---------  ------------  ---------------  ----------
_ALL_DS   True          Unset            Unset
_VM_DS    True          Unset            Unset
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
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=vmgroup1 --default-datastore=datastore1
vmgroup 'vmgroup1' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
9de84179-6894-44ad-b444-470e8619a5ed  vmgroup1                             datastore1

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
```

The "default_datastore" can be also set to a special value "_VM_DS" during vmgroup create.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                  Default_datastore  VM_list
------------------------------------  --------  ---------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is the default vmgroup  _VM_DS

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=vmgroup1 --default-datastore="_VM_DS"
vmgroup 'vmgroup1' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                  Default_datastore  VM_list
------------------------------------  --------  ---------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is the default vmgroup  _VM_DS
30545fdc-20e0-409a-8330-6ebe027fcc34  vmgroup1                               _VM_DS

[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore  Allow_create  Max_volume_size  Total_size
---------  ------------  ---------------  ----------
_VM_DS    True          Unset            Unset

```

"Default_datastore" cannot be set to "_ALL_DS". An attempt to do so will generate an error"
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=vmgroup2 --default-datastore="_ALL_DS"
Cannot use _ALL_DS as default datastore. Please use specific datastore name or _VM_DS special datastore
```

The vmgroup to VM association can be done at create time.

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=vmgroup1 --default-datastore=datastore1 --vm-list=photon7
vmgroup 'vmgroup1' is created. Do not forget to run 'vmgroup vm add' to add vm to vmgroup.

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
04423382-efa4-4525-b0a6-16b98ce38f0f  vmgroup1                             datastore1         photon7

```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create -h
usage: vmdkops_admin.py vmgroup create [-h] --name NAME --default-datastore
                                       DEFAULT_DATASTORE
                                       [--description DESCRIPTION]
                                       [--vm-list vm1, vm2, ...]

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the vmgroup
  --default-datastore DEFAULT_DATASTORE
                        Datastore to be used by default for volumes placement
  --description DESCRIPTION
                        The description of the vmgroup
  --vm-list vm1, vm2, ...
                        A list of VM names to place in this vmgroup

```
### List
List existing vmgroups, the datastores vmgroups have access to and the VMs assigned.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
04423382-efa4-4525-b0a6-16b98ce38f0f  vmgroup1                             datastore1         photon7

```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls -h
usage: vmdkops_admin.py vmgroup ls [-h]

optional arguments:
  -h, --help  show this help message and exit
```

### Update
Update existing vmgroup. This command allows to update "Description" and "Default_datastore" fields, or rename an existing vmgroup.
"Default_datastore" is either a valid datastore name or a special value "_VM_DS".
After changing the "default_datastore" for a vmgroup, a full access privilege to the new "default_datastore" will be created automatically, and the existing access privilege to old "default_datastore" will remain. User can remove the access privilege to old "default_datastore" if not needed using `vmgroup access rm` subcommands.
Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
0767f5f8-73de-4382-8c38-1935bb636ef4  vmgroup1                             datastore1         photon7

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup update --name=vmgroup1 --description="New description of vmgroup1" --new-name=new-vmgroup1 --default-datastore=datastore2
vmgroup modify succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name          Description                  Default_datastore  VM_list
------------------------------------  ------------  ---------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT      This is a default vmgroup    _VM_DS
0767f5f8-73de-4382-8c38-1935bb636ef4  new-vmgroup1  New description of vmgroup1  datastore2         photon7

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=new-vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
datastore2  True          Unset            Unset
```
Please use the test suggested above, for "create".

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup  update -h
usage: vmdkops_admin.py vmgroup update [-h] --name NAME
                                        [--default-datastore DEFAULT_DATASTORE]
                                        [--description DESCRIPTION]
                                        [--new-name NEW_NAME]

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the vmgroup
  --default-datastore DEFAULT_DATASTORE
                        The name of the datastore to be used by default for
                        volumes placement
  --description DESCRIPTION
                        The new description of the vmgroup
  --new-name NEW_NAME   The new name of the vmgroup

```

### Remove
Remove a vmgroup, optionally all volumes for a vmgroup can be removed as well.

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup rm --name=vmgroup1 --remove-volumes
All Volumes will be removed
vmgroup rm succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                 Default_datastore  VM_list
------------------------------------  --------  --------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup
```

In MultiNode mode, VMs from different hosts can be a part of a vmgroup.
When in this mode, vmgroup which has member VMs in it cannot be directly deleted.
First remove the VMs individually from the vmgroup using admin cli
on the same host on which the VM resides.
Then remove the vmgroup.
#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup rm -h
usage: vmdkops_admin.py vmgroup rm [-h] --name NAME [--remove-volumes]

optional arguments:
  -h, --help        show this help message and exit
  --name NAME       The name of the vmgroup
  --remove-volumes  BE CAREFUL: Removes this vmgroup volumes when removing a
                    vmgroup

```

### Virtual Machine

#### Add
Add a VM to a vmgroup. A VM can only access the datastores for the vmgroup it is assigned to.
VMs can be assigned to only one vmgroup at a time.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  --------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vmgroup
6d810c66-ffc7-47c8-8870-72114f86c2cf  vmgroup1                                                 photon6

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm add --name=vmgroup1 --vm-list=photon7
vmgroup vm add succeeded

```

#### List
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm ls --name=vmgroup1
Uuid                                  Name
------------------------------------  --------
564d5849-b135-1259-cc73-d2d3aa1d9b8c  photon6
564d99a2-4097-9966-579f-3dc4082b10c9  photon7
```

#### Remove
Remove a VM from a vmgroup's list of VMs. VM will no longer be able to access the volumes created for the vmgroup.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm rm --name=vmgroup1 --vm-list=photon7
vmgroup vm rm succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm ls --name=vmgroup1
Uuid                                  Name
------------------------------------  --------
564d5849-b135-1259-cc73-d2d3aa1d9b8c  photon6

```

### Replace
Replace VMs from a vmgroup's list of VMs. VMs which are replaced will no longer be able to access the volumes created for the vmgroup.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm ls --name=vmgroup1
Uuid                                  Name
------------------------------------  --------
564d5849-b135-1259-cc73-d2d3aa1d9b8c  photon6

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm replace --name=vmgroup1 --vm-list=photon7
vmgroup vm replace succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm ls --name=vmgroup1
Uuid                                  Name
------------------------------------  --------
564d99a2-4097-9966-579f-3dc4082b10c9  photon7
```

Note: If the VMs have volumes attached (containers running), their membership change i.e. changing the vmgroup to which
they belong is not permitted. Make sure no volumes are attached.
To do so:
1. Get the list of containers running. (docker ps)
2. If the container has any vDVS volume mounted (docker inspect container_name), stop the container.
3. Ensure that the dvs volumes have status detached (docker volume inspect)

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup vm -h
usage: vmdkops_admin.py vmgroup vm [-h] {rm,add,ls,replace} ...

positional arguments:
  {rm,add,ls,replace}
    rm                 Remove VM(s) from a vmgroup
    add                Add a VM(s) to a vmgroup
    ls                 list VMs in a vmgroup
    replace            Replace VM(s) for a vmgroup

optional arguments:
  -h, --help           show this help message and exit

```

### Access
Change the access control for a vmgroup.
This includes ability to grant privileges & set resource consumption limits for a datastore.

#### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access -h
usage: vmdkops_admin.py vmgroup access [-h] {rm,add,set,ls} ...

positional arguments:
  {rm,add,set,ls}
    rm             Remove all access to a datastore for a vmgroup
    add            Add a datastore access for a vmgroup
    set            Modify datastore access for a vmgroup
    ls             List all access info for a vmgroup

optional arguments:
  -h, --help       show this help message and exit

```

#### Add
Grants datastore access to a vmgroup.
Valid value for "datastore" includes the name of valid datastores in the ESX host , special value "_VM_DS" or "_ALL_DS".
When DS is set to _VM_DS, access to vm_datastore where vm lives is allowed for vms in vmgroup.
When DS is set to _ALL_DS, access to all DS is allowed for vms in vmgroup.
Sample:

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset

[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=vmgroup1 --datastore=datastore2  --volume-maxsize=500MB --volume-totalsize=1GB
vmgroup access add succeeded
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
datastore2  False         500.00MB         1.00GB

```

By default no "allow_create" right is given

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=vmgroup1 --datastore=datastore2  --volume-maxsize=500MB --volume-totalsize=1GB
vmgroup access add succeeded
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
datastore2  False         500.00MB         1.00GB
```

"allow_create" right is given when you run the command with "--allow-create" flag.
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=vmgroup1 --datastore=datastore2  --volume-maxsize=500MB --volume-totalsize=1GB --allow-create
vmgroup access add succeeded
[root@localhost:~]
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
datastore2  True          500.00MB         1.00GB
```
For _VM_DS and _ALL_DS special DS names, --volume-totalzie has to be "Unset".
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=vmgroup1 --datastore=_VM_DS  --volume-maxsize=500MB --volume-totalsize=1GB --allow-create
Canont set volume-totalsize for _VM_DS
[root@localhost:~]
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=vmgroup1 --datastore=_ALL_DS  --volume-maxsize=500MB --volume-totalsize=1GB --allow-create
Canont set volume-totalsize for _VM_DS
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=vmgroup1 --datastore=_ALL_DS  --volume-maxsize=500MB  --allow-create
vmgroup access add succeeded
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
_ALL_DS    True          500.00MB         Unset

```


##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add -h
usage: vmdkops_admin.py vmgroup access add [-h]
                                           [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                           --name NAME
                                           [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                           [--allow-create] --datastore
                                           DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this vmgroup
  --name NAME           The name of the vmgroup
  --volume-maxsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum size of the volume that can be created
  --allow-create        Allow create and delete on datastore if set
  --datastore DATASTORE
                        Datastore which access is controlled


```

#### List
List the current access control granted to a vmgroup.

When displaying the result keep in mind:

- For capacity Unset indicates no limits

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
datastore2  True          500.00MB         1.00GB
```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls -h
usage: vmdkops_admin.py vmgroup access ls [-h] --name NAME

optional arguments:
  -h, --help   show this help message and exit
  --name NAME  The name of the vmgroup

```

#### Remove
Remove access to a datastore for a vmgroup.
Removing of access privilege to "default_datastore" is not suported
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                Default_datastore  VM_list
------------------------------------  --------  -------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup  _VM_DS
2a97fef4-30cd-4a50-bf31-3dbc7d130be2  vmgroup1                             datastore1         photon7

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset
datastore2  True          500.00MB         1.00GB

[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access rm --name=vmgroup1 --datastore=datastore1
Removing of access privilege to "default_datastore" is not supported
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access rm --name=vmgroup1 --datastore=datastore2
vmgroup access rm succeeded
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          Unset            Unset

```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup  access rm -h
usage: vmdkops_admin.py vmgroup access rm [-h] --name NAME --datastore
                                           DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the vmgroup
  --datastore DATASTORE
                        Datstore which access is controlled

```

#### Set
Set command allows to change the existing access control in place for a vmgroup.

Sample:

```shell
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         Unset            Unset
_ALL_DS    True          500.00MB         Unset

[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access set --name=vmgroup1 --datastore=datastore1 --allow-create=True  --volume-maxsize=1000MB --volume-totalsize=2GB
vmgroup access set succeeded
[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          1000.00MB        2.00GB
_ALL_DS    True          500.00MB         Unset
```

"-volume-totalsize" cannot be set to the value other than unlimit when add privilege for special value "_VM_DS" and "_ALL_DS".
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access ls --name=vmgroup1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          1000.00MB        2.00GB
_ALL_DS    True          500.00MB         Unset
_VM_DS     True          Unset            Unset

[root@localhost:~]
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access set --name=vmgroup1 --datastore=_VM_DS --volume-totalsize=1GB
Canont set volume-totalsize for _VM_DS

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access set --name=vmgroup1 --datastore=_ALL_DS --volume-totalsize=1GB
Canont set volume-totalsize for _ALL_DS
```

##### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access set -h
usage: vmdkops_admin.py vmgroup access set [-h]
                                            [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                            --name NAME
                                            [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                            [--allow-create Value{True|False} - e.g. True]
                                            --datastore DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this vmgroup
  --name NAME           The name of the vmgroup
  --volume-maxsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum size of the volume that can be created
  --allow-create Value{True|False} - e.g. True
                        Allow create and delete on datastore if set to True;
                        disallow create and delete on datastore if set to
                        False
  --datastore DATASTORE
                        Datastore name

```

## Volume

#### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls -h
usage: vmdkops_admin.py volume ls [-h] [-c Col1,Col2,...]

optional arguments:
  -h, --help        show this help message and exit
  -c Col1,Col2,...  Display selected columns: Choices = ['volume',
                    'datastore', 'created-by', 'created', 'attached-to',
                    'policy', 'capacity', 'used']
```

#### List All
List all properties for all Docker volumes that exist on datastores accessible to the host.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
Volume  Datastore   VMGroup   Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
------  ----------  ---------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1    datastore1  _DEFAULT   100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 21:36:13 2016
vol12   datastore1  _DEFAULT   100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:29:39 2016
vol1    datastore1  vmgroup1  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:13 2016
vol2    datastore1  vmgroup1  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:23 2016
```

Note that the `Policy` column shows the named VSAN storage policy created with the same tool
(vmdkops_admin.py).  Since these example virtual disks live on a VMFS datastore they do not have a storage
policy and show up as `N/A'.

Note that the `VMGroup` column shows the vmgroup by which the volume was created. If the vmgroup which created the volume has been removed, the `VMGroup` column shows up as 'N/A'. See the following example:

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
Volume  Datastore   VMGroup  Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
------  ----------  --------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1    datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 21:36:13 2016
vol12   datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:29:39 2016
vol1    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:13 2016
vol2    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:23 2016
```

#### List selected columns

Show only the selected columns.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls -c volume,datastore,attached-to
Volume     Datastore   Attached To VM
---------  ----------  --------------
large-vol  datastore1  detached
vol        datastore1  detached
```

Note that the that the choices are given in a comma separated list with no spaces, and are shown in
the help given above with `vmdkops_admin ls -h`.

### Set
Modify attribute settings on a given volume. The volume is identified by its name, vmgroup_name which the volume belongs to and datastore,
for example if the volume name is `container-vol` then the volume is specified as "container-vol@datastore-name".
The attributes to set/modify are specified as a comma separated list as "<attr1>=<value>, <attr2>=<value>....". For example,
a command line would look like this.

```bash
$ vmdkops-admin set --volume=<volume@datastore> --vmgroup=<vmgroup_name> --options="<attr1>=<value>, <attr2>=<value>, ..."
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
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
Volume  Datastore   VMGroup  Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
------  ----------  --------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1    datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 21:36:13 2016
vol12   datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:29:39 2016
vol1    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:13 2016
vol2    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:23 2016

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume set --volume=vol1@datastore1 --vmgroup=_DEFAULT --options="access=read-only"
Successfully updated settings for : vol1@datastore1

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
Volume  Datastore   VMGroup  Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
------  ----------  --------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1    datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-only   independent_persistent  photon-6    Sun Sep 11 21:36:13 2016
vol12   datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:29:39 2016
vol1    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:13 2016
vol2    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:23 2016

```


## Policy

Create, configure and show the VSAN policy names and their corresponding VSAN policy strings. Also show whether or not they are in use.

#### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy -h
usage: vmdkops_admin.py policy [-h] {rm,create,ls,update} ...

positional arguments:
  {rm,create,ls,update}
    rm                  Remove a storage policy
    create              Create a storage policy
    ls                  List storage policies and volumes using those policies
    update              Update the definition of a storage policy and all VSAN
                        objects using that policy

optional arguments:
  -h, --help            show this help message and exit
```

#### Create

Create a VSAN storage policy.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy create --name some-policy --content '(("proportionalCapacity" i0)("hostFailuresToTolerate" i0))'
Successfully created policy: some-policy
```

Note that the VSAN storage policy string given with `--content` is a standard VSAN storage policy
string.  Please refer to the [VSAN documentation](https://pubs.vmware.com/vsphere-60/index.jsp?topic=%2Fcom.vmware.vcli.ref.doc%2Fesxcli_vsan.html)
for storage policy options.

#### List

List all VSAN storage policies.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy ls
Policy Name  Policy Content                                             Active
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
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy update --name some-policy --content '(("proportionalCapacity" i1))'
This operation may take a while. Please be patient.
Successfully updated policy: some-policy
```

#### Remove  (`rm`)

Remove a VSAN storage policy. Note that a storage policy cannot be removed if it is currently in use
by one or more virtual disks.

The ability to list which virtual disks are using a specific storage policy, change storage policies
for a virtual disk, and reset virtual disks to the default storage policy is a necessary
enhancement tracked [here](https://github.com/vmware/docker-volume-vsphere/issues/577).

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy rm --name=some-policy
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
[root@localhost:~] vmdkops_admin config init -h
usage: vmdkops_admin.py config init [-h] [--local] [--force]
                                    [--datastore DATASTORE]

optional arguments:
  -h, --help            show this help message and exit
  --local               Allows local (SingleNode) Init
  --force               Force operation, ignore warnings
  --datastore DATASTORE
                        Config DB will be placed on a shared datastore
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

[root@localhost:~] vmdkops_admin status
Version: 0.12.fea683a-0.0.1
Status: Running
DB_LocalPath: /etc/vmware/vmdkops/auth-db
DB_SharedLocation: /etc/vmware/vmdkops/auth-db
DB_Mode: SingleNode (local DB exists)    <==== LOCAL configuration
Pid: 5979684
Port: 1019
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: DEBUG
```


#### Remove (`config rm`)

Allows to remove local configuration DB. Since this is a destructive operation, admin needs to type both `--local` flag (to confirm it's local only operation and does not impact shared database, if any) , and `--confirm` flag to confirm that she actually wants to delete the local Config DB.

Running `vmdkops_admin config rm` with no flags prints an explanation on how to remove the shared config DB, if any.

```
usage: vmdkops_admin.py config rm [-h] [--local] [--no-backup] [--confirm]

optional arguments:
  -h, --help   show this help message and exit
  --local      Remove only local link or local DB
  --no-backup  Do not create DB backup before removing
  --confirm    Explicitly confirm the operation
```


#### Status

To get config DB  status, use  `/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py status` command.

#### Move (`config mv`)

[Not implemented yet] Allows to relocate config DB between datastores.

## Status

Show config and run-time information about the service.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py status  -h
usage: vmdkops_admin.py status [-h] [--fast]

optional arguments:
  -h, --help  show this help message and exit
  --fast      SKip some of the data collection (port, version)
  ```


```bash
[root@localhost:~] time /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py status
Version: 0.12.0afa0ec-0.0.1
Status: Running
DB_LocalPath: /etc/vmware/vmdkops/auth-db
DB_SharedLocation: N/A
DB_Mode: SingleNode (local DB exists)
Pid: 6298936
Port: 1019
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: DEBUG

real	0m 2.01s
user	0m 0.59s
sys	0m 0.00s
```

 Some of the information retrieval may be slow (e.g. VIB version (`Version` field) # or VMCI port number (`Port` field). `--fast` flag skips slow data collection and prints `?` for fields with no information.

```bash
[root@localhost:~] time /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py status --fast
Version: ?
Status: Running
DB_LocalPath: /etc/vmware/vmdkops/auth-db
DB_SharedLocation: N/A
DB_Mode: SingleNode (local DB exists)
Pid: 6298936
Port: ?
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: DEBUG

real	0m 0.72s
user	0m 0.51s
sys	0m 0.00s
```
