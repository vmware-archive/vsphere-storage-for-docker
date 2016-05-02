[![Build Status](https://ci.vmware.run/api/badges/vmware/docker-volume-vsphere/status.svg)](https://ci.vmware.run/vmware/docker-volume-vsphere)

Docker Volume Driver for vSphere
================================

This repo hosts the Docker Volume Driver for vSphere. The plugin integrated with [docker volume
plugin framework](https://docs.docker.com/engine/extend/plugins_volume/) will help customers address persistence storage requirements of docker containers backed by vSphere storage (vSAN, VMFS, NFS etc). 

To read more about code development and testing read
[CONTRIBUTING.md](https://github.com/vmware/docker-volume-vsphere/blob/master/CONTRIBUTING.md)

## Tested on

VMware ESXi:
- 6.0
- 6.0 u1
- 6.0 u2

Docker: 1.9 and higher

Guest Operating System:
- Photon 1.0 RC
- Ubuntu 14.04 64 bit (needs Upstart or systemctl)

## Installation Instructions
### On ESX

Install vSphere Installation Bundle (VIB).  [Please refer to
vSphere documentation.](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html#GUID-29491174-238E-4708-A78F-8FE95156D6A3)

For e.g.:
```
# Using local setup
esxcli software vib install --no-sig-check  -v /vmfs/volumes/Datastore/DirectoryName/<vib_name>.vib
```
### On Docker Host (VM)

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

# Demo To be continued...

# Contact 

Please let us know what you think! Contact us at [cna-storage@vmware.com](cna-storage <cna-storage@vmware.com>)
