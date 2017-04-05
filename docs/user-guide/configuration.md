
The docker volume plugin at the moment supports vSphere and Photon platforms. All the volume provisioning and management operations are supported on both the platforms. Both the platforms can be used to provision VMDK based volumes and can be attached to containers. 

The configurations used by a driver while performing operations are read from a JSON file and the default location where it looks for it /etc/docker-volume-vsphere.conf. You can also override it to use  a different configuration file by providing --config option and the full path to the file. Finally the parameters passed on the CLI override the one from the configuration file.

## vSphere Docker volume Driver

The vSphere volume driver can be used on a standalone or cluster of ESX servers via the ESX service. ESX service needs to be installed and running on all servers. The volumes are created on ESX host using the VIM (Virtual Infrastructure Management) APIs.

The configuration for vSphere docker volume driver require the driver type which can be one of vsphere or vmdk (For backwards compatibility). You can also provide the log related configurations.

```
{
    "Driver": "vsphere"
    "MaxLogAgeDays": 28,
    "MaxLogSizeMb": 100,
    "LogPath": "/var/log/docker-volume-vsphere.log",
    "LogLevel": "info"
}
```
| Paramater Name  		| Description 		|
| ---------------------:|:------------------|
| driver 				|The name of Driver – vsphere/vmdk for vSphere driver and photon for the photon driver|
| MaxLogAgeDays			|The max number of days for which to keep logs on the machine.|
|MaxLogSizeMb			|The maximum size of the log file|
|LogPath|The location at which loge file will be  created|
|LogLevel|The verbosity of the log file can be one of INFO, DEBUG, ERROR (Please confirm log levels)|


* Q: Does the ESX service runs on every VM or you mean on a ESX server?
* Q: Are there defaults for Log related configs if not provided? Are they manadatory?
* Q: What happens if the MaxLogSize reaches before the MAxLogAgeDays? Also What happens after let’s say MaxLogSize is reached – is the old file deleted and new one created?


## Photon Driver
The docker photon volume driver creates and manages the volumes on a photon platform. Photon platform API is used to create and manage volumes. You need two additional configurations for photon driver in the configuration file.

```
{
    "Driver": "<driver name - vsphere/vmdk/photon>"
    "MaxLogAgeDays": 28,
    "MaxLogSizeMb": 100,
    "LogPath": "/var/log/docker-volume-vsphere.log",
    "LogLevel": "info",
    "Target" : "http://<photon_controller_ip>:<target port>",
    "Project" : "<21-digit photon project ID>",
    "Host" : "<32-digit photon VM ID "
}
```
| Paramater Name  		| Description 		|
| ---------------------:|:------------------|
| target 				|The endpoint of the Photon Controller with hostname and port in format: http://<photon_host>:<photon_port>|
|project|The project ID in Photon to which this host belongs.|
|host|The ID of docker host in Photon.|

## Deployment strategies

As you might have noticed, the installation of plugins is fairly straightforward but the configurations for the plugin will vary from host to host and based on your topology. It is best to deploy the plugin and configuration files through a configuration management tool such as Ansible/Salt/Puppet etc. so that you can use configuration templates and produce the configurations files for each host based on knowledge of topology and architecture. 
