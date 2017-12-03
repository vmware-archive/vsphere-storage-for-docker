---
title: vFile volume plugin for Docker

---
## Overview
Depending on the underlying block storage system, it might not be possible to access the same
persistent volume across different hosts/nodes simultaneously.
For example, currently users cannot mount the same persistent volume which is created through
vSphere Docker Volume Service (vDVS) on containers running on two different hosts at the same time.

This can be solved through distributed file systems, such as NFS, Ceph, Gluster, etc.
However, setting up and maintaining enterprise storage offerings for Cloud Native use cases is not a trivial work.
Furthermore, users can face more challenges in order to achieve high availability, scalability, and load balancing.

__vFile volume plugin for Docker__ provides simultaneous persistent volume access between hosts in the
same Docker Swarm cluster for the base volume plugin service such as vDVS, with zero configuration effort,
along with high availability, scalability, and load balancing support.

## Detailed documentation
Detailed documentation can be found on our [GitHub Documentation Page](http://vmware.github.io/docker-volume-vsphere/documentation/).

## Prerequisites
* Docker version: 17.06.0 or newer
* Base docker volume plugin: [vSphere Docker Volume Service](https://github.com/vmware/docker-volume-vsphere)
* All hosts running in [Swarm mode](https://docs.docker.com/engine/swarm/swarm-tutorial/)
* All docker swarm managers should open [official etcd ports](http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt) 2379 and 2380, or the user-defined ETCD ports (please find more details about user-defined ETCD ports in the Installation section)

## Installation
The recommended way to install vFile plugin is from docker cli:

```
~$ docker plugin install --grant-all-permissions --alias vfile vmware/vfile:latest VFILE_TIMEOUT_IN_SECOND=90
```

Note: please make sure the base volume plugin is already installed!

Internally, vFile creates and uses etcd cluster to store metadata for volumes.
By default, the etcd cluster listens on port 2379 for client communication and port 2380 for peer communication.
If you have other etcd cluster which already listens on those default ports, you need to specify different ports to avoid conflict when installing the vFile plugin.
Please see the following example:

```
~$ docker plugin install --grant-all-permissions --alias vfile vmware/vfile:latest VFILE_TIMEOUT_IN_SECOND=90 VFILE_ETCD_CLIENT_PORT=4001 VFILE_ETCD_PEER_PORT=4002
```

* The `VFILE_TIMEOUT_IN_SECOND` setting is strongly recommended before [Issue #1954](https://github.com/vmware/docker-volume-vsphere/issues/1954) is resolved.

## Remove and Reinstallation
The recommended order to remove and reinstallation is:

* remove and reinstall vFile plugin on all worker nodes
* remove and reinstall vFile plugin on all manager nodes which are not swarm leader
* remove and reinstall vFile plugin on manger node which is swarm leader

Run the following command to remove and reinstall vFile plugin from docker cli:

```
~$ docker plugin rm vfile:latest vfile:latest
~$ docker plugin install --grant-all-permissions --alias vfile vmware/vfile:latest VFILE_TIMEOUT_IN_SECOND=90
```

Note: Please make sure no volume exists when trying to remove and reinstall the vFile plugin.

You will get the following error when trying to remove vFile plugin if a volume still exists.

```
~$ docker plugin rm vfile:latest
~$ Error response from daemon: plugin vfile:latest is in use
```

You can use:

```
~$ docker plugin rm -f vfile:latest
```

to force remove the vFile plugin and reinstall the vFile plugin, but after that, that volume is not usable.

## Usage examples

### Steps for create/use/delete a vFile volume

#### Creating a persistent volume from vFile plugin

```
~$ docker volume create --driver=vfile --name=SharedVol -o size=10gb
~$ docker volume ls
~$ docker volume inspect SharedVol
```

Options for creation will be the same for the base volume plugin.
Please refer to the base volume plugin for more options.
Note: vFile volume plugin doesn't support filesystem type options.
Note: The valid volume name can only be ```[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]```.

#### Mounting this volume to a container running on the first host

```
# ssh to node1
~$ docker run --rm -it -v SharedVol:/mnt/myvol --name busybox-on-node1 busybox
/ # cd /mnt/myvol
# read/write data into mounted vFile volume
```

#### Mounting this volume to a container running on the second host

```
# ssh to node2
~$ docker run --rm -it -v SharedVol:/mnt/myvol --name busybox-on-node2 busybox
/ # cd /mnt/myvol
# read/write data from mounted vFile volume
```

#### Stopping the two containers on each host

```
~$ docker stop busybox-on-node1
~$ docker stop busybox-on-node2
```

#### Removing the vFile volume

```
~$ docker volume rm SharedVol
```

## Configuration
### Options for vFile plugin
Users can choose the base volume plugin for vFile plugin, by setting configuration during install process.
<!---
* Through CLI flag can only be done through non-managed plugin.
--->

* Default config file location: `/etc/vfile.conf`.
* Default base volume plugin: vSphere Docker Volume Service
* Sample config file:

```
{
        "InternalDriver": "vsphere"
}
```

The user can override the default configuration by providing a different configuration file,
via the `--config` option, specifying the full path of the file.

### Options for logging
* Default log location: `/var/log/vfile.log`.
* Logs retention, size for rotation and log location can be set in the config file too:

```
 {
	"MaxLogAgeDays": 28,
	"MaxLogFiles": 10,
	"MaxLogSizeMb": 10,
	"LogPath": "/var/log/vfile.log"
}
```

## How to troubleshoot issues

### Which log is needed to debug issues? Where is the log located?
Log file `vfile.log` is needed to debug. This file is located at `/var/log` directory of the docker host where the vFile plugin is installed.

### What is the default log level? How to change the log level?
Default log level is `INFO`. If needed, log level can be changed to `DEBUG` using the following command during vFile plugin install:

```
~$ docker plugin install --grant-all-permissions --alias vfile vmware/vfile:latest VFILE_TIMEOUT_IN_SECOND=90 VDVS_LOG_LEVEL=debug
```

### What should I do if the Docker Swarm cluster is in unhealthy state?
vFile plugin can only work properly when the Docker Swarm cluster is healthy. If Swarm cluster is unhealthy, you may need to destroy and recreate the Docker Swarm cluster.

## Q&A

### How to install and use the driver?
Please follow the instructions at [installation-on-esxi](http://vmware.github.io/docker-volume-vsphere/documentation/install.html#installation-on-esxi).

### Can I use another base volume plugin other than vDVS?
Currently vFile volume service is only developed and tested with vDVS as the base volume plugin.

### Can I use default driver for local volumes as a base volume plugin?
No, this is not supported by vFile plugin.

### I got "Operation now in progress" error when mounting a vFile volume to a container.
Please make sure the routing mesh of Docker Swarm cluster is working properly.
Use the following way to verify:

```
~$ docker service create --replicas 1 -p 8080:80 --name web nginx
```

Then on each host, make sure there is valid return of `curl 127.0.0.1:8080`.
`Connection refused` error means the routing mesh of this Docker Swarm cluster is broken and needs to be fixed.

### Mounting volume failed and I saw "name must be valid as a DNS name component" error in the log when mounting a vFile volume to a container.
When you see something like the following in the log

```
2017-08-24 11:57:16.436786459 -0700 PDT [WARNING] Failed to create file server for volume space vol7. Reason: Error response from daemon: {"message":"rpc error: code = 3 desc = name must be valid as a DNS name component"}
```

Please make sure the volume you used is a valid volume name. A valid volume name consists of ```[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]```.

### I got " VolumeDriver.Mount: Failed to blocking wait for Mounted state. Error: Timeout reached; BlockingWait is not complete.." when mounting a volume.
We see this issue only on platform where the space is low. When available disk space is low, Docker Swarm service may take longer to start a service. In general, it's better free up some disk space. You can also try to increase the service start timeout value, controlled by ```VFILE_TIMEOUT_IN_SECOND``` env variable:

```
~$ docker plugin install --grant-all-permissions --alias vfile vmware/vfile:latest VFILE_TIMEOUT_IN_SECOND=90
```

This will increase timeout to 90 seconds, from default of 30 seconds.

### Volume mount failed, retry mount with the same volume still failed.
Check the "Volume Status" field of ``` docker volume inspect ```, if the "Volume Status" field shows "Error", which means volume is in error status, user should remove the volume manually.

```
~$ docker volume inspect vol1
[
    {
        "Driver": "vfile:latest",
        "Labels": {},
        "Mountpoint": "/mnt/vfile/vol1/",
        "Name": "vol1",
        "Options": {},
        "Scope": "global",
        "Status": {
            "Clients": null,
            "File server Port": 0,
            "Global Refcount": 0,
            "Service name": "",
            "Volume Status": "Error"
        }
    }
]
```


### I got "docker volume ls" operation very slow and "docker volume rm/create" a vFile volume hang forever
Please check the log at `/var/log/vfile.log` and look up if there are error message about swarm status as follow:

`The swarm does not have a leader. It's possible that too few managers are online. Make sure more than half of the managers are online.`

This error message indicates the swarm cluster is not in a healthy status.

Please follow the following [recommendations for the Swarm manager nodes setup](https://docs.docker.com/engine/swarm/how-swarm-mode-works/nodes/#manager-nodes):

    1. Run the swarm cluster with a single manager only for testing purpose.
    2. An `N` manager cluster will tolerate the loss of at most `(N-1)/2` managers.
    3. Docker recommends a maximum of seven manager nodes for a swarm.


### Does vFile plugin support multi-tenancy feature?
No. vFile plugin does not support multi-tenency, which is currently an experimental feature for vDVS.
Assume we have created a vmgroup ```vmgroup1``` and add VM "node1" to ```vmgroup1```. VM "node2" has not been added to any vmgroup and is considered belong to ```_DEFAULT``` vmgroup.
Then create a volume "vol1" from "node2" and a volume "vol2" from "node1" using vFile plugin.

On "node1", we are able to see both "vol1" and "vol2" which are created by vFile plugin.

```
~$ docker volume ls
DRIVER              VOLUME NAME
vsphere:latest      _vF_vol2@vsanDatastore
vfile:latest        vol1
vfile:latest        vol2

```

On "node2", we are also able to see both "vol1" and "vol2".

```
~$ docker volume ls
DRIVER              VOLUME NAME
vsphere:latest      _vF_vol1@vsanDatastore
vfile:latest        vol1
vfile:latest        vol2

```

### Why should I choose vFile instead of just enabling multi-writer flag on vSphere file systems?
vSAN multi-writer mode permits multiple VMs to access the same VMDKs in read-write mode to enable the use of in-guest
shared-storage clustering solutions such as Oracle RAC.
However, the guests must be capable of safely arbitrating and coordinating multiple systems accessing the same storage.
If your application is not designed for maintaining consistency in the writes performed to the shared disk, enabling
multi-writer mode could result in data corruption.
More information can be found [here](https://kb.vmware.com/s/article/1034165) and [here](https://kb.vmware.com/s/article/2121181).

### I got "no space left on device" error when installing vFile plugin but I have enough space on my node
This error can happen when your docker version is below required version.
Please check /var/log/vfile.log to locate if there are warning messages about your current docker server version.
