[TOC]
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

## Tenant

### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant -h
usage: vmdkops_admin.py tenant [-h] {create,vm,update,access,ls,rm} ...

positional arguments:
  {create,vm,update,access,ls,rm}
    create              Create a new tenant
    vm                  Add, removes and lists VMs in a tenant
    update              Update an existing tenant
    access              Add or remove Datastore access and quotas for a tenant
    ls                  List tenants and the VMs they are applied to
    rm                  Delete a tenant

optional arguments:
  -h, --help            show this help message and exit
```

### Create
A tenant named "_DEFAULT" will be created automatically post install.

Creates a new named tenant and optionally assigns VMs.

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name=tenant1 
tenant create succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list  
------------------------------------  --------  ------------------------  -----------------  -------  
527e94ec-43e9-4d78-81fe-d99ab06a54b3  _DEFAULT  This is a default tenant                              
3277ad45-e916-4e06-8fd5-381e6090d15b  tenant1                                
```

The tenant to VM association can be done at create time.

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name=tenant1 --vm-list=photon4
tenant create succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list  
------------------------------------  --------  ------------------------  -----------------  -------  
527e94ec-43e9-4d78-81fe-d99ab06a54b3  _DEFAULT  This is a default tenant                              
c932f2bf-6554-442a-86f6-ec721dd3dced  tenant1                                                photon4  
```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create -h
usage: vmdkops_admin.py tenant create [-h] --name NAME
                                      [--description DESCRIPTION]
                                      [--vm-list vm1, vm2, ...]

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the tenant
  --description DESCRIPTION
                        The description of the tenant
  --vm-list vm1, vm2, ...
                        A list of VM names to place in this Tenant
```
### List
List existing tenants, the datastores tenants have access to and the VMs assigned.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list  
------------------------------------  --------  ------------------------  -----------------  -------  
527e94ec-43e9-4d78-81fe-d99ab06a54b3  _DEFAULT  This is a default tenant                              
c932f2bf-6554-442a-86f6-ec721dd3dced  tenant1                                                photon4  
```

#### Help
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls -h
usage: vmdkops_admin.py tenant ls [-h]

optional arguments:
  -h, --help  show this help message and exit
```

### Update
Update existing tenant. This command allows to update "Description" and "Default_datastore" fields, or rename an existing tenant.
Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list 
------------------------------------  --------  ------------------------  -----------------  ------- 
a11dda75-0632-4fe0-9caa-d2308bf73df7  _DEFAULT  This is a default tenant                             
946570fe-9842-491f-a172-426284b36eeb  tenant1                             datastore2         photon4 

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant update --name=tenant1 --description="New description of tenant1" --new-name=new-tenant1 --default-datastore=datastore1
tenant modify succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name         Description                 Default_datastore  VM_list 
------------------------------------  -----------  --------------------------  -----------------  ------- 
a11dda75-0632-4fe0-9caa-d2308bf73df7  _DEFAULT     This is a default tenant                               
946570fe-9842-491f-a172-426284b36eeb  new-tenant1  New description of tenant1  datastore1         photon4 
``` 

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant update -h
usage: vmdkops_admin.py tenant update [-h] --name NAME
                                      [--default-datastore DEFAULT_DATASTORE]
                                      [--description DESCRIPTION]
                                      [--new-name NEW_NAME]

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the tenant
  --default-datastore DEFAULT_DATASTORE
                        The name of the datastore to be used by default for
                        volumes placement
  --description DESCRIPTION
                        The new description of the tenant
  --new-name NEW_NAME   The new name of the tenant

```

### Remove
Remove a tenant, optionally all volumes for a tenant can be removed as well.

Sample:
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant rm --name=tenant1 --remove-volumes
All Volumes will be removed
tenant rm succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list  
------------------------------------  --------  ------------------------  -----------------  -------  
527e94ec-43e9-4d78-81fe-d99ab06a54b3  _DEFAULT  This is a default tenant                              
```

#### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant rm -h
usage: vmdkops_admin.py tenant rm [-h] --name NAME [--remove-volumes]

optional arguments:
  -h, --help        show this help message and exit
  --name NAME       The name of the tenant
  --remove-volumes  BE CAREFUL: Removes this tenant volumes when removing a
                    tenant  

```

### Virtual Machine

#### Add
Add a VM to a tenant. A VM can only access the datastores for the tenant it is assigned to.
VMs can be assigned to only one tenant at a time.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm add --name=tenant1 --vm-list=photon5
tenant vm add succeeded
```

#### List
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm ls --name=tenant1
Uuid                                  Name     
------------------------------------  -------  
564df562-3d58-c99a-e76e-e8792b77ca2d  photon4  
564d4728-f1c7-2029-d01e-51f5e6536cd9  photon5  
```

#### Remove
Remove a VM from a tenant's list of VMs. VM will no longer be able to access the volumes created for the tenant.
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm rm --name=tenant1 --vm-list=photon5
tenant vm rm succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm ls --name=tenant1
Uuid                                  Name     
------------------------------------  -------  
564df562-3d58-c99a-e76e-e8792b77ca2d  photon4  
```

#### Help
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm -h
usage: vmdkops_admin.py tenant vm [-h] {rm,add,ls} ...

positional arguments:
  {rm,add,ls}
    rm         Remove VM(s) from a tenant
    add        Add a VM(s) to a tenant
    ls         list VMs in a tenant

optional arguments:
  -h, --help   show this help message and exit
```

### Access
Change the access control for a tenant.
This includes ability to grant privileges & set resource consumption limits for a datastore.

#### Help
```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access -h
usage: vmdkops_admin.py tenant access [-h] {rm,add,set,ls} ...

positional arguments:
  {rm,add,set,ls}
    rm             Remove all access to a datastore for a tenant
    add            Add a datastore access for a tenant
    set            Modify datastore access for a tenant
    ls             List all access info for a tenant

optional arguments:
  -h, --help       show this help message and exit
```

#### Add
Grants datastore access to a tenant. 

The datastore will be automatically set as "default_datastore" for the tenant 
when you grant first datastore access for a tenant.

Sample:

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add --name=tenant1 --datastore=datastore1  --volume-maxsize=500MB --volume-totalsize=1GB
tenant access add succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list  
------------------------------------  --------  ------------------------  -----------------  -------  
527e94ec-43e9-4d78-81fe-d99ab06a54b3  _DEFAULT  This is a default tenant                              
c932f2bf-6554-442a-86f6-ec721dd3dced  tenant1                             datastore1         photon4  
```

The datastore will be set as "default_datastore" for the tenant when you grant datastore access for a tenant with "--default-datastore" flag.

Sample:

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add --name=tenant1 --datastore=datastore2  --allow-create --default-datastore --volume-maxsize=500MB --volume-totalsize=1GB
tenant access add succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name=tenant1
Datastore   Allow_create  Max_volume_size  Total_size  
----------  ------------  ---------------  ----------  
datastore1  True          500.00MB         1.00GB      
datastore2  True          500.00MB         1.00GB      

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list  
------------------------------------  --------  ------------------------  -----------------  -------  
527e94ec-43e9-4d78-81fe-d99ab06a54b3  _DEFAULT  This is a default tenant                              
371a68e3-8c86-467a-a4dc-753f3066ca8a  tenant1                             datastore2         photon4  
```

By default no "allow_create" right is given

```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add --name tenant1 --datastore datastore1
tenant access add succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name=tenant1 
Datastore   Allow_create  Max_volume_size  Total_size  
----------  ------------  ---------------  ----------  
datastore1  False         500.00MB         1.00GB      
```

"allow_create" right is given when you run the command with "--allow-create" flag.
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add --name=tenant1 --datastore=datastore1 --allow-create --volume-maxsize=500MB --volume-totalsize=1GB
tenant access add succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name=tenant1
Datastore   Allow_create  Max_volume_size  Total_size  
----------  ------------  ---------------  ----------  
datastore1  True          500.00MB         1.00GB      
```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add -h
usage: vmdkops_admin.py tenant access add [-h]
                                          [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                          [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                          [--allow-create] --name NAME
                                          [--default-datastore] --datastore
                                          DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this tenant
  --volume-maxsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum size of the volume that can be created
  --allow-create        Allow create and delete on the datastore if set to True
  --name NAME           The name of the tenant
  --default-datastore   Mark datastore as a default datastore for this tenant
  --datastore DATASTORE
                        Datastore which access is controlled

```

#### List
List the current access control granted to a tenant.

When displaying the result keep in mind:

- For capacity Unset indicates no limits

```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name tenant1
Datastore   Allow_create  Max_volume_size  Total_size  
----------  ------------  ---------------  ----------  
datastore1  True          Unset            Unset    
```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls -h
usage: vmdkops_admin.py tenant access ls [-h] --name NAME

optional arguments:
  -h, --help   show this help message and exit
  --name NAME  The name of the tenant
```

#### Remove
Remove access to a datastore for a tenant.
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access rm --name=tenant1 --datastore=datastore1
tenant access rm succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name=tenant1
Datastore  Allow_create  Max_volume_size  Total_size  
---------  ------------  ---------------  ----------  
```

##### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access rm -h
usage: vmdkops_admin.py tenant access rm [-h] --name NAME --datastore
                                         DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the tenant
  --datastore DATASTORE
                        Datstore which access is controlled

```

#### Set
Set command allows to change the existing access control in place for a tenant.

Sample:

```shell
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name=tenant1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  False         500.00MB         1.00GB      

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access set --name=tenant1 --datastore=datastore1 --allow-create --volume-maxsize=1000MB --volume-totalsize=2GB
tenant access set succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name=tenant1
Datastore   Allow_create  Max_volume_size  Total_size
----------  ------------  ---------------  ----------
datastore1  True          1000.00MB        2.00GB      
```

##### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access set -h
usage: vmdkops_admin.py tenant access set [-h]
                                          [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                          --name NAME
                                          [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                          [--allow-create] --datastore
                                          DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this tenant
  --name NAME           Tenant name
  --volume-maxsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum size of the volume that can be created
  --allow-create        Allow create and delete on datastore if set to True
  --datastore DATASTORE
                        Datastore name

```

## Volume

### List

#### Help
```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls -h
usage: vmdkops_admin.py ls [-h] [-c Col1,Col2,...]

optional arguments:
  -h, --help        show this help message and exit
  -c Col1,Col2,...  Display selected columns: Choices = ['volume',
                    'datastore', 'created-by', 'created', 'attached-to',
                    'policy', 'capacity', 'used']
```

#### List All
List all properties for all Docker volumes that exist on datastores accessible to the host.

```bash
Volume          Datastore      Created By VM  Created                   Attached To VM  Policy          Capacity  Used      Filesystem Type  Access      Attach As               
--------------  -------------  -------------  ------------------------  --------------  --------------  --------  --------  ---------------  ----------  ----------------------  
MyVol           vsanDatastore  photon.vsan.1  Mon Jul  4 17:44:28 2016  detached        [VSAN default]  100.00MB  40.00MB   ext4             read-write  independent_persistent  
unt             vsanDatastore  photon.vsan.1  Mon Jul  4 19:55:26 2016  detached        [VSAN default]  100.00MB  40.00MB   ext4             read-write  independent_persistent  
iland           vsanDatastore  photon.vsan.1  Mon Jul  4 20:11:42 2016  detached        [VSAN default]  100.00MB  40.00MB   ext4             read-write  independent_persistent  
foo             vsanDatastore  photon.vsan.1  Mon Jul 11 03:17:09 2016  detached        [VSAN default]  20.00GB   228.00MB  ext4             read-write  independent_persistent  
MyVolume        vsanDatastore  photon.vsan.1  Mon Jul 11 10:25:45 2016  detached        [VSAN default]  10.00GB   192.00MB  ext4             read-write  independent_persistent  
storage-meetup  vsanDatastore  photon.vsan.1  Mon Jul 11 10:50:20 2016  detached        [VSAN default]  10.00GB   348.00MB  ext4             read-write  independent_persistent  
storage         vsanDatastore  photon.vsan.1  Tue Jul 12 12:53:07 2016  detached        [VSAN default]  10.00GB   192.00MB  ext4             read-write  independent_persistent  
storage2        vsanDatastore  photon.vsan.1  Tue Jul 12 13:00:11 2016  detached        [VSAN default]  10.00GB   192.00MB  ext4             read-write  independent_persistent  
```

Note that the `Policy` column shows the named VSAN storage policy created with the same tool
(vmdkops_admin.py).  Since these example virtual disks live on a VMFS datastore they do not have a storage
policy and show up as `N/A'.

#### List selected columns

Show only the selected columns.

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls -c volume,datastore,attached-to
Volume     Datastore   Attached To VM
---------  ----------  --------------
large-vol  datastore1  detached
vol        datastore1  detached
```

Note that the that the choices are given in a comma separated list with no spaces, and are shown in
the help given above with `vmdkops_admin ls -h`.

### Set
Modify attribute settings on a given volume. The volume is identified by its name and datastore,
for example if the volume name is `container-vol` then the volume is specified as "container-vol@datastore-name".
The attributes to set/modify are specified as a comma separated list as "<attr1>=<value>, <attr2>=<value>....". For example,
a command line would look like this.

```bash
$ vmdkops-admin set --volume=<volume@datastore> --options="<attr1>=<value>, <attr2>=<value>, ..."
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
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy create --name some-policy --content '(("proportionalCapacity" i0)("hostFailuresToTolerate" i0)'
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
-----------  ---------------------------------------------------------  ------
some-policy  (("proportionalCapacity" i0)("hostFailuresToTolerate" i0)  Unused
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
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy update --name some-policy --content '(("proportionalCapacity" i1)'
This operation may take a while. Please be patient.
Successfully updated policy: some-policy
```

#### Remove

Remove a VSAN storage policy. Note that a storage policy cannot be removed if it is currently in use
by one or more virtual disks.

The ability to list which virtual disks are using a specific storage policy, change storage policies
for a virtual disk, and reset virtual disks to the default storage policy is a necessary
enhancement tracked [here](https://github.com/vmware/docker-volume-vsphere/issues/577).

```bash
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy rm some-policy
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
