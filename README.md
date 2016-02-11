# Docker VMDK volume plugin 

WIP: This is work in progress and I do not expect it to be fully working yet

Docker-side service supporting Docker Volume API for VMWARE VMDKs.

The service runs on Docker VM, inside ESX, and communicates with a dedicated
service on ESX side. The code makes use of  vmdkops module  (source in ./vmdkops)
and ESX python service (source in ./vmdkops-esxsrc). 

The end results is that "docker volume create --drive vmdk" is capable
of creating VMDK disks on enclosing ESX host, and using the new volume auto
attaches and mounts the storage so it is immediately usable
 

Build prerequisites:
 - Linux with Docker 1.9
 - git

## To build:

### Using docker

```Shell
mkdir -p $(GOPATH)/src/github.com/vmware
cd $(GOPATH)/src/github.com/vmware
git clone https://github.com/vmware/docker-vmdk-plugin.git
cd docker-vmdk-plugin
make
```

If you have GO installed on your build machine and do not want to use docker 
for build, you can ran
```
make DOCKER_USE=false"
```
instead of make.  Note that in this case .vib file will NOT be build as it DOES
required docker currently; so this option is good for dealing with guest-side
plugin only.

Note: on Photon TP2, which does not have git installed and has Docker 1.8,
you need to  do  the following:

```Shell
tyum install -y git      # photon only
tyum install -y make     # photon only
git clone https://github.com/vmware/docker-vmdk-plugin.git
cd docker-vmdk-plugin
make
```
Build results are in ./bin


## To install:

Proper install on ESX/guest is not ready yet, but top-level Makefile
has a "make deploy" target, which send files over ssh  to the ESX and GUEST.

You need to manually change HOST and GUEST IPs in the Makefile to point to
to IPs in your environment. The copy will rely on SSH being properly enabled
on target machines and between them and your build machine.

```
make deploy
```
will copy python code, vmci shared lib code and a mkfs.ext4 binary
to ESX (/usr/lib/vmware/vmdkops/bin) and will copy dvolplug binary to guest
Linux VM

If you build machine does not have 'make' (e.g. Photon OS), you can use
```
docker run --rm -v $PWD:/work -w /work docker-vmdk-plugin  make deploy
```

Note: when fully  implemented, will will have VIB install via excli, RPM
install on VC (and pushing VIBs automatically), and proper container to run
on Linux guest

## To run:

on ESX:
```
python /usr/lib/vmware/vmdkops/bin/vmci_srv.py
```

on Linux guest:
```
sudo /usr/local/bin/dvolplug
```

Then you can run "docker volume" commands against docker on you Linux guest,
something like that:
```Shell
docker volume create --driver=vmdk --name=MyVolume -o size=10gb
docker volume ls
docker volume inspect MyVolume
docker run  --name=my_container -it -v MyVolume:/mnt/myvol -w /mnt/myvol busybox sh
docker volume rm MyVolume # SHOULD FAIL - still used by container
docker rm my_container
docker volume rm MyVolume # Should pass
```

Note: when fully implemented, all will run automatically as service and this
step won't be needed
