[![Build Status](https://ci.vmware.run/api/badges/vmware/docker-volume-vsphere/status.svg)](https://ci.vmware.run/vmware/docker-volume-vsphere)

Docker Volume Driver for vSphere
================================

This repo hosts the Docker Volume Driver for vSphere. The plugin integrated with [docker volume
plugin framework](https://docs.docker.com/engine/extend/plugins_volume/) will help customers address persistence storage requirements of docker containers backed by vSphere storage (vSAN, VMFS, NFS etc).

To read more about code development and testing read
[CONTRIBUTING.md](https://github.com/vmware/docker-volume-vsphere/blob/master/CONTRIBUTING.md)


## Download

We use [Github releases] (https://github.com/vmware/docker-volume-vsphere/releases).

The download consists of 2 parts

1. The ESX side code packaged as a [vib or an offline depot] (http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)
2. The VM side docker plugin packaged as a deb or rpm file.

Please pick the latest release and use the same version of ESX and VM release.

## Contact us

Please let us know what you think! Contact us at

* [cna-storage@vmware.com](cna-storage <cna-storage@vmware.com>)
* [Slack] (https://vmware.slack.com/archives/docker-volume-vsphere)
* [Telegram] (https://telegram.me/cnastorage)
* [Issues] (https://github.com/vmware/docker-volume-vsphere/issues)

## Tested on

VMware ESXi:
- 6.0
- 6.0 u1
- 6.0 u2

Docker: 1.9 and higher

Guest Operating System:
- Photon 1.0 RC
- Ubuntu 14.04 or higher (64 bit) (needs Upstart or systemctl to start and stop the plugin)

## Installation Instructions
### On ESX

Install vSphere Installation Bundle (VIB).  [Please refer to
vSphere documentation.](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)

For e.g.:
```
# Using local setup
esxcli software vib install --no-sig-check  -v /vmfs/volumes/Datastore/DirectoryName/<vib_name>.vib
```
Make sure you provide the **absolute path** to the `.vib` file or the install will fail.
### On Docker Host (VM)

The Docker volume plugin requires the docker engine to be installed as a prerequisite. This requires
Ubuntu users to configure the docker repository and pull the `docker-engine` package from there.
Ubuntu users can find instructions [here](https://docs.docker.com/engine/installation/linux/ubuntulinux/).

```
# DEB
sudo dpkg -i <name>.deb
# RPM
sudo rpm -ivh <name>.rpm
```

## Using Docker CLI

```
$ docker volume create --driver=vmdk --name=MyVolume -o size=10gb
$ docker volume ls
$ docker volume inspect MyVolume
$ docker run --name=my_container -it -v MyVolume:/mnt/myvol -w /mnt/myvol busybox sh
$ docker rm my_container
$ docker volume rm MyVolume
```

## Using ESXi Admin CLI
```
$ /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls
```
