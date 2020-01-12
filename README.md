[![Build Status](https://ci.vmware.run/api/badges/vmware/vsphere-storage-for-docker/status.svg)](https://ci.vmware.run/vmware/vsphere-storage-for-docker)
[![Go Report Card](https://goreportcard.com/badge/github.com/vmware/vsphere-storage-for-docker)](https://goreportcard.com/report/github.com/vmware/vsphere-storage-for-docker)
[![Docker Pulls](https://img.shields.io/badge/docker-pull-blue.svg)](https://store.docker.com/plugins/vsphere-docker-volume-service?tab=description)
[![VIB_Download](https://api.bintray.com/packages/vmware/vDVS/VIB/images/download.svg)](https://bintray.com/vmware/vDVS/VIB/_latestVersion)
[![Windows Plugin](https://img.shields.io/badge/Windows%20Plugin-latest-blue.svg)](https://bintray.com/vmware/vDVS/VDVS_Windows/_latestVersion)

# VMware has ended active development of this project, this repository will no longer be updated.
# vSphere Storage for Docker

vSphere Storage for Docker enables customers to address persistent storage requirements for Docker containers in vSphere environments. This service is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/). Docker users can now consume vSphere Storage (vSAN, VMFS, NFS, VVol) to stateful containers using Docker.

[<img src="https://github.com/vmware/vsphere-storage-for-docker/blob/master/docs/misc/Docker%20Certified.png" width="180" align="right">](https://store.docker.com/plugins/vsphere-docker-volume-service?tab=description)vSphere Storage for Docker is Docker Certified to use with Docker Enterprise Edition and available in [Docker store](https://store.docker.com/plugins/e15dc9d5-e20e-4fb8-8876-9615e6e6e852?tab=description).

If you would like to contribute then please check out
[CONTRIBUTING.md](https://github.com/vmware/vsphere-storage-for-docker/blob/master/CONTRIBUTING.md)
& [FAQ on the project site](http://vmware.github.io/vsphere-storage-for-docker/documentation/faq.html).

## Documentation

Detailed documentation can be found on our [GitHub Documentation Page](http://vmware.github.io/vsphere-storage-for-docker/documentation/).

## Downloads

**Download releases from [Github releases](https://github.com/vmware/vsphere-storage-for-docker/releases) page**

The download consists of 2 parts:

1. VIB (VDVS driver): The ESX code is packaged as [a vib or an offline depot](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)
2. Managed plugin (VDVS plugin): Plugin is available on [Docker store](https://store.docker.com/plugins/e15dc9d5-e20e-4fb8-8876-9615e6e6e852?tab=description).

Please check [VDVS Installation User Guide](http://vmware.github.io/vsphere-storage-for-docker/documentation/install.html) to get started. To ensure compatibility, make sure to use the same version of driver (on ESX) and managed plugin (on Docker host VM) for vSphere Storage for Docker.

## Supported Platforms

**ESXi:** 6.0U2 and above<br />
**Docker (Linux):** 17.06.1 and above to use managed plugin<br />
**Docker (Windows):** 17.06 and above (Windows containers mode only)<br />
**Guest Operating System**:
- Ubuntu 14.04 or higher (64 bit)
   - Needs Upstart or systemctl to start and stop the service
   - Needs [open vm tools or VMware Tools installed](https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=340) ```sudo apt-get install open-vm-tools```
- RedHat 6.9 or higher (64 bit)
- Windows Server 2016 (64 bit)
- [Photon 1.0, Revision 2](https://github.com/vmware/photon/wiki/Downloading-Photon-OS#photon-os-10-revision-2-binaries) (v4.4.51 or later), [Photon 2.0](https://github.com/vmware/photon/wiki/Downloading-Photon-OS#photon-os-20-ga-binaries)<br />

## Logging
The relevant logging for debugging consists of the following:
* Docker Logs
* Plugin logs - VM (docker-side)
* Plugin logs - ESX (server-side)

**Docker logs**: see https://docs.docker.com/engine/admin/logging/overview/
```
/var/log/upstart/docker.log # Upstart
journalctl -fu docker.service # Journalctl/Systemd
```

**VDVS Plugin logs**

* Log location (Linux): `/var/log/vsphere-storage-for-docker.log`
* Log location (Windows): `C:\Windows\System32\config\systemprofile\AppData\Local\vsphere-storage-for-docker\logs\vsphere-storage-for-docker.log`
* Config file location (Linux): `/etc/vsphere-storage-for-docker.conf`.
* Config file location (Windows): `C:\ProgramData\vsphere-storage-for-docker\vsphere-storage-for-docker.conf`.
* This JSON-formatted file controls logs retention, size for rotation
 and log location. Example:
```
 {"MaxLogAgeDays": 28,
 "MaxLogFiles": 10,
 "MaxLogSizeMb": 10,
 "LogPath": "/var/log/vsphere-storage-for-docker.log"}
```
* **Turning on debug logging**:

   - **Package user (DEB/RPM installation)**: Stop the service and manually run with `--log_level=debug` flag

   - **Managed plugin user**: You can change the log level by passing `VDVS_LOG_LEVEL` key to `docker plugin install`.

   - **Managed plugin user**: Set the group ID to use for the plugin socket file via the VDVS_SOCKET_GID env. variable.

      e.g.
      ```
      docker plugin install --grant-all-permissions --alias vsphere vmware/vsphere-storage-for-docker:latest VDVS_LOG_LEVEL=debug VDVS_SOCKET_GID=<group name>
      ```

**VDVS Driver logs**

* Log location: `/var/log/vmware/vmdk_ops.log`
* Config file location: `/etc/vmware/vmdkops/log_config.json`  See Python
logging config format for content details.
* **Turning on debug logging**: replace all 'INFO' with 'DEBUG' in config file, restart the service

Please refer [VDVS configuration page](http://vmware.github.io/vsphere-storage-for-docker/documentation/configuration.html) for detailed steps.

## References

* **Known Issues**: Please check [VDVS known issue page](http://vmware.github.io/vsphere-storage-for-docker/documentation/known-issues.html) to find out about known issues.

* **Contact us**: Please [click here](http://vmware.github.io/vsphere-storage-for-docker/documentation/contactus.html) for requesting any feature or reporting a product issue.

 * **Blogs**: Please check our [VDVS blog page](http://vmware.github.io/vsphere-storage-for-docker/documentation/blogs.html).
