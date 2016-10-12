[TOC]

# Introduction
In the context of the Docker volume plugin for vSphere, each ESXi host manages multiple VMs, with
each of them acting as a Docker host. The Docker engine on these hosts communicates with the Docker
volume plugin in order to create and delete virtual disks (VMDKs), as well as mount them as Docker
volumes. These virtual disks may live on any datastore accessible to the ESXi host and are managed
by the Docker user via the Docker CLI. However, the Docker CLI is limited in what visibility it can
provide to the user. Furthermore, it is desirable that an administrator be able to get a global view
of all virtual disks created and in use on the host.  For these reasons, an admin CLI has been
created that runs on the ESXi host and that provides access to information not visible from the
Docker CLI. The remainder of this document will describe each admin CLI command and provide examples
of their usage.


# ls

### Help
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls -h
usage: vmdkops_admin.py ls [-h] [-c Col1,Col2,...]

optional arguments:
  -h, --help        show this help message and exit
  -c Col1,Col2,...  Display selected columns: Choices = ['volume',
                    'datastore', 'created-by', 'created', 'attached-to',
                    'policy', 'capacity', 'used']
```

### List All

List all properties for all Docker volumes that exist on datastores accessible to the host.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls
Volume     Datastore   Created By VM  Created                   Attached To VM  Policy  Capacity  Used
---------  ----------  -------------  ------------------------  --------------  ------  --------  -------
large-vol  datastore1  Ubuntu_15.10   Sat Apr 16 13:34:12 2016  detached        N/A     1.00GB    25.00MB
vol        datastore1  Ubuntu_15.10   Sat Apr 16 20:13:18 2016  detached        N/A     100.00MB  14.00MB
```

Note that the `Policy` column shows the named VSAN storage policy created with the same tool
(vmdkops_admin.py).  Since these example virtual disks live on a VMFS datastore they do not have a storage
policy and show up as `N/A'.

### List selected columns

Show only the selected columns.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls -c volume,datastore,attached-to
Volume     Datastore   Attached To VM
---------  ----------  --------------
large-vol  datastore1  detached
vol        datastore1  detached
```

Note that the that the choices are given in a comma separated list with no spaces, and are shown in
the help given above with `vmdkops_admin ls -h`.

# policy

Create, configure and show the VSAN policy names and their corresponding VSAN policy strings. Also show whether or not they are in use.

### Help
```
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

### Create

Create a VSAN storage policy.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy create --name some-policy --content '(("proportionalCapacity" i0)("hostFailuresToTolerate" i0)'
Successfully created policy: some-policy
```

Note that the VSAN storage policy string given with `--content` is a standard VSAN storage policy
string.  Please refer to the [VSAN documentation](https://pubs.vmware.com/vsphere-60/index.jsp?topic=%2Fcom.vmware.vcli.ref.doc%2Fesxcli_vsan.html)
for storage policy options.

### List

List all VSAN storage policies.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy ls
Policy Name  Policy Content                                             Active
-----------  ---------------------------------------------------------  ------
some-policy  (("proportionalCapacity" i0)("hostFailuresToTolerate" i0)  Unused
```

When creating a virtual disk using `docker volume create`, the policy name should be given with the `-o`
option such as `docker volume create --driver=vmdk --name=some-vol -o vsan-policy-name=some-policy`.
The number of virtual disks using the policy will then show up in the `Active` column.

### Update

Update a VSAN storage policy.

This command will update a VSAN storage policy for all virtual disks currently using this policy. If
the command fails, the number of virtual disks that were successfully updated and the number that
failed to update will be shown. The names of the virtual disks that failed to update will be logged
so that manual action can be taken.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy update --name some-policy --content '(("proportionalCapacity" i1)'
This operation may take a while. Please be patient.
Successfully updated policy: some-policy
```

### Remove

Remove a VSAN storage policy. Note that a storage policy cannot be removed if it is currently in use
by one or more virtual disks.

The ability to list which virtual disks are using a specific storage policy, change storage policies
for a virtual disk, and reset virtual disks to the default storage policy is a necessary
enhancement tracked [here](https://github.com/vmware/docker-volume-vsphere/issues/577).

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy rm some-policy
Successfully removed policy: some-policy
```
# Set
Modify attribute settings on a given volume. The volume is identified by its name and datastore, 
for example if the volume name is `container-vol` then the volume is specified as "container-vol@datastore-name".
The attributes to set/modify are specified as a comma separated list as "<attr1>=<value>, <attr2>=<value>....". For example,
a command line would look like this.

```
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

# Status

Show config and run-time information about the service.

```
[root@localhost:~]  /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py status
 Version: 1.0.0-0.0.1
 Status: Running
 Pid: 161104
 Port: 1019
 LogConfigFile: /etc/vmware/vmdkops/log_config.json
 LogFile: /var/log/vmware/vmdk_ops.log
 LogLevel: INFO
```
