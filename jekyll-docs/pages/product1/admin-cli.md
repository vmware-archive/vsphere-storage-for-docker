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

The admin cli also enables ESX admins to implement tenancy.

The remainder of this document will describe each admin CLI command and provide examples
of their usage.

## Vm-group

### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group -h
usage: vmdkops_admin.py vm-group [-h] {create,vm,update,access,ls,rm} ...

positional arguments:
  {create,vm,update,access,ls,rm}
    create              Create a new vm-group
    vm                  Add, removes and lists VMs in a vm-group
    update              Update an existing vm-group
    access              Add or remove Datastore access and quotas for a vm-
                        group
    ls                  List vm-groups and the VMs they are applied to
    rm                  Delete a vm-group

optional arguments:
  -h, --help            show this help message and exit
```

### Create
A vm-group named "_DEFAULT" will be created automatically post install.

Creates a new named vm-group and optionally assigns VMs. Valid vm-group name is only allowed to be "[a-zA-Z0-9_][a-zA-Z0-9_.-]*"

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group create --name=vm-group1
vm-group create succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vm-group
1ddb5b46-6a9f-4649-8e48-c47039905752  vm-group1
```

The vm-group to VM association can be done at create time.

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group create --name=vm-group1 --vm-list=photon6
vm-group create succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  --------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vm-group
035ddfb7-349b-4ba1-8abf-e77a430d5098  vm-group1                                                 photon6


```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group create -h
usage: vmdkops_admin.py vm-group create [-h] --name NAME
                                        [--description DESCRIPTION]
                                        [--vm-list vm1, vm2, ...]

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the vm-group
  --description DESCRIPTION
                        The description of the vm-group
  --vm-list vm1, vm2, ...
                        A list of VM names to place in this vm-group

```
### List
List existing vm-groups, the datastores vm-groups have access to and the VMs assigned.
```
[root@localhost:~] usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  --------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vm-group
035ddfb7-349b-4ba1-8abf-e77a430d5098  vm-group1                                                 photon6

```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls -h
usage: vmdkops_admin.py vm-group ls [-h]

optional arguments:
  -h, --help  show this help message and exit
```

### Update
Update existing vm-group. This command allows to update "Description" and "Default_datastore" fields, or rename an existing vm-group.
Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  --------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vm-group
035ddfb7-349b-4ba1-8abf-e77a430d5098  vm-group1                                                 photon6

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group update --name=vm-group1 --description="New description of vm-group1" --new-name=new-vm-group1 --default-datastore=datastore1
vm-group modify succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name           Description                   Default_datastore  VM_list
------------------------------------  -------------  ----------------------------  -----------------  --------
11111111-1111-1111-1111-111111111111  _DEFAULT       This is a default vm-group
035ddfb7-349b-4ba1-8abf-e77a430d5098  new-vm-group1  New description of vm-group1  datastore1         photon6

```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group  update -h
usage: vmdkops_admin.py vm-group update [-h] --name NAME
                                        [--default-datastore DEFAULT_DATASTORE]
                                        [--description DESCRIPTION]
                                        [--new-name NEW_NAME]

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the vm-group
  --default-datastore DEFAULT_DATASTORE
                        The name of the datastore to be used by default for
                        volumes placement
  --description DESCRIPTION
                        The new description of the vm-group
  --new-name NEW_NAME   The new name of the vm-group

```

### Remove
Remove a vm-group, optionally all volumes for a vm-group can be removed as well.

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group rm --name=vm-group1 --remove-volumes
All Volumes will be removed
vm-group rm succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name      Description                 Default_datastore  VM_list
------------------------------------  --------  --------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vm-group
```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group rm -h
usage: vmdkops_admin.py vm-group rm [-h] --name NAME [--remove-volumes]

optional arguments:
  -h, --help        show this help message and exit
  --name NAME       The name of the vm-group
  --remove-volumes  BE CAREFUL: Removes this vm-group volumes when removing a
                    vm-group

```

### Virtual Machine

#### Add
Add a VM to a vm-group. A VM can only access the datastores for the vm-group it is assigned to.
VMs can be assigned to only one vm-group at a time.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  --------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vm-group
6d810c66-ffc7-47c8-8870-72114f86c2cf  vm-group1                                                 photon6

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm add --name=vm-group1 --vm-list=photon7
vm-group vm add succeeded

```

#### List
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm ls --name=vm-group1
Uuid                                  Name
------------------------------------  --------
564d5849-b135-1259-cc73-d2d3aa1d9b8c  photon6
564d99a2-4097-9966-579f-3dc4082b10c9  photon7
```

#### Remove
Remove a VM from a vm-group's list of VMs. VM will no longer be able to access the volumes created for the vm-group.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm rm --name=vm-group1 --vm-list=photon7
vm-group vm rm succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm ls --name=vm-group1
Uuid                                  Name
------------------------------------  --------
564d5849-b135-1259-cc73-d2d3aa1d9b8c  photon6

```

### Replace
Replace VMs from a vm-group's list of VMs. VMs which are replaced will no longer be able to access the volumes created for the vm-group.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm ls --name=vm-group1
Uuid                                  Name
------------------------------------  --------
564d5849-b135-1259-cc73-d2d3aa1d9b8c  photon6

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm replace --name=vm-group1 --vm-list=photon7
vm-group vm replace succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm ls --name=vm-group1
Uuid                                  Name
------------------------------------  --------
564d99a2-4097-9966-579f-3dc4082b10c9  photon7
```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group vm -h
usage: vmdkops_admin.py vm-group vm [-h] {rm,add,ls,replace} ...

positional arguments:
  {rm,add,ls,replace}
    rm                 Remove VM(s) from a vm-group
    add                Add a VM(s) to a vm-group
    ls                 list VMs in a vm-group
    replace            Replace VM(s) for a vm-group

optional arguments:
  -h, --help           show this help message and exit

```

### Access
Change the access control for a vm-group.
This includes ability to grant privileges & set resource consumption limits for a datastore.

#### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access -h
usage: vmdkops_admin.py vm-group access [-h] {rm,add,set,ls} ...

positional arguments:
  {rm,add,set,ls}
    rm             Remove all access to a datastore for a vm-group
    add            Add a datastore access for a vm-group
    set            Modify datastore access for a vm-group
    ls             List all access info for a vm-group

optional arguments:
  -h, --help       show this help message and exit

```

#### Add
Grants datastore access to a vm-group.

The datastore will be automatically set as "default_datastore" for the vm-group
when you grant first datastore access for a vm-group.

Sample:

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access add --name=vm-group1 --datastore=datastore1  --volume-maxsize=500MB --volume-totalsize=1GB
vm-group access add succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vm-group
6d810c66-ffc7-47c8-8870-72114f86c2cf  vm-group1                              datastore1         photon7
```

The datastore will be set as "default_datastore" for the vm-group when you grant datastore access for a vm-group with "--default-datastore" flag.

Sample:

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access add --name=vm-group1 --datastore=datastore2  --allow-create --default-datastore --volume-maxsize=500MB --volume-totalsize=1GB
vm-group access add succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         500.00MB         1.00GB
datastore2  True          500.00MB         1.00GB

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vm-group
6d810c66-ffc7-47c8-8870-72114f86c2cf  vm-group1                              datastore2         photon7

```

By default no "allow_create" right is given

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access add --name=vm-group1 --datastore=datastore1  --volume-maxsize=500MB --volume-totalsize=1GB
vm-group access add succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         500.00MB         1.00GB
```

"allow_create" right is given when you run the command with "--allow-create" flag.
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access add --name=vm-group1 --datastore=datastore2  --allow-create --default-datastore --volume-maxsize=500MB --volume-totalsize=1GB
vm-group access add succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         500.00MB         1.00GB
datastore2  True          500.00MB         1.00GB
```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access add -h
usage: vmdkops_admin.py vm-group access add [-h]
                                            [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                            [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                            [--allow-create] --name NAME
                                            [--default-datastore] --datastore
                                            DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this vm-group
  --volume-maxsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum size of the volume that can be created
  --allow-create        Allow create and delete on datastore if set
  --name NAME           The name of the vm-group
  --default-datastore   Mark datastore as a default datastore for this vm-
                        group
  --datastore DATASTORE
                        Datastore which access is controlled


```

#### List
List the current access control granted to a vm-group.

When displaying the result keep in mind:

- For capacity Unset indicates no limits

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         500.00MB         1.00GB
datastore2  True          500.00MB         1.00GB
```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls -h
usage: vmdkops_admin.py vm-group access ls [-h] --name NAME

optional arguments:
  -h, --help   show this help message and exit
  --name NAME  The name of the vm-group

```

#### Remove
Remove access to a datastore for a vm-group.
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         500.00MB         1.00GB
datastore2  True          500.00MB         1.00GB

[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group  access rm --name=vm-group1 --datastore=datastore1
vm-group access rm succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore2  True          500.00MB         1.00GB
```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group  access rm -h
usage: vmdkops_admin.py vm-group access rm [-h] --name NAME --datastore
                                           DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the vm-group
  --datastore DATASTORE
                        Datstore which access is controlled

```

#### Set
Set command allows to change the existing access control in place for a vm-group.

Sample:

```shell
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         500.00MB         1.00GB

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access set --name=vm-group1 --datastore=datastore1 --allow-create=True  --volume-maxsize=1000MB --volume-totalsize=2GB
vm-group access set succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls
usage: vmdkops_admin.py vm-group access ls [-h] --name NAME
vmdkops_admin.py vm-group access ls: error: argument --name is required
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access ls --name=vm-group1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          1000.00MB        2.00GB



```

##### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vm-group access set -h
usage: vmdkops_admin.py vm-group access set [-h]
                                            [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                            --name NAME
                                            [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                            [--allow-create Value{True|False} - e.g. True]
                                            --datastore DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this vm-group
  --name NAME           The name of the vm-group
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
Volume  Datastore   VM-Group   Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
------  ----------  ---------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1    datastore1  _DEFAULT   100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 21:36:13 2016
vol12   datastore1  _DEFAULT   100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:29:39 2016
vol1    datastore1  vm-group1  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:13 2016
vol2    datastore1  vm-group1  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:23 2016
```

Note that the `Policy` column shows the named vSAN storage policy created with the same tool
(vmdkops_admin.py).  Since these example virtual disks live on a VMFS datastore they do not have a storage
policy and show up as `N/A'.

Note that the `VM-Group` column shows the vm-group by which the volume was created. If the vm-group which created the volume has been removed, the `VM-Group` column shows up as 'N/A'. See the following example:

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
Volume  Datastore   VM-Group  Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
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
Modify attribute settings on a given volume. The volume is identified by its name, vm-group_name which the volume belongs to and datastore,
for example if the volume name is `container-vol` then the volume is specified as "container-vol@datastore-name".
The attributes to set/modify are specified as a comma separated list as "<attr1>=<value>, <attr2>=<value>....". For example,
a command line would look like this.

```bash
$ vmdkops-admin set --volume=<volume@datastore> --vm-group=<vm-group_name> --options="<attr1>=<value>, <attr2>=<value>, ..."
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
Volume  Datastore   VM-Group  Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
------  ----------  --------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1    datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 21:36:13 2016
vol12   datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:29:39 2016
vol1    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:13 2016
vol2    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:23 2016

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume set --volume=vol1@datastore1 --vm-group=_DEFAULT --options="access=read-only"
Successfully updated settings for : vol1@datastore1

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
Volume  Datastore   VM-Group  Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
------  ----------  --------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1    datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-only   independent_persistent  photon-6    Sun Sep 11 21:36:13 2016
vol12   datastore1  _DEFAULT  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:29:39 2016
vol1    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:13 2016
vol2    datastore1  N/A       100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Sun Sep 11 22:48:23 2016

```


## Policy

Create, configure and show the vSAN policy names and their corresponding vSAN policy strings. Also show whether or not they are in use.

#### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy -h
usage: vmdkops_admin.py policy [-h] {rm,create,ls,update} ...

positional arguments:
  {rm,create,ls,update}
    rm                  Remove a storage policy
    create              Create a storage policy
    ls                  List storage policies and volumes using those policies
    update              Update the definition of a storage policy and all vSAN
                        objects using that policy

optional arguments:
  -h, --help            show this help message and exit
```

#### Create

Create a vSAN storage policy.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy create --name some-policy --content '(("proportionalCapacity" i0)("hostFailuresToTolerate" i0)'
Successfully created policy: some-policy
```

Note that the vSAN storage policy string given with `--content` is a standard vSAN storage policy
string.  Please refer to the [vSAN documentation](https://pubs.vmware.com/vsphere-60/index.jsp?topic=%2Fcom.vmware.vcli.ref.doc%2Fesxcli_vsan.html)
for storage policy options.

#### List

List all vSAN storage policies.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy ls
Policy Name  Policy Content                                             Active
-----------  ---------------------------------------------------------  ------
some-policy  (("proportionalCapacity" i0)("hostFailuresToTolerate" i0)  Unused
```

When creating a virtual disk using `docker volume create`, the policy name should be given with the `-o`
option such as `docker volume create --driver=vsphere --name=some-vol -o vsan-policy-name=some-policy`.
The number of virtual disks using the policy will then show up in the `Active` column.

#### Update

Update a vSAN storage policy.

This command will update a vSAN storage policy for all virtual disks currently using this policy. If
the command fails, the number of virtual disks that were successfully updated and the number that
failed to update will be shown. The names of the virtual disks that failed to update will be logged
so that manual action can be taken.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy update --name some-policy --content '(("proportionalCapacity" i1)'
This operation may take a while. Please be patient.
Successfully updated policy: some-policy
```

#### Remove

Remove a vSAN storage policy. Note that a storage policy cannot be removed if it is currently in use
by one or more virtual disks.

The ability to list which virtual disks are using a specific storage policy, change storage policies
for a virtual disk, and reset virtual disks to the default storage policy is a necessary
enhancement tracked [here](https://github.com/vmware/docker-volume-vsphere/issues/577).

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy rm --name=some-policy
Successfully removed policy: some-policy
```
## Status

Show config and run-time information about the service.

```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py status
 Version: 1.0.0-0.0.1
 Status: Running
 Pid: 161104
 Port: 1019
 LogConfigFile: /etc/vmware/vmdkops/log_config.json
 LogFile: /var/log/vmware/vmdk_ops.log
 LogLevel: INFO
```