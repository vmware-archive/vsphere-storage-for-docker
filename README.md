# Docker VMDK volume plugin  (with Go vmdkops module and ESX python service)

WIP: This is work in progress and I do not expect it to be fully working yet

Build prerequisites: 
 - Linux with Docker 1.9
 - git
 
## To build: 

```Shell
git clone https://vmware/docker-vmdk-plugin.git
cd docker-vmdk-plugin
./build.sh
```
  
Note: on Photon TP2, which does not have git installed and has Docker 1.8, 
you need to  do  the following: 

```Shell
tyum install git                     # photon only
git clone https://vmware/docker-vmdk-plugin.git
cd docker-vmdk-plugin
sed -i 's/ARG WHO/ENV WHO/' Dockerfile # photon only
./build.sh  
```

Build results are in ./bin 

## To install:
 
Proper install on ESX/guest is not ready yet, but top-level Makefile 
has a "make deploy" target, which send files over ssh  to the ESX and GUEST.
You need to manually change HOST and GUEST IPs in the Makefile.

This relies on SSH being properly enabled between you build 
machine and target machines.
 
