[![Build Status](https://ci.vmware.run/api/badges/vmware/docker-volume-vsphere/status.svg)](https://ci.vmware.run/vmware/docker-volume-vsphere)
[![Go Report Card](https://goreportcard.com/badge/github.com/vmware/docker-volume-vsphere)](https://goreportcard.com/report/github.com/vmware/docker-volume-vsphere)
[![Join the chat at https://gitter.im/vmware/docker-volume-vsphere](https://badges.gitter.im/vmware/docker-volume-vsphere.svg)](https://gitter.im/vmware/docker-volume-vsphere?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) [![Docker Pulls](https://img.shields.io/badge/docker-pull-blue.svg)](https://store.docker.com/plugins/vsphere-docker-volume-service?tab=description) [![VIB_Download](https://api.bintray.com/packages/vmware/vDVS/VIB/images/download.svg)](https://bintray.com/vmware/vDVS/VIB/_latestVersion)


# vSphere Docker Volume Service

vSphere Docker Volume Service (vDVS) enables customers to address persistent storage requirements for Docker containers in vSphere environments. This service is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/). Docker users can now consume vSphere Storage (vSAN, VMFS, NFS) to stateful containers using Docker.

[<img src="https://github.com/vmware/docker-volume-vsphere/blob/master/docs/misc/Docker%20Certified.png" width="180" align="right">](https://store.docker.com/plugins/vsphere-docker-volume-service?tab=description)vDVS is Docker Certified to use with Docker Enterprise Edition and available in [Docker store](https://store.docker.com/plugins/e15dc9d5-e20e-4fb8-8876-9615e6e6e852?tab=description).

To read more about code development and testing please read
[CONTRIBUTING.md](https://github.com/vmware/docker-volume-vsphere/blob/master/CONTRIBUTING.md)
as well as the
[FAQ on the project site](http://vmware.github.io/docker-volume-vsphere/documentation/faq.html).

## Detailed documentation

Detailed documentation can be found on our [GitHub Documentation Page](http://vmware.github.io/docker-volume-vsphere/documentation/).

## Download

**[Click here to download from Github releases](https://github.com/vmware/docker-volume-vsphere/releases)**

The download consists of 2 parts:

1. VIB (vDVS driver): The ESX code is packaged as [a vib or an offline depot](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)
2. Managed plugin (vDVS plugin): Plugin is available on [Docker store](https://store.docker.com/plugins/e15dc9d5-e20e-4fb8-8876-9615e6e6e852?tab=description).

Pick the latest release and use the same version of vDVS plugin (on Docker host VM) and driver (on ESX).

## Demos

The demos are located on the project [site](https://vmware.github.io/docker-volume-vsphere/documentation) and [wiki](https://github.com/vmware/docker-volume-vsphere/wiki/Demos)

## Project Website

Project page is located @ [https://vmware.github.io/docker-volume-vsphere/](https://vmware.github.io/docker-volume-vsphere). Documentation, FAQ and other content can be found @ [https://vmware.github.io/docker-volume-vsphere/documentation](https://vmware.github.io/docker-volume-vsphere/documentation)

## Supported Platform

ESXi: 6.0 and above
Docker: 1.12 and higher (Recommended 1.13/17.03 and above to use managed plugin)

## Installation Instructions

### On ESX

Install vSphere Installation Bundle (VIB).  [Please refer to
vSphere documentation.](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)

Install using localcli on an ESX node
```
esxcli software vib install -v /tmp/<vib_name>.vib
```

Make sure you provide the **absolute path** to the `.vib` file or the install will fail.

### On Docker Host (VM)
#### Managed Plugin
1. Please make sure to uninstall older releases of DEB/RPM using following commands.
```
sudo dpkg -r docker-volume-vsphere # Ubuntu or deb based distros
sudo rpm -e docker-volume-vsphere # Photon or rpm based distros
```
2. Docker service needs to be restarted until [Issue #32635](https://github.com/docker/docker/issues/32635) is resolved.
```
systemctl restart docker
```

**For Docker 1.13 and above**, install managed plugin from Docker Store.
```
docker plugin install --grant-all-permissions --alias vsphere vmware/docker-volume-vsphere:latest
```

#### Using DEB/RPM
**For Docker 1.12 and earlier**, use DEB or RPM package.
```
sudo dpkg -i <name>.deb # Ubuntu or deb based distros
sudo rpm -ivh <name>.rpm # Photon or rpm based distros
```
**Note**: DEB/RPM packages will be deprecated going forward and will not be available.

## Using Docker CLI
Refer to [tenancy
documentation](http://vmware.github.io/docker-volume-vsphere/documentation/tenancy.html) for setting up tenants.
```
# To select datastore use --name=MyVolume@<Datastore Name>
$ docker volume create --driver=vsphere --name=MyVolume -o size=10gb
$ docker volume ls
$ docker volume inspect MyVolume
# To select datastore use MyVolume@<Datastore Name>
$ docker run --rm -it -v MyVolume:/mnt/myvol busybox
$ cd /mnt/myvol # to access volume inside container, exit to quit
$ docker volume rm MyVolume
```

## Using ESXi Admin CLI
```
$ /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
```

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

**VM (Docker-side) Plugin logs**

* Log location: `/var/log/docker-volume-vsphere.log`
* Config file location: `/etc/docker-volume-vsphere.conf`.
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

      e.g.
      ```
      docker plugin install --grant-all-permissions --alias vsphere vmware/docker-volume-vsphere:latest VDVS_LOG_LEVEL=debug
      ```

**ESX Plugin logs**

* Log location: `/var/log/vmware/vmdk_ops.log`
* Config file location: `/etc/vmware/vmdkops/log_config.json`  See Python
logging config format for content details.
* **Turning on debug logging**: replace all 'INFO' with 'DEBUG' in config file, restart the service


## Tested on

**VMware ESXi**:
- 6.0, 6.0U1, 6.0U2
- 6.5

**Guest Operating System**:
- Ubuntu 14.04 or higher (64 bit)
   - Needs Upstart or systemctl to start and stop the service
   - Needs [open vm tools or VMware Tools installed](https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=340) ```sudo apt-get install open-vm-tools```
- RedHat and CentOS
- [Photon 1.0, Revision 2](https://github.com/vmware/photon/wiki/Downloading-Photon-OS#photon-os-10-revision-2-binaries) (v4.4.51 or later)

**Docker**: 1.12 and higher (Recommended 1.13/17.03 and above to use managed plugin)

# Known Issues

-  Volume metadata file got deleted while removing volume from VM(placed on Esx2) which is in use by another VM(placed on Esx1) [#1191](https://github.com/vmware/docker-volume-vsphere/issues/1191). It's an ESX issue and will be available in the next vSphere release.
-  Full volume name with format like "volume@datastore" cannot be specified in the compose file for stack deployment. [#1315](https://github.com/vmware/docker-volume-vsphere/issues/1315). It is a docker compose issue and a workaround has been provided in the issue.

## Contact us

### Public
* [containers@vmware.com](containers@vmware.com)
* [Issues](https://github.com/vmware/docker-volume-vsphere/issues)
* [![Join the chat at https://gitter.im/vmware/docker-volume-vsphere](https://badges.gitter.im/vmware/docker-volume-vsphere.svg)](https://gitter.im/vmware/docker-volume-vsphere?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
* [VMware Code Slack](https://vmwarecode.slack.com/archives/docker-volume-vsphere) requires [joining VMware Code](https://code.vmware.com/web/code/join)

# Blogs

- Cormac Hogan
    - Overview
        - [Docker Volume Driver for vSphere](http://cormachogan.com/2016/06/01/docker-volume-driver-vsphere/)
        - [Docker Volume Driver for vSphere – short video](http://cormachogan.com/2016/06/03/docker-volume-driver-vsphere-short-video/)
    - Docker + VSAN
        - [Docker Volume Driver for vSphere on Virtual SAN](http://cormachogan.com/2016/06/09/docker-volume-driver-vsphere-virtual-san-vsan/)
        - [Using vSphere docker volume driver to run Project Harbor on VSAN](http://cormachogan.com/2016/07/29/using-vsphere-docker-volume-driver-run-project-harbor-vsan/)
        - [Docker Volume Driver for vSphere using policies on VSAN](http://cormachogan.com/2016/09/26/docker-volume-driver-vsphere-using-policies-vsan-short-video/)
    - 0.7 Release Overview
        - [Some nice enhancements to Docker Volume Driver for vSphere v0.7](http://cormachogan.com/2016/10/06/nice-enhancements-docker-volume-driver-vsphere-v0-7/)
- William Lam
    - [Getting Started with Tech Preview of Docker Volume Driver for vSphere - updated](http://www.virtuallyghetto.com/2016/05/getting-started-with-tech-preview-of-docker-volume-driver-for-vsphere.html)
- Virtual Blocks
    - [vSphere Docker Volume Driver Brings Benefits of vSphere Storage to Containers](https://blogs.vmware.com/virtualblocks/2016/06/20/vsphere-docker-volume-driver-brings-benefits-of-vsphere-storage-to-containers/)
    - [vSphere Docker Volume Service is now Docker Certified!](https://blogs.vmware.com/virtualblocks/2017/03/29/vsphere-docker-volume-service-now-docker-certified/)
- Ryan Kelly
    - [How to use the Docker Volume Driver for vSphere with Photon OS](http://www.vmtocloud.com/how-to-use-the-docker-volume-driver-for-vsphere-with-photon-os/)
