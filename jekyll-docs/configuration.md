---
title: Configuration
---

vSphere Docker Volume Service consists of two components the VIB and Docker plugin. The VIB configuration is driver configuration while the plugin configuration resides on individual VMs where plugin is installed.

## Driver configuration

The configurations used by a driver while performing operations are read from a JSON file that by default is located at /etc/vmware/vmdkops/log_config.json. The configuration is largely around logging.


```
{
    "handlers": {
        "rotate_file": {
            "level": "INFO",
            "maxBytes": 1048576,
            "formatter": "standard",
            "class": "logging.handlers.RotatingFileHandler",
            "filename": "/var/log/vmware/vmdk_ops.log",
            "backupCount": 1,
            "encoding": "utf8"
        }
    },
    "formatters": {
        "standard": {
            "format": "%(asctime)-12s %(process)d [%(threadName)s] [%(levelname)-7s] %(message)s",
            "datefmt": "%x %X"
        }
    },
    "version": 1,
    "loggers": {
        "": {
            "level": "INFO",
            "handlers": [
                "rotate_file"
            ]
        }
    },
    "disable_existing_loggers": false,
    "info": [
        "Logging configuration for vmdk_opsd service, in python logging config format.",
        "",
        "'level' defines verbosity and could be DEBUG, INFO, WARNING, ERROR, CRITICAL",
        "'maxBytes' and 'backupCount' define max log size and number of log backup files kept",
        "For more, see https://docs.python.org/2/library/logging.config.html#logging-config-dictschema",
        "",
        "Do NOT change 'rotate_file' name in handlers - it is used in code to locate the log file."
    ]
}
```

## Plugin Configuration

The configuration for  Docker volume plugin can be provided via a JSON formatted configuration file and on the command line (only when the plugin is manually executed vs. as a managed plugin by docker). The plugin can be provided the driver type (which must be set to be "vsphere" (default)), log configuration and the group-id used to create the plugin socket file. The log configuration includes the log level and the location where the log file is located. The configuration file is by default located at /etc/vsphere-storage-for-docker.conf. The configuration file to use can also be specified by providing the --config option and the full path to the file. Finally the parameters passed on the commandline override those read from the configuration file.

```
{
    "Driver": "vsphere",
    "MaxLogAgeDays": 28,
    "MaxLogFiles": 10,
    "MaxLogSizeMb": 10,
    "LogPath": "/var/log/vsphere-storage-for-docker.log",
    "LogLevel": "info",
    "GroupID": "root"  (default or alternatively use "docker" as the group ID)
}

The group ID is "root" by default and works fine with most Linux OS based Docker hosts, the alternative to use "docker" as the group ID is needed for coreOS hosts.
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
      <td>The name of the driver – vsphere</td>
    </tr>
    <tr>
      <td>MaxLogAgeDays</td>
      <td>The max number of days for which to keep logs on the machine.</td>
    </tr>
    <tr>
      <td>MaxLogFiles</td>
      <td>The max number of backup log files retains on the machine.</td>
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
      <td>The verbosity of the log file can be one of info, debug, error, warn etc.</td>
    </tr>
</tbody>
</table>
