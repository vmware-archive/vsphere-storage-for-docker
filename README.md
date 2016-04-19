[![Build
Status](https://ci.vmware.run/api/badges/vmware/docker-vmdk-plugin/status.svg)](https://ci.vmware.run/vmware/docker-vmdk-plugin)

# Docker VMDK Plugin

This repo hosts the docker volume plugin for VMware vSphere. The plugin allows
storage owned by vSphere to be managed and consumed via the [docker volume
plugin framework](https://docs.docker.com/engine/extend/plugins_volume/).

To read more about code development and testing read
[CONTRIBUTING.md](https://github.com/vmware/docker-vmdk-plugin/blob/master/CONTRIBUTING.md)

## Tested on

ESXi:

- 6.0
- 6.0 u1
- 6.0 u2

Docker: 1.9 and higher

VM:
- Ubuntu 14.04 64 bit (using systemd)
- Photon

The VM plugin code is tested against the listed enumerated above but it should
work against any 64 bit distro with systemd installed.

## Installation

In order to get going, pick the latest stable release (for now
only pre TP release is available) from
https://github.com/vmware/docker-vmdk-plugin/releases.

There are 2 components that need installing, backend service on ESX and the docker
plugin on the docker host.

### ESX Setup.

Install the vSphere side of code (vib or offline depot), [please refer to
vSphere documentation.](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)

Same sample options:
```
# Log on to ESX after copying the vib over and run
localcli software vib install --no-sig-check  -v /tmp/<vib_name>.vib
```
Or use the helper scripts part of the build and test infrasructure, refer
[CONTRIBUTING.md](https://github.com/vmware/docker-vmdk-plugin/blob/master/CONTRIBUTING.md)

### Plugin installation on the docker host (VM)

VM running docker needs the plugin installed. Use the deb or rpm file to
install the plugin. Plugin requires systemd for starting and stopping the
plugin. For manual steps not using rpm or deb file please refer
[CONTRIBUTING.md](https://github.com/vmware/docker-vmdk-plugin/blob/master/CONTRIBUTING.md)

```
# DEB
sudo dpkg -i docker-vmdk-plugin_v0.1.pre-tp_amd64.deb
# RPM
sudo rpm -ivh docker-vmdk-plugin-v0.1.pre_tp-1.x86_64.rpm
```

## Using the plugin.

```
# Docker commands to use plugin
docker volume create --driver=vmdk --name=MyVolume -o size=10gb
docker volume ls
docker volume inspect MyVolume docker run --name=my_container -it -v MyVolume:/mnt/myvol -w /mnt/myvol busybox sh
docker rm my_container docker volume rm MyVolume
```

# Plugin Overview

Native ESXi VMDK support for Docker Data Volumes.

When Docker runs in a VM under ESXi hypervisor, we allow Docker user to create
and use VMDK-based data volumes. Example:

```
docker volume create --driver=vmdk --name=MyStorage -o size=10gb
docker run --rm -it -v MyStorage:/mnt/data busybox sh
```

This will create a MyStorage.vmdk on the same datastore where Docker VM is
located. This vmdk will be attached to the Docker VM on "docker run" and the
containers can use this storage for data.

This repo contains guest code and ESXi code.

The docker-vmdk-plugin service runs in docker VM and talks to Docker Volume
Plugin API via Unix Sockets. It then relays requests via VMWare vSockets
host-guest communication to a dedicated service on ESXi.

The docker plugin code makes use of  vmdkops module  (found  in ./vmdkops) and
ESX python service (found in ./vmdkops-esxsrc).

The end results is that "docker volume create --drive vmdk" is capable of
creating VMDK disks on enclosing ESX host, and using the new volume auto
attaches and mounts the storage so it is immediately usable

# Demo To be continued...

# Contact

[CNA Storage](cna-storage <cna-storage@vmware.com>)
