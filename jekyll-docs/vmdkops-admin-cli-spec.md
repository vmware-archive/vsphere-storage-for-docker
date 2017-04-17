[TOC]

In order to manage the inventory of VMDKs created and used by multiple docker engines we must have
an administrative CLI that lives on ESX. This administrative CLI can provide additional status such
as access control, storage policies, and disk usage and capacity.

The admin cli itself is non-interactive. All commands are of the form `vmdkops-admin <Cmd> [Arg1, Arg2,...]`

All output from the admin cli defaults to human readable formats. It will be made easily grepable.

The majority of testing will be automated. We can ensure that parsing calls the right callbacks with
the right information by generating representative input and mocking the callbacks to assert that
the right information is parsed and delivered correctly. Additionally, and specifically for testing
access control, we can create access control definition (vmgroups and privileges) 
and then test that they act as expected by invoking vmdk_ops commmands on behalf of a fake VM.
Unit
tests for stateless logic can be fed mock input representing data from sidecar and the filesystem.
These techniques should be sufficient enough to provide confidence in the implementation.

The rest of this specification will detail the commands to be implemented in the admin cli. Note
that the commands covered in this document only operate on a single ESX host.

# ls
List all volumes by reading the `dockvols` directories and metadata stored in Sidecar files.
Volumes in all datastores will be shown with the volume name in the first column and the datastore
in the second.

`ls` supports the following options:
  * No options - List the names of all volumes
  * `ls -l` - List all volumes and their corresponding metadata. Note that missing metadata will be
    marked `N/A`.
    * Created by (VM Name)
    * Creation Time
    * Last Attached Time
    * Datastore
    * Policy Name
    * Capacity
    * Used Space
    * Attached To (VM Name)
  * `ls -c [Column1, Column2, ...]` - List only the columns specified, implies `-l`
  * `ls -f <ColumnName Pattern Value>` - Filter output by only including volumes whose columns match the
                                         sort pattern (See below). Note that the priority is P2 on
                                         this and it may be developed after the initial release
                                         depending on timing/effort.

Sort patterns are used for filtering. The argument, `X`, itself is of a type specific to the attribute. All dates are given in [ISO 8601 format](https://en.wikipedia.org/wiki/ISO_8601)

 * `> X` The attribute is greater than X
 * `< X` The attribute is equal to X
 * `= X` The attribute matches a Glob X

Examples:
```
vmdkops-admin ls -c CreatedBy,CreationTime,Datastore,PolicyName
vmdkops-admin ls -f 'LastAttachedTime > 2016-03-11'
vmdkops-admin ls -f 'AttachedTo = Test*'
```

# policy
Create, configure and show the VSAN policy names and their corresponding VSAN policies. Also show whether or not they are in use.


Examples:
 * `vmdkops-admin policy create --name=myPolicy --content="string"` - Create a new policy
 * `vmdkops-admin policy rm myPolicy` - Remove the given policy
 * `vmdkops-admin policy ls` - List policies and the Volumes that are using that policy.
 * `vmdkops-admin policy update --name=myPolicy --content="string"` - Update an existing VSAN policy and
   any volumes currently using that policy

Note that on volume creation from docker, a policy name will be passed with a `-o` option.

# vmgroup
Create, delete, configure and show access control settings for a vmgroup.
A vmgroup is defined as a collection of VMs, so access control settings are assigned via a
VM naming convention.
*** The rest of section below needs rework as it represents obsolete "role" design ***
An example will help clarify
this. Let's say that an administrator wants to allow any `Test` VM to create, delete and mount
volumes, and only allow creation of volumes of a maximum size of 2TB. The admin would first create a
`Test` role specifying these permissions, as well as a glob indicating the vm naming convention that
the role should be applied to. In this instance the permissions in the role `Test` will be applied to any VM with a
name ending in `Test`. Since the admin creates the VMs they can control the naming convention and
permissions in a straightforward manner.

`vmdkops-admin role create --name=Test --matches-vm=\*Test --volume-maxsize=2TB --rights=create,delete,mount`

`role create` accepts the following options as shown in the prior example:
 * `--name=<Name>` - The name of the role to be created
 * `--matches-vm=<Glob>` - The glob that matches VM names where the role should be applied. This
                           flag may be applied multiple times, where a VM matching any of the globs would
                           apply.
 * `--rights=<Perm1,Perm2,...>` - The permissions granted to matching VMs. The list of applicable rights is:
   * `create` - Allow volume creation. If the `--volume-maxsize` parameter is given it applies to create, otherwise size is unlimited.
   * `rm` - Allow volume deletion
   * `mount` - Allow volume mounting

Currently the idea is to allow volume deletion and mounting of any volume if the rights are granted.
However it may make sense to limit these to specific volumes, perhaps via a glob match.

Note that the `--volume-maxsize` parameter is human readable and given in the following format: `MB | GB | TB`.

Besides `role create`, there are 4 other commands with regards to roles. Roles can be deleted with
`role rm` and listed with `role ls`. Listing of roles will show all roles and the VMs they apply
to. `role set` takes identical paramters to role create, except it will update an existing role
only. Finally, there needs to be a way to check permissions. This can be performed with the `role
get` command. `role get` takes a VM name and returns both a list of the rights granted to the VM and
the roles that the VM matches on separate lines.

Examples are provided below.
```
 vmdkops-admin role create --name=myrole --matches-vm="glob expression over vm names" --volume-maxsize=2TB --rights=create,delete,mount
 vmdkops-admin role rm myrole
 vmdkops-admin role ls
 vmdkops-admin role set --name=myrole --matches-vm="glob expression over vm names" --volume-maxsize=4TB --rights=create,mount
 vmdkops-admin role get <VmName>
```

Role information is stored in a flat file, for lack of a better solution. The format of the file
is JSON, but the schema is currently undefined. Permission checking will need to be performed at
runtime by the ESX service, using the information provided by `vmdkops-admin role get <VmName>`. It
does seem inefficient to call out to a separate command to perform the check, but this allows the
simplest implementation and isolation of admin information inside a single script.

# Status
Show any interesting information about the service. This includes file paths of config files, version
information, and PID of running service. A simple example is shown here, although it's possible
that the exact format may be somewhat different.

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

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py status
Version: 1.0.0-0.0.1
Status: Running
Pid: 73114
Port: 1019
LogConfigFile: /etc/vmware/vmdkops/log_config.json
LogFile: /var/log/vmware/vmdk_ops.log
LogLevel: INFO
```

# help
Show help as described in this doc.
