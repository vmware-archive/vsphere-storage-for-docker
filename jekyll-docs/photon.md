---
title: Photon Platform Integration
---
## Photon Platform: Introduction
Docker volumes on [Photon platform](https://vmware.github.io/photon-controller) are provisioned, managed entirely via the Photon platform [API](https://github.com/vmware/photon-controller/wiki/API). Docker volumes are placed on datastores as decided by the Photon platform. The Photon platform selects datastores from within those that have been explicitly configured by the user on hosts managed by the Photon Controller and include backends such as NFS, SAN, VSAN, vVol. Accessibility of volumes to VMs running on the hosts is therefore dependent on whether the datastores are shared among hosts configured to the Photon platform. Accessibility of docker volumes on Photon is governed by the tenancy model of Photon, where each volume is scoped to a project within a tenant. Volumes are scoped to a project and hence accessible to all VMs within the same project. In addition, the count of volumes and storage capacity allocated (in total) to docker volumes within a project are managed via resource limits defined by the resource ticket associated with the tenant that the project belongs to. The Photon platform admin must hence take care to configure sufficient storage capacity, taking into consideration the max. number of volumes and the typical sizes of those. Once a configuration has been created, the Photon platform seamlessly manages accounting the count and storage capacity consumed by volumes in a project.

## vDVS with Photon Platform

The docker volume plugin also supports Photon platform. The plugin supports all volume provisioning and managenment operations, defined by the Docker Volume plugin interface, on both platforms (vSphere and Photon).Creation of volumes in photon platform is done via the open Photon platform API using the Photon controller. 

## Configuration: Photon Driver

The configurations used by a driver while performing operations are read from a JSON file and the default location where it looks for it /etc/docker-volume-vsphere.conf. You can also override it to use  a different configuration file by providing --config option and the full path to the file. Finally the parameters passed on the CLI override the one from the configuration file. You need following configurations for photon driver in the configuration file.

```
{
    "Driver": "vmdk"
    "MaxLogAgeDays": 28,
    "MaxLogSizeMb": 100,
    "LogPath": "/var/log",
    "LogLevel": "info",
    "Target" : "http://<photon_controller_ip>:<target port>",
    "Project" : "<21-digit photon project ID>",
    "Host" : "<32-digit photon VM ID "
}
```
<table class="table table-striped table-hover ">
  <thead>
    <tr>
      <th>Parameter Name</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>target</td>
      <td>The endpoint of the Photon Controller with hostname and port in format: http://{photon_host}:{photon_port} </td>
    </tr>
    <tr>
      <td>project</td>
      <td>The project ID in Photon to which this host belongs.</td>
    </tr>
    <tr>
      <td>host</td>
      <td>The ID of docker host in Photon.</td>
    </tr>
</tbody>
</table>


<div class="panel panel-info">
  <div class="panel-heading">
    <h3 class="panel-title">Details of Photon Platform</h3>
  </div>
  <div class="panel-body">
     The project and VM IDs mentioned above can be obtained via the “photon” CLI, see https://github.com/vmware/photon-controller/wiki/Compile-Photon-Controller-CLI.
  </div>
</div> 

## Using the Service in Docker

### Creation and management of docker volumes
The docker volume commands are completely supported by vDVS plugin. This section demonstrates use of various commands with examples.


#### Size
You can specify the size of volume while creating a volume. Supported units of sizes are mb, gb and tb. By default if you don’t specify the size, a 100MB volume is created.

```
docker volume create --driver=photon --name=MyVolume -o size=10gb
```

#### File System Type (fstype)
You can specify the filesystem which will be used it to create the volumes. The docker plugin will look for existing filesystesm in /sbin/mkfs.fstype but if the specified filesystem is not found then it will return a list for which it has found mkfs. The default filesystem if not specified is ext4.

```
docker volume create --driver=photon --name=MyVolume -o size=10gb -o fstype=xfs
docker volume create --driver=photon --name=MyVolume -o size=10gb -o fstype=ext4 (default)

```

#### Flavour 
You can specify the flavour of persistent disk available in the Photon controller while creating a volume. The flavour in Photon implies certain limits on the volume being created.
More information about flavours is available [here](https://github.com/vmware/photon-controller/wiki/Flavors).

```
docker volume create --driver=photon --name=CloneVolume -o flavor=<Photon persistent disk flavor name>
```
### Listing Volumes
Docker volume list can be used to volume names & their DRIVER type
```
docker volume ls
DRIVER              VOLUME NAME
vsphere                MyVolume@vsanDatastore
vsphere                minio1@vsanDatastore
vsphere                minio2@vsanDatastore
photon                 redis-data@vsanDatastore
```
### Docker volume inspect
You can use `docker volume inspect` command to see attributes of a particular volume.

```
docker volume create —driver=photon —name=MyVolume -o size=2gb -o vsan-policy-name=myPolicy -o fstype=xfs
```
```
docker volume inspect MyVolume
[
    {
        "Driver": "photon",
        "Labels": {},
        "Mountpoint": "/mnt/vmdk/MyVolume",
        "Name": "MyVolume",
        "Options": {
            "fstype": "xfs",
            "size": "2gb"
        },
        "Scope": "global",
        "Status": {
            "access": "read-write",
            "capacity": {
                "allocated": "32MB",
                "size": "2GB"
            },
            "created": "Wed Mar  1 20:06:02 2017",
            "created by VM": "esx1_swarm01",
            "datastore": "vsanDatastore",
            "status": "detached"
        }
    }
]
```

### Remove volume
You can remove the volume with following command
```
# docker volume rm db_data
db_data
```


For more information about Photon plaform, please refer [Wiki page](https://github.com/vmware/photon-controller/wiki).
