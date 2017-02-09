[TOC]
The Docker volume plugin supports the below platforms and the corresponding drivers for those. The plugin supports all volume provisioning and managenment operations, defined by the Docker Volume plugin interface, on both platforms.

1. Vmdk
2. Photon

<script type="text/javascript" src="https://asciinema.org/a/80417.js" id="asciicast-80417" async></script>

## Docker Vmdk volume driver 
The Docker Vmdk volume driver supports provisioning and managing docker volumes on a standalone or cluster of ESX servers via a service (ESX service) that's installed and runs on each server. Docker volumes are created and managed via publicly available VIM (Virtual Infrastructure Management) APIs on the ESX host.

## Docker Photon volume driver 
The Docker Photon Volume driver supports provisioning and managing docker volumes on a Photon platform consisting of a cluster of ESX hosts managed via a Photon controller instance. Docker volumes are created and managed entirely via the open Photon platform API via the Photon controller.

The Docker Volume plugin can support either or both types of volumes, as required, on a given Docker host.

## Configuring the Docker Volume Plugin
The docker volume plugin is designed to load run time options and values from a json configuration file (default /etc/docker-volume-vsphere.conf) on the host. The user can also provide a configuration file, via the "--config" option, specifying the full path of the file. The file contains the values for run time options used by the plugin. Options that are currently recognized include the below set. Options provided on the command line will override those in the configuration file.

### Options for the photon volume driver
Target    - URL at which to contact the Photon Controller
Project   - project ID in Photon to which the docker host belongs
Host      - ID of the docker host VM in Photon

### Options for logging
LogLevel      - logging level for the plugin
LogPath       - location where plugin log fils are created
MaxLogSizeMb  - max. size of the plugin log file
MaxLogAgeDays - number of days to retain plugin log files

## Sample plugin configuration, the "Target", "Project" and "Host" options are for photon only.
{
	"MaxLogAgeDays": 28,
	"MaxLogSizeMb": 100,
	"LogPath": "/var/log/docker-volume-vsphere.log",
	"LogLevel": "info",
	"Target" : "http://<photon_controller_ip>:<target port>",
	"Project" : "<21-digit photon project ID>",
	"Host" : "<32-digit photon VM ID "
}

Note:
1. The current version of the Photon volume plugin doesn't support authentication as yet, this will be added up in a subsequent release.
2. The "target port" is set to "9000" if authentication isn't enabled in photon controller, else port "443" is used.
3. Both project and VM IDs mentioned above can be obtained via the "photon" CLI, see https://github.com/vmware/photon-controller/wiki/Compile-Photon-Controller-CLI.
