---
title: Known Issues
---


<span style="color:red">**Note for Photon OS user**:  Managed plugin installation is not supported due to known Photon OS [Issue](https://github.com/vmware/photon/issues/640).</span> In the meantime, you can try one of alternative proposed in [#1215](https://github.com/vmware/docker-volume-vsphere/issues/1215#issuecomment-298841769)  (manually upgrade to latest Docker release or use RPM)

This section lists the major known issues with vSphere Docker Volume Service. For complete list of issues please check our Github issues(https://github.com/vmware/docker-volume-vsphere/issues) page. If you notice an issue not listed in Github issues page, please do file a bug on the [Github repo](https://github.com/vmware/docker-volume-vsphere/issues)

-  Multi-tenancy feature is limited to single ESX [#1032](https://github.com/vmware/docker-volume-vsphere/issues/1032)
-  In some use cases (e.g. force power off) vmdk remains attached to a VM but not used by Docker. [#369](https://github.com/vmware/docker-volume-vsphere/issues/369)
