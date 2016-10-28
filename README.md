[![Build Status](https://ci.vmware.run/api/badges/vmware/docker-volume-vsphere/status.svg)](https://ci.vmware.run/vmware/docker-volume-vsphere)

Docker Volume Driver for vSphere
================================

This repo hosts the Docker Volume Driver for vSphere. Docker Volume Driver enables customers to address persistent storage requirements for Docker containers in vSphere environments. This plugin is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/plugins_volume/). Docker users can now consume vSphere Storage (vSAN, VMFS, NFS) to address persistency requirements of containerized cloud native apps using Docker Ecosystem.

To read more about code development and testing please read
[CONTRIBUTING.md](https://github.com/vmware/docker-volume-vsphere/blob/master/CONTRIBUTING.md)
as well as the
[FAQ on the project site](https://vmware.github.io/docker-volume-vsphere/user-guide/faq/).

## Download

**[Click here to download (Github releases)] (https://github.com/vmware/docker-volume-vsphere/releases).**

The download consists of 2 parts

1. ESX: The ESX code is packaged as a [vib or an offline depot] (http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)
2. VM Running Docker: The docker plugin is packaged as a deb or rpm file.
   * Photon/RedHat => Download RPM
   * Ubuntu => Download Deb.

Please pick the latest release and use the same version of ESX and VM release.

## Demos

The demos are located on the project [site](https://vmware.github.io/docker-volume-vsphere/) and [wiki](https://github.com/vmware/docker-volume-vsphere/wiki/Demos)

## Project Website

Documentation, FAQ and other content can be found @ [https://vmware.github.io/docker-volume-vsphere](https://vmware.github.io/docker-volume-vsphere)

## Installation Instructions

### On ESX

Install vSphere Installation Bundle (VIB).  [Please refer to
vSphere documentation.](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)

Install using localcli on an ESX node
```
esxcli software vib install --no-sig-check  -v /tmp/<vib_name>.vib
```

Make sure you provide the **absolute path** to the `.vib` file or the install will fail.

### On Docker Host (VM)

The Docker volume plugin requires the docker engine to be installed as a prerequisite. This requires
Ubuntu users to configure the docker repository and pull the `docker-engine` package from there.
Ubuntu users can find instructions [here](https://docs.docker.com/engine/installation/linux/ubuntulinux/).

[Docker recommends that the docker engine should start after the plugins.] (https://docs.docker.com/engine/extend/plugin_api/)

```
sudo dpkg -i <name>.deb # Ubuntu or deb based distros
sudo rpm -ivh <name>.rpm # Photon or rpm based distros
```

## Using Docker CLI

```
$ docker volume create --driver=vmdk --name=MyVolume -o size=10gb
$ docker volume ls
$ docker volume inspect MyVolume
$ docker run --rm -it -v MyVolume:/mnt/myvol busybox
$ cd /mnt/myvol # to access volume inside container, exit to quit
$ docker volume rm MyVolume
```

## Using ESXi Admin CLI
```
$ /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls
```

## Restarting Docker and Docker-Volume-vSphere plugin

The volume plugin needs to be started up before starting docker.

```
service docker stop
service docker-volume-vsphere restart
service docker start
```

using systemctl

```
systemctl stop docker
systemctl restart docker-volume-vsphere
systemctl start docker
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
* **Turning on debug logging**: stop the service and manually run with ``--log_level=debug` flag

**ESX Plugin logs**

* Log location: `/var/log/vmware/vmdk_ops.log`
* Config file location: `/etc/vmware/vmdkops/log_config.json`  See Python
logging config format for content details.
* **Turning on debug logging**: replace all 'INFO' with 'DEBUG' in config file, restart the service


## Tested on

VMware ESXi:
- 6.0
- 6.0 u1
- 6.0 u2

Docker: 1.9 and higher

Guest Operating System:
- [Photon 1.0] (https://vmware.github.io/photon/) (Includes open-vm-tools)
- Ubuntu 14.04 or higher (64 bit)
   - Needs Upstart or systemctl to start and stop the plugin
   - Needs [open vm tools or VMware Tools installed](https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=340) ```sudo apt-get install open-vm-tools```
- RedHat and CentOS

# Known Issues
1. Operations are serialized. Thus, when large volume is formatted, all plugin operations on the same Docker engine will be serialized behind it. [#35](/../../issues/35)
2. VM level snapshots do not include docker data volumes. [#60](/../../issues/60)
3. Exiting bug in Docker around cleanup if mounting of volume fails when -w command is passed. [Docker Issue #22564] (https://github.com/docker/docker/issues/22564)
4. VIB, RPM and Deb files are not signed.[#273](/../../issues/273)
5. Pre-GA releases of Photon can crash on attaching a Docker volume. This issue is resolved in [Photon Issue 455](https://github.com/vmware/photon/issues/455) (Photon GA). Workaround: `power off` the Photon VM, change  SCSI Adapter type from LSI Logic to PVSCSI, and `power on` the VM.

## Contact us

### Public
* [cna-storage@vmware.com](cna-storage <cna-storage@vmware.com>)
* [Telegram] (https://telegram.me/cnastorage)
* [Issues] (https://github.com/vmware/docker-volume-vsphere/issues)

### Internal
* [VMware Internal Slack] (https://vmware.slack.com/archives/docker-volume-vsphere)

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
- Ryan Kelly
    - [How to use the Docker Volume Driver for vSphere with Photon OS](http://www.vmtocloud.com/how-to-use-the-docker-volume-driver-for-vsphere-with-photon-os/)
