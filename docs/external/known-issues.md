---
title: Known Issues
---

This section lists the major known issues with vSphere Docker Volume Service. For a complete list of issues, please check our Github issues(https://github.com/vmware/docker-volume-vsphere/issues) page. If you notice an issue not listed in Github issues page, please do file a bug on the [Github repo](https://github.com/vmware/docker-volume-vsphere/issues)

-  Volume metadata file got deleted while removing volume from VM(placed on Esx2) which is in use by another VM(placed on Esx1) [#1191](https://github.com/vmware/docker-volume-vsphere/issues/1191). It's an ESX issue and will be available in the next vSphere release.
-  Full volume name with format like "volume@datastore" cannot be specified in the compose file for stack deployment. [#1315](https://github.com/vmware/docker-volume-vsphere/issues/1315). It is a docker compose issue and a workaround has been provided in the issue.
-  Volume creation using VFAT filesystem is not working currently. [#1327](https://github.com/vmware/docker-volume-vsphere/issues/1327)
-  Plugin fails to create volumes after installation on VM running boot2docker Linux. This is because open-vm-tools available for boot2docker doesn't install the vSockets driver and hence the plugin is unable to contact the ESX service. [#1744](https://github.com/vmware/docker-volume-vsphere/issues/1744)
-  Currently "vmdk-opsd stop" just stops (exits) the service forcefully. If there are operations in flight it could kill them in the middle of execution. This can potentially create inconsistencies in VM attachement, KV files or auth-db. [#1073](https://github.com/vmware/docker-volume-vsphere/issues/1073)

## Known Differences Between Linux And Windows Plugin
- Docker, by default, converts volume names to lower-case on Windows. Therefore, volume operations involving case-sensitive names will always be handled in lower case.