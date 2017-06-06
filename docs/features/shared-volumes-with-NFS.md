# Supporting NFS shared volumes with vDVS

## Requirements
* Developer should be able to use externally managed NFS share as “shared docker volume” using “vsphere” volume driver.
* This allows them to use NFS directories as docker volumes using Docker APIs, Compose etc 
* Extend the vSphere Docker Volume plugin to support NFS shared volumes and multiple readers/writers access
* Volumes are accessible on multiple docker hosts running on  multiple ESX hosts in a Docker swarm
* Allow admin to specify what remote dirs are managed by the plugin
* Support all current docker volume plugin API for shared volumes

## User experience
### Set up remote directories for docker volumes
The design here assumes that the admin is responsible to identify remote directories to export and has configured remote servers to export those. Specifically, the admin needs to have,
1. Exported remote directories to be used to host docker volumes.
1. Configured sufficient permissions for the admins on the Docker host to be able to mount and create folders and files on those directories.
1. The docker host has the necessary packages of the desired version of NFS client installed.

The admin can use the same remote directory configuration on multiple docker hosts on separate physical hosts and be able to view and use the same volumes on their containers (the applications must be able to synchronize access to shared volumes).

### Update plugin configuration
The plugin needs to be provided the details of the exported diretories and the servers from which to mount them. The details are entered into the plugin configuration file at /etc/docker-volume-vsphere.conf in the format as detailed later.

### Restart plugin
The plugin needs to be restarted to load the updated configuration.

### Using shared volumes
Docker volume commands must be used as shown below to use volumes on remote directories. Similar to how local VMDK based volumes are qualified with a datastore name, shared volumes must also be qualified with the label (see remote directory configuration below) that identifies the remote dir. to use to locate and refer a volume.
1. docker volume create - 
   1. _docker volume create --driver=vsphere --name=remote-vol -o type=<fstype - nfs/nfs4> - uses the default entry in the list of remote dirs to locate the volume
   1. _docker volume create --driver=vsphere --name=remote-vol@remote-dir-label_ - creates the volume in the location identified by the label in the remote dirs list
1. docker volume list - No change
1. docker volume inspect - the volume must be qualified by the remote dir label as below,
   1. _docker volume inspect vol-name@remote-dir-label_
1. docker run - the volume must be qualified by the remote dir label as below,
   1. _docker run -it --volume-driver=vsphere -v remote-vol@remote-dir-label:/vol1 --name ub ubuntu_
1. docker volume rm - the volume must be qualified by the remote dir label as below,
   1. _docker volume rm vol-name@remote-dir-label_


# Extending the vSphere driver to support shared volumes

The vSphere plugin driver supports an extendable framework that allows adding support for volumes backed by specific storage types. Presently the driver supports VMDK backed volumes on storage available to the local ESX host. A new module to support NFS backed volumes (on local or NFS mounted directories) is added to the existing framework that supports all of the docker volume functions as listed above.

The new NFS backed volumes will support NFS v3/v4.0 protocols to mount and use remote directories to host docker volumes. Docker volumes will be supported as folders on the mounted directories and exported to containers in the same way that VMDK volumes are made available to containers.

# Plugin configuration for remote directories

The plugin configuration defined in /etc/docker-volume-vsphere.conf must be modified to include the details of the remote directories to use to locate volumes. The format describing a remote directory is as shown below,

## /etc/docker-volume-vsphere.conf
{
"MaxLogAgeDays": 28,
"MaxLogSizeMb": 100,
"LogPath": "/var/log/docker-volume-vsphere.log",
"LogLevel": "debug",
_“RemoteDirs”: {_
     _"DefaultLabel: remote-dir-label"_
     _"RemoteDirTbl: {"_
        _“<dir-A-label>” :   {_
             _“Path”: <remote path to mount>,_
             _“Args: <mount args>,_
             _“Type”: <FS type – nfs, nfs4>,_
             _“VolPath”: <optional path name under path above to locate volumes>,_
	     _"Src": <label of another entry in this list to use to locate volumes>_
        _},_
        _“<dir-B-label>” :   {_
             _“Path”: <remote path to mount>,_
             _“Args: <mount args>,_
             _“Type”: <FS type – nfs, nfs4>,_
             _“VolPath”: <optional path name under path above to locate volumes>,_
	     _"Src": <label of another entry in this list to use to locate volumes>_
        _},_
        _“<dir-C-label>” :   {_
             _“VolPath”: <optional path name under path above to locate volumes>_
	     _"Src": "dir-A-label"_
        _},_
     _},_
  _}_

The configuration items are described below:

1. “RemoteDirs”:  Map of remote dirs.
1. “<dir-A-label>” – must be unique and admin defined and identifies a specific remote dir or another entry within the list
1. “args – args to mount remote dir.
1. “path” – path on server to mount
1. “type” – FS type of remote dir.
1. “vol-path” – a path name that is relative to the mounted path abnove, volumes will be create relative to this path if non-null, if not specified then volumes are located under path above. The _vol-path_ entry allows the admin  admin to separate volumes for different docker service stacks within a single remote dir.
1. "src" - indicates that the entry in the remote directory table refers to a parent entry in the same table. Combined with a _VolPath_ this allows the admin to manage placement of volumes on separate folders within a single remote dir. Especially useful when the admin wants to separate volumes for different service stacks into separate folders. It also allows the admin from having to list many duplicate entries, with the same remote directory details only to differ in the _VolPath_. For example consider two services _Svc_A_ and _Svc_B_ that are using volumes on a remote dir _NetShare_. To locate volumes for the two services in two separate directories under _NetShare_ the admin needs to create two duplicate entries with the same details with different _VolPath_ values. A change in one will need to be duplicated in the other as well. Instead the admin creates a single entry for _NetShare_ and two entries for _Svc_A_ and _Svc_B_ respectively that each refer to the entry for _NetShare_ and with a unit _VolPath_ for each.
1. "DefaultLabel" - If a label isn't specified at creation time of a volume for _nfs/nfs4 type volumes then this entry will be used to select the location for the volume.

# Volumes

1. Provisioned as a folder on the named network FS
1. /mnt/vmdk/<remote—dir-label>/<vol-path>/<vol-name>/
1. Volume _create_ mkdir’s the volume name folder if not already present

## Volume meta-data
Plugin maintains no in-memory state for either volumes or remote dirs. (except the configuration). Volume met-data listed below are generated when requested via a Get() from Docker,
1. _Location_ – remote-dir, path within remote-dir
1. _Type and version_ - volume type (NFS, EFS)
1. _Capacity used in FS blocks_
1. _Current usage count-
1. _Create date/time_
1. _Modified time_
1. _Access time_
1. _Access permissions_
1. _Use count_


# Volume options and Tenancy

## Volume create options
   - _access perms (ro, rw)_, remote-dir label, type (_file_ only allowed, if not specified defaults to _VMDK_ type)

## Tenancy
There is no support for multi-tenancy for NFS backed volumes, within the plugin. Instead, the admin manages this via the configuration of shared dirs to control which NFS datastores are used to place volumes for which containers or groups of containers(like service stacks for example).

# Authentication

Applies to NFS v4.0/v4.1 in case the admin has enabled authentication based access to remote dirs. The admin needs to provide the admin name and password to the plugin (via secrets). _This is TBD and will be elaborated once the functionality without authentication is completed._

# Future work

## Support EFS
Amazon;s EFS supports NFS v4.0/v4.1 and can be easily supported with the feature descibed above. This can be taken up once the above functionality is implemented.

## Data mover APIs

Support data mover APIs for all backing types of volumes supported in te plugin and hence allow migration/replication of volumes as desired between datastores.
