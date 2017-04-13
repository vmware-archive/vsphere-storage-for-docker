
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
<table class="table table-striped table-hover ">
  <thead>
    <tr>
      <th>Parameter Name</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>driver</td>
      <td>The name of Driver – vsphere/vmdk for vSphere driver and photon for the photon driver</td>
    </tr>
    <tr>
      <td>MaxLogAgeDays</td>
      <td>The max number of days for which to keep logs on the machine.</td>
    </tr>
    <tr>
      <td>MaxLogSizeMb</td>
      <td>The maximum size of the log file</td>
    </tr>
    <tr>
      <td>LogPath</td>
      <td>The location at which loge file will be  created</td>
    </tr>    
    <tr>
      <td>LogLevel</td>
      <td>The verbosity of the log file can be one of INFO, DEBUG, ERROR (Please confirm log levels)</td>
    </tr>
</tbody>
</table>


* Q: Does the ESX service runs on every VM or you mean on a ESX server?
* Q: Are there defaults for Log related configs if not provided? Are they manadatory?
* Q: What happens if the MaxLogSize reaches before the MAxLogAgeDays? Also What happens after let’s say MaxLogSize is reached – is the old file deleted and new one created?


## Deployment strategies

As you might have noticed, the installation of plugins is fairly straightforward but the configurations for the plugin will vary from host to host and based on your topology. It is best to deploy the plugin and configuration files through a configuration management tool such as Ansible/Salt/Puppet etc. so that you can use configuration templates and produce the configurations files for each host based on knowledge of topology and architecture. 
