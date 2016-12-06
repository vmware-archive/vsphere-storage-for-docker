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
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant -h
usage: vmdkops_admin.py tenant [-h] {access,rm,create,ls,vm} ...

positional arguments:
  {access,rm,create,ls,vm}
    access              Add or remove Datastore access and quotas for a tenant
    rm                  Delete a tenant
    create              Create a new tenant
    ls                  List tenants and the VMs they are applied to
    vm                  Add, removes and lists VMs in a tenant

optional arguments:
  -h, --help            show this help message and exit
```

### Create
Creates a new named tenant and optionally assigns VMs.

Sample:
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name ProjectX
tenant create succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls 
Uuid                                  Name      Description  Default_datastore  VM_list                      
------------------------------------  --------  -----------  -----------------  ---------------------------  
4acd19ee-1ed6-4716-bcda-c8637c9658f2  tenant1                default_ds         photon.vsan.1,photon.vsan.2  
2918b105-0837-4955-a0dc-5319ce456e28  ProjectX               default_ds        
```

The tenant to VM association can be done at create time.

Sample:
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name ProjectX --vm-list ubuntu
tenant create succeeded
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name     Description  Default_datastore  VM_list
------------------------------------  -------  -----------  -----------------  ---------------------------
dc17b3b1-1f4d-4f8b-a6f4-3dba064ba88f  tenant1               default_ds         photon.vsan.1,photon.vsan.2
5afa4767-ecd8-489e-9d97-5c46ae50d3fc  ProjectX              default_ds         ubuntu
```

#### Help
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create -h
usage: vmdkops_admin.py tenant create [-h] --name NAME
                                      [--vm-list vm1, vm2, ...]

optional arguments:
  -h, --help            show this help message and exit
  --name NAME           The name of the tenant
  --vm-list vm1, vm2, ...
                        A list of VM names to place in this Tenant
```
### List
List existing tenants, the datastores tenants have access to and the VMs assigned.
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name     Description  Default_datastore  VM_list
------------------------------------  -------  -----------  -----------------  ---------------------------
dc17b3b1-1f4d-4f8b-a6f4-3dba064ba88f  tenant1               default_ds         photon.vsan.1,photon.vsan.2
5afa4767-ecd8-489e-9d97-5c46ae50d3fc  PojectX               default_ds         ubuntu
```

#### Help
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls -h
usage: vmdkops_admin.py tenant ls [-h]

optional arguments:
  -h, --help  show this help message and exit
```


### Virtual Machine

#### Add
Add a VM to a tenant. A VM can only access the datastores for the tenant it is assigned to.
VMs can be assigned to only one tenant at a time.
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm add --name tenant1 --vm-list photon.vsan.2
tenant vm add succeeded
```

#### List
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm ls --name tenant1
Uuid                                  Name
------------------------------------  -------------
564d672f-0576-8906-7ba0-a42317328499  photon.vsan.1
564d6f23-eabb-1b63-c79d-2086cca704d5  photon.vsan.2
```

#### Remove
Remove a VM from a tenant's list of VMs. VM will no longer be able to access the volumes created for the tenant.
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm rm --name tenant1 --vm-list photon.vsan.1
tenant vm rm succeeded
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant vm ls --name tenant1
Uuid                                  Name
------------------------------------  -------------
564d6f23-eabb-1b63-c79d-2086cca704d5  photon.vsan.2
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

Sample:

```bash
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add --name tenant1 --datastore datastore1 --rights create,delete,mount
tenant access add succeeded
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name tenant1
Datastore   Create_volume  Delete_volume  Mount_volume  Max_volume_size  Total_size
----------  -------------  -------------  ------------  ---------------  ----------
datastore1  1              1              1             0B               0B
```

By default no rights are given

```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add --name tenant1 --datastore datastore1
tenant access add succeeded
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name tenant1
Datastore   Create_volume  Delete_volume  Mount_volume  Max_volume_size  Total_size
----------  -------------  -------------  ------------  ---------------  ----------
datastore1  0              0              0             0B               0B
```

- For capability 0 indicates false
- For capacity 0 indicates no limits

##### Help
```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add -h
usage: vmdkops_admin.py tenant access add [-h]
                                          [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                          --name NAME
                                          [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                          --datastore DATASTORE
                                          [--rights create,delete,mount]

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this tenant
  --name NAME           The name of the tenant
  --volume-maxsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum size of the volume that can be created
  --datastore DATASTORE
                        Datastore which access is controlled
  --rights create,delete,mount
                        Datastore access Permissions granted: Choices =
                        ['create', 'delete', 'mount', 'all']
```

#### List
List the current access control granted to a tenant.

When displaying the result keep in mind:

- For capability 0 indicates false
- For capacity 0 indicates no limits

```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name tenant1
Datastore   Create_volume  Delete_volume  Mount_volume  Max_volume_size  Total_size
----------  -------------  -------------  ------------  ---------------  ----------
datastore1  1              1              1             0B               0B
```

##### Help
```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls -h
usage: vmdkops_admin.py tenant access ls [-h] --name NAME

optional arguments:
  -h, --help   show this help message and exit
  --name NAME  The name of the tenant
```

#### Remove
Remove access to a datastore for a tenant.
```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access rm --name tenant1 --datastore datastore1
tenant access rm succeeded
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name tenant1
Datastore  Create_volume  Delete_volume  Mount_volume  Max_volume_size  Total_size
---------  -------------  -------------  ------------  ---------------  ----------
```

##### Help
```bash
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access rm -h
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
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name tenant1
Datastore   Create_volume  Delete_volume  Mount_volume  Max_volume_size  Total_size
----------  -------------  -------------  ------------  ---------------  ----------
datastore1  0              0              0             0B               0B
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access set --volume-totalsize 2TB --volume-maxsize 500GB --volume-maxcount 100 --add-rights all --name tenant1 --datastore datastore1
tenant access set succeeded
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access ls --name tenant1
Datastore   Create_volume  Delete_volume  Mount_volume  Max_volume_size  Total_size
----------  -------------  -------------  ------------  ---------------  ----------
datastore1  1              1              1             500.00GB         2.00TB
```

##### Help
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access set -h
usage: vmdkops_admin.py tenant access set [-h]
                                          [--volume-totalsize Num{MB,GB,TB} - e.g. 2TB]
                                          [--volume-maxsize Num{MB,GB,TB} - e.g. 2TB]
                                          [--volume-maxcount VOLUME_MAXCOUNT]
                                          [--rm-rights create,delete,mount,all]
                                          [--add-rights create,delete,mount,all]
                                          --name NAME --datastore DATASTORE

optional arguments:
  -h, --help            show this help message and exit
  --volume-totalsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum total size of all volume that can be created
                        on the datastore for this tenant
  --volume-maxsize Num{MB,GB,TB} - e.g. 2TB
                        Maximum size of the volume that can be created
  --volume-maxcount VOLUME_MAXCOUNT
                        Maximum number of volumes to create on the datastore
                        for this tenant
  --rm-rights create,delete,mount,all
                        Datastore access Permissions removed: Choices =
                        ['create', 'delete', 'mount', 'all']
  --add-rights create,delete,mount,all
                        Datastore access Permissions granted: Choices =
                        ['create', 'delete', 'mount', 'all']
  --name NAME           Tenant name
  --datastore DATASTORE
                        Datastore name
```

### Remove
Remove a tenant, optionally all volumes for a tenant can be removed as well.

Sample:
```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant rm --name=tenant1 --remove-volumes
```

#### Help
```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant rm -h
usage: vmdkops_admin tenant rm [-h] --name NAME [--remove-volumes]

optional arguments:
  -h, --help        show this help message and exit
  --name NAME       The name of the tenant
  --remove-volumes  BE CAREFUL: Removes this tenant volumes when removing a
                    tenant
                        
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
option such as `docker volume create --driver=vmdk --name=some-vol -o vsan-policy-name=some-policy`.
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
