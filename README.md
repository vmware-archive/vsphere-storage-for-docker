[![Build Status](https://ci.vmware.run/api/badges/vmware/docker-volume-vsphere/status.svg)](https://ci.vmware.run/vmware/docker-volume-vsphere)

Docker Volume Driver for vSphere
================================

[![Join the chat at https://gitter.im/vmware/docker-volume-vsphere](https://badges.gitter.im/vmware/docker-volume-vsphere.svg)](https://gitter.im/vmware/docker-volume-vsphere?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

This repo hosts the Docker Volume Driver for vSphere. Docker Volume Driver enables customers to address persistent storage requirements for Docker containers in vSphere environments. This plugin is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/plugins_volume/). Docker users can now consume vSphere Storage (vSAN, VMFS, NFS) to address persistency requirements of containerized cloud native apps using Docker Ecosystem. 

To read more about code development and testing read
[CONTRIBUTING.md](https://github.com/vmware/docker-volume-vsphere/blob/master/CONTRIBUTING.md)

## Download

**[Click here to download (Github releases)] (https://github.com/vmware/docker-volume-vsphere/releases).**

The download consists of 2 parts

1. ESX: The ESX code is packaged as a [vib or an offline depot] (http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)
2. VM Running Docker: The docker plugin is packaged as a deb or rpm file.
   * Photon => Download RPM
   * Ubuntu => Download Deb.

Please pick the latest release and use the same version of ESX and VM release.

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
The relevant logging for debugging consists of

Docker logs: https://docs.docker.com/engine/admin/logging/overview/
```
/var/log/upstart/docker.log # Upstart
journalctl -fu docker.service # Journalctl/Systemd
```

VM Plugin logs
```
/var/log/docker-volume-vsphere.log
```

ESX Plugin logs
```
/var/log/vmware/vmdk_ops.log
```

## Tested on

VMware ESXi:
- 6.0
- 6.0 u1
- 6.0 u2

Docker: 1.9 and higher

Guest Operating System:
- [Photon 1.0 RC] (https://vmware.github.io/photon/) (Includes open-vm-tools)
- Ubuntu 14.04 or higher (64 bit)
   - Needs Upstart or systemctl to start and stop the plugin
   - Needs [open vm tools or VMware Tools installed](https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=340) ```sudo apt-get install open-vm-tools```

# Known Issues
1. Operations are serialized. Thus, if a large volume is created, other operations will block till the format is complete. [#35](/../../issues/35)
2. VM level snapshots do not include docker data volumes. [#60](/../../issues/60)
3. Exiting bug in Docker around cleanup if mounting of volume fails when -w command is passed. [Docker Issue #22564] (https://github.com/docker/docker/issues/22564)
4. VIB, RPM and Deb files are not signed.[#273](/../../issues/273)
5. Pre-GA releases of Photon can crash on attaching a Docker volume. This issue is resolved in [Photon Issue 455](https://github.com/vmware/photon/issues/455) (Photon GA). Workaround: `power off` the Photon VM, change  SCSI Adapter type from LSI Logic to PVSCSI, and `power on` the VM.

## Contact us

Please let us know what you think! Contact us at

* [cna-storage@vmware.com](cna-storage <cna-storage@vmware.com>)
* [Slack] (https://vmware.slack.com/archives/docker-volume-vsphere)
* [Telegram] (https://telegram.me/cnastorage)
* [Issues] (https://github.com/vmware/docker-volume-vsphere/issues)
