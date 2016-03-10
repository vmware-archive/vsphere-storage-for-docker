[![Build Status](https://ci.vmware.run/api/badges/vmware/docker-vmdk-plugin/status.svg)](https://ci.vmware.run/vmware/docker-vmdk-plugin)

# Quicksteps to get going

Here the basic commands you need to run to get going with the docker vmdk plugin.

```
./build.sh # Use a docker image to run make.
# Output is in the bin folder.

make deploy-esx ESX=root@10.20.105.54
make deploy-vm  VM=root@10.20.105.121

# To run on ESX

python /usr/lib/vmware/vmdkops/bin/vmci_srv.py

# To run on Linux guest

sudo apt-get install libc6-i386

sudo /usr/local/bin/docker-vmdk-plugin

# Docker commands to use plugin

docker volume create --driver=vmdk --name=MyVolume -o size=10gb
docker volume ls
docker volume inspect MyVolume
docker run --name=my_container -it -v MyVolume:/mnt/myvol -w /mnt/myvol busybox sh
docker volume rm MyVolume # SHOULD FAIL - still used by container
docker rm my_container
docker volume rm MyVolume # Should pass
```

# Docker VMDK volume plugin (WIP)

Native ESXi VMDK support for Docker Data Volumes.

When Docker runs in a VM under ESXi hypervisor, we allow Docker user to 
create and use VMDK-based data volumes. Example: 

```Shell
docker volume create --driver=vmdk --name=MyStorage -o size=10gb
docker run --rm -it -v MyStorage:/mnt/data busybox sh
```

This will create a MyStorage.vmdk on the same datastore where Docker VM is 
located. This vmdk will be attached to the Docker VM on "docker run" and 
the containers can use this storage for data. 

This repo contains guest code and ESXi code. 

The docker-vmdk-plugin service runs in docker VM and talks to Docker Volume
Plugin API via Unix Sockets. It then relays requests via VMWare vSockets 
host-guest communication to a edicated service on ESXi. 

The docker plugin code makes use of  vmdkops module  (found  in ./vmdkops)
and ESX python service (found in ./vmdkops-esxsrc). 

The end results is that "docker volume create --drive vmdk" is capable
of creating VMDK disks on enclosing ESX host, and using the new volume auto
attaches and mounts the storage so it is immediately usable

## To build:

Build prerequisites:
 - Linux with Docker (1.8+ is supported)
 - git
 - make
 
Build results are in ./bin.
 
### Using docker (Recommended)

Use it when you do not have GO, vibauthor and 32 bit C libraries on your machine, 
or do not plan to impact your GO projects. 

```Shell
git clone https://github.com/vmware/docker-vmdk-plugin.git
cd docker-vmdk-plugin
./build.sh
./build.sh test
./build.sh clean
```

There are also the following targets (can use used with build.sh):
```
make clean       # removes binaries build by 'make'
make test        # runs whatever unit tests we have
make deploy      # deploys to your test box (see below)
make cleanremote # uninstalls from test boxes)
make testremote  # runs sanity check docker commands for volumes
```

### Building on Photon TP2 Minimal

Photon TP2 Minimal does not have git or make installed, and does not 
not start docker by default, so you need to do this before running make: 

```Shell
 # Photon TP2 Minimal only:
tyum install -y git
tyum install -y make
systemctl start docker
```
and then the git/cd/make sequence. 

### Building without Docker

This build requires
- GO to be installed
- vibauthor
- 32 bit libc headers (as it is needed for ESX-side vSocket shim compilation.)

With these prerequisites, you can do the following to build: 

```
mkdir -p $(GOPATH)/src/github.com/vmware
cd $(GOPATH)/src/github.com/vmware
git clone https://github.com/vmware/docker-vmdk-plugin.git
cd docker-vmdk-plugin
make 
```

## To install:

```
make deploy-esx ESX=root@10.20.105.54
make deploy-vm VM=root@10.20.105.121
```
This will copy the VIB file to ESXi and install in so  /usr/lib/vmware/vmdkops/bin
has all neccessary file, and will copy docker-vmdk-plugin binary to guest
Linux VM /usr/local/bin. 

Note: when fully  implemented, deploy will have the VIB install actually starting
the service properly, RPM install on VC for pushing VIBs automatically, 
and proper container to run service on Linux guest.

## To Run:

Check quicksteps above.
One dependency that needs to be installed on the guest side is the 32 bit libc.

# WIP
To be continued...
