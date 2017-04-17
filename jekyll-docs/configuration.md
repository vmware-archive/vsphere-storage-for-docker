---
title: Configuration
---

vSphere Docker Volume Service consists of two components the VIB and Docker plugin. Only the Docker plugin requires configuration which is described in this section. The docker volume plugin uses VMDK based volumes and are attached to container.

The configurations used by a driver while performing operations are read from a JSON file and the default location where it looks for it /etc/docker-volume-vsphere.conf. You can also override it to use  a different configuration file by providing --config option and the full path to the file. Finally the parameters passed on the CLI override the one from the configuration file.

The configuration for  Docker volume plugin require the driver type which can be one of vsphere or vmdk (For backwards compatibility). You can also provide the log related configurations.

```
{
    "Driver": "vsphere"
    "MaxLogAgeDays": 28,
    "MaxLogSizeMb": 100,
    "LogPath": "/var/log",
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
      <td>The name of Driver – vsphere for vSphere driver</td>
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
