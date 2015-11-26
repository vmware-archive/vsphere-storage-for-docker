# dvolplug - Docker VMDDK volume mgr / plugin

this is a docker-side code to provide Docker Volume API for VMWARE VMDKs
It communicates to the host (esx) 'dagent' servie over vSocket, and requests
vmdk create/delete/attach/detach. 

The over-vSocket API is JSON RPC, with the following commands

Create:
{
  "ops": "create",
  "name": <vmdk name>,
  "size"
  "vsanPolicy: (optional)
  "format": ext4 (optional)
  "datastoretype": local/vsan/shared(optional)  TBD
}

Mount:


Unmount:





Build: 

make  - build statically linked go code for lin64
the result is a container msterin/dvolplug, to be run as "docker run --privileged --rm -v /etc/docker/plugin msterib/dvolplug"

Note: TBD configure persistent runs as 'apt-get install' ? 
