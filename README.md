[![Build Status](https://ci.vmware.run/api/badges/vmware/docker-volume-vsphere/status.svg)](https://ci.vmware.run/vmware/docker-volume-vsphere)
[![Go Report Card](https://goreportcard.com/badge/github.com/vmware/docker-volume-vsphere)](https://goreportcard.com/report/github.com/vmware/docker-volume-vsphere)
[![Join the chat at https://gitter.im/vmware/docker-volume-vsphere](https://badges.gitter.im/vmware/docker-volume-vsphere.svg)](https://gitter.im/vmware/docker-volume-vsphere?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Docker Pulls](https://img.shields.io/badge/docker-pull-blue.svg)](https://store.docker.com/plugins/vsphere-docker-volume-service?tab=description)
[![VIB_Download](https://api.bintray.com/packages/vmware/vDVS/VIB/images/download.svg)](https://bintray.com/vmware/vDVS/VIB/_latestVersion)
[![Windows Plugin](https://img.shields.io/badge/Windows%20Plugin-latest-blue.svg)](https://bintray.com/vmware/vDVS/vDVS_Windows/_latestVersion)

# vSphere Docker Volume Service

vSphere Docker Volume Service (vDVS) enables customers to address persistent storage requirements for Docker containers in vSphere environments. This service is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/). Docker users can now consume vSphere Storage (vSAN, VMFS, NFS) to stateful containers using Docker.

[<img src="https://github.com/vmware/docker-volume-vsphere/blob/master/docs/misc/Docker%20Certified.png" width="180" align="right">](https://store.docker.com/plugins/vsphere-docker-volume-service?tab=description)vDVS is Docker Certified to use with Docker Enterprise Edition and available in [Docker store](https://store.docker.com/plugins/e15dc9d5-e20e-4fb8-8876-9615e6e6e852?tab=description).

If you would like to contribute then please check out 
[CONTRIBUTING.md](https://github.com/vmware/docker-volume-vsphere/blob/master/CONTRIBUTING.md)
& [FAQ on the project site](http://vmware.github.io/docker-volume-vsphere/documentation/faq.html).

## Documentation

Detailed documentation can be found on our [GitHub Documentation Page](http://vmware.github.io/docker-volume-vsphere/documentation/).

## Downloads

**Download releases from [Github releases](https://github.com/vmware/docker-volume-vsphere/releases) page**

The download consists of 2 parts:

1. VIB (vDVS driver): The ESX code is packaged as [a vib or an offline depot](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)
2. Managed plugin (vDVS plugin): Plugin is available on [Docker store](https://store.docker.com/plugins/e15dc9d5-e20e-4fb8-8876-9615e6e6e852?tab=description).

Please check [vDVS Installation User Guide](http://vmware.github.io/docker-volume-vsphere/documentation/install.html) to get started. To ensure compatibility, make sure to use the same version of vDVS driver (on ESX) and managed plugin (on Docker host VM).

## Supported Platforms

**ESXi:** 6.0 and above<br />
**Docker (Linux):** 1.12 and higher (Recommended 1.13/17.03 and above to use managed plugin)<br />
**Docker (Windows):** 1.13/17.03 and above (Windows containers mode only)

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

**vDVS Plugin logs**

* Log location (Linux): `/var/log/docker-volume-vsphere.log`
* Log location (Windows): `C:\Windows\System32\config\systemprofile\AppData\Local\docker-volume-vsphere\logs\docker-volume-vsphere.log`
* Config file location (Linux): `/etc/docker-volume-vsphere.conf`.
* Config file location (Windows): `C:\ProgramData\docker-volume-vsphere\docker-volume-vsphere.conf`.
* This JSON-formatted file controls logs retention, size for rotation
 and log location. Example:
```
 {"MaxLogAgeDays": 28,
 "MaxLogSizeMb": 100,
 "LogPath": "/var/log/docker-volume-vsphere.log"}
```
* **Turning on debug logging**:

   - **Package user (DEB/RPM installation)**: Stop the service and manually run with `--log_level=debug` flag

   - **Managed plugin user**: You can change the log level by passing `VDVS_LOG_LEVEL` key to `docker plugin install`.

   - **Managed plugin user**: Set the group ID to use for the plugin socket file via the VDVS_SOCKET_GID env. variable.

      e.g.
      ```
      docker plugin install --grant-all-permissions --alias vsphere vmware/docker-volume-vsphere:latest VDVS_LOG_LEVEL=debug VDVS_SOCKET_GID=<group name>
      ```

**vDVS Driver logs**

* Log location: `/var/log/vmware/vmdk_ops.log`
* Config file location: `/etc/vmware/vmdkops/log_config.json`  See Python
logging config format for content details.
* **Turning on debug logging**: replace all 'INFO' with 'DEBUG' in config file, restart the service

Please refer [vDVS configuration page](http://vmware.github.io/docker-volume-vsphere/documentation/configuration.html) for detailed steps.

## Tested on

**VMware ESXi**:
- 6.0, 6.0U1, 6.0U2
- 6.5

**Guest Operating System**:
- Ubuntu 14.04 or higher (64 bit)
   - Needs Upstart or systemctl to start and stop the service
   - Needs [open vm tools or VMware Tools installed](https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=340) ```sudo apt-get install open-vm-tools```
- RedHat and CentOS
- Windows Server 2016 (64 bit)
- [Photon 1.0, Revision 2](https://github.com/vmware/photon/wiki/Downloading-Photon-OS#photon-os-10-revision-2-binaries) (v4.4.51 or later)

**Docker (Linux)**: 1.12 and higher (Recommended 1.13/17.03 and above to use managed plugin)
**Docker (Windows)**: 1.13/17.03

## References

* **Known Issues**: Please check [vDVS known issue page](http://vmware.github.io/docker-volume-vsphere/documentation/known-issues.html) to find out about known issues.

* **Contact us**: Please [click here](http://vmware.github.io/docker-volume-vsphere/documentation/contactus.html) for requesting any feature or reporting a product issue.

 * **Blogs**: Please check our [vDVS blog page](http://vmware.github.io/docker-volume-vsphere/documentation/blogs.html).
