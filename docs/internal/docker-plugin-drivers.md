---
title: Overview

---
## Overview
The Docker volume plugin supports the vSphere platform and provides corresponding driver for it. The plugin supports all volume provisioning and managenment operations, defined by the Docker Volume plugin interface. The corresponding driver allows users to provision and use VMDK backed volumes for containers in Docker.

## Docker vsphere volume driver
The Docker vsphere volume driver supports provisioning and managing docker volumes on a standalone or cluster of ESX servers via a service (ESX service) that's installed and runs on each server. Docker volumes are created and managed via publicly available VIM (Virtual Infrastructure Management) APIs on the ESX host.

## Configuring the Docker Volume Plugin
The docker volume plugin loads runtime options and values from a JSON configuration file (default `/etc/docker-volume-vsphere.conf` for Linux, `C:\ProgramData\docker-volume-vsphere\docker-volume-vsphere.conf` for Windows) on the host. The user can override the default configuration by providing a different configuration file, via the `--config` option, specifying the full path of the file. Options that are currently recognized include the below set. Options passed on the command line override those in the configuration file.

### Selecting the driver to handle volume operations
The docker volume plugin supports a driver, namely, `vsphere` for the vSphere platform. The usage of driver is specified as below in the [sample configuration](#sample-plugin-configuration).

### Options for logging
* LogLevel      - logging level for the plugin
* LogPath       - location where plugin log fils are created
* MaxLogSizeMb  - max. size of the plugin log file
* MaxLogAgeDays - number of days to retain plugin log files

## Sample plugin configuration
```
{
	"Driver": "vsphere",
	"MaxLogAgeDays": 28,
	"MaxLogSizeMb": 100,
	"LogPath": "/var/log/docker-volume-vsphere.log",
	"LogLevel": "info"
}
```
