---
title: Datastores Support
---

Docker volumes in vSphere are backed by VMDK files. These VMDKs could be backed by any storage backend. For example the backend storage could be one of NFS, SAN, vSAN. The VMDKs reside on a central location and can be reqested by any container running on any host. Ability to attach volumes from anywhere in cluster coupled with a clustering technology such as Swarm enables a highly available architecture.

## Multiple datastores Support

By default the volume is created in the datastore that hosts the VM. But there are use cases where you want to create and consume volumes from different datastores for specific performance or functionality. vDVS allows you to create a volume on any of the datastores available.

## Default datastore
By default when you just specify name of volume - it is created in the datastore that hosts the VM. 

```
$docker volume create --driver=vsphere --name=defaultDsVolume -o size=10gb

defaultDsVolume
```
We can check the volume is created in the datastore that hosts the VM.

```
$esx ls /vmfs/volumes/vsanDatastore/dockvols

defaultDsVolume.vmdk
```

## Custom datastore

To choose a datastore other than one in which VM is hosted, you have to specify name and append the datastore name with ```@``` symbol.

```
$docker volume create --driver=vsphere --name=ds1volume@datastore1 -o size=10gb

ds1volume@datastore1
``` 
We can check that volume is created in the datastore1

```
$esx ls /vmfs/volumes/datastore1/dockvols/<VMGROUP_NAME>/ds1volume

ds1volume.vmdk
```

Finally in case of multi tenancy the logic used to resolve the datastore location is described in [Tenancy](tenancy.html#default-datastore)

## Listing Volumes

When you list the volumes, you can clearly distinguish the volumes in the datastore that hosts the VM vs. the volumes in other datastores

```
$ docker volume ls
DRIVER 			VOLUME NAME
vsphere			defaultDsVolume
vsphere			ds1volume@datastore1
photon			ds2volume@datastore2
```

Datastore for a volume is decided as shown below (preference from top to bottom)

1. Using volume long/fully qualified name (with @<datastore_name>): volume is created on custom datastore
2. Using short name (not passed @<datastore_name>): volume is created on the vmgroup's default_datastore.

## Attach Volume

Sample command to run a container with vDVS volume attached:

```
root@sc-rdops-vm02-dhcp-52-237:~# docker run -it --volume-driver=vsphere -v testVolume@vsanDatastore:/vol1 --name ub2 ubuntu
```
