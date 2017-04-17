---
title: Known Issues
---

This section lists the major known issues with vSphere Docker Volume Service. For complete list of issues please check our Github issues(https://github.com/vmware/docker-volume-vsphere/issues) page. If you notice an issue not listed in Github issues page, please do file a bug on the [Github repo](https://github.com/vmware/docker-volume-vsphere)

-  Multi-tenancy feature is limited to single ESX [#1032](https://github.com/vmware/docker-volume-vsphere/issues/1032)
-  In some use cases (e.g. force power off) vmdk remains attached to a VM but not used by Docker. [#369](https://github.com/vmware/docker-volume-vsphere/issues/369)
