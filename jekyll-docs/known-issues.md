---
title: Known Issues
---

This section lists the major known issues with vSphere Docker Volume Service. For complete list of issues please check our Github issues(https://github.com/vmware/docker-volume-vsphere/issues) page. If you notice an issue not listed in Github issues page, please do file a bug on the [Github repo](https://github.com/vmware/docker-volume-vsphere/issues)

-  Volume metadata file got deleted while removing volume from VM(placed on Esx2) which is in use by another VM(placed on Esx1) [#1191](https://github.com/vmware/docker-volume-vsphere/issues/1191). It's an ESX issue and will be available in the next vSphere release.

-  Full volume name with format like "volume@datastore" cannot be specified in the compose file for stack deployment. [#1315](https://github.com/vmware/docker-volume-vsphere/issues/1315). It is a docker compose issue and a workaround has been provided in the issue.
