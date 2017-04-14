---
title: Prerequisites
---

This section enumerates the prerequisites like environments, packages required to use vSphere Docker Volume plugin.

## VMWare Environment

vSphere Docker Volume Service can be used with vSphere environment or Photon Platform but in all section it is implicit that we are refering to vSphere environment. There is separate section for Photon platform. 

## Docker
On all the machines where you want to operate the vDVS, you will need docker-engine installed. You can get the installable specific to your OS form Docker website.

## Docker volume plugin
Docker engine can be extended by using the plugin framework. Docker provides a plugin API and standard interfaces which can be used to extend docker engine’s core functionality. Docker volume plugins specifically are targeted at storage related integrations and can be used to work with underlying storage technologies. You can read more about the [Docker’s plugin system](https://docs.docker.com/engine/extend/).

VMWare uses the volume plugin mechanism to enable vSphere docker volume service (vDVS) for vSphere environments. 

## vSphere Docker Volume Service components
vSphere Docker Volume Service comprises of Docker plugin and vSphere Installation Bundle which bridges the Docker and vSphere ecosystems. 

### Managed Plugin
vSphere Docker Volume Service is integrated with Docker Volume Plugin Framework and doesn't require credential management or configuration management. 
The managed plugin on [Docker store](https://store.docker.com/plugins/vsphere-docker-volume-service?tab=description)

### vSphere Installation Bundle (VIB)
The second component of the volume service is the VIB which needs to be installed on ESXi server.
The VIB for ESXi server [available here](https://bintray.com/vmware/vDVS)

## Storage Technologies: Introduction

We might refer to the terms related to storage technologies later in the documentation, but this is a good place to understand various technologies and know more about them if you want to.

VMware vSAN is a enterprise class storage technology from VMWare which makes storage easy in a hyper converged infrastructure.  You can learn more about [VMWare vSAN here](http://www.vmware.com/in/products/virtual-san.html)

[VMWare VMFS](https://pubs.vmware.com/vsphere-50/index.jsp?topic=%2Fcom.vmware.vsphere.storage.doc_50%2FGUID-5EE84941-366D-4D37-8B7B-767D08928888.html) (Virtual Machine File System) is a highly specialized and optimized file system format for storing virtual machine files on a storage system.

NFS is a storage technology protocol for storing the files over a network as if you were storing locally. The protocol was originally developed at Sun Microsystems and multiple implementations do exist in the market today. You can understand a bit more about the NFS protocol and various implementations at [this Wikipedia link](https://en.wikipedia.org/wiki/Network_File_System)
