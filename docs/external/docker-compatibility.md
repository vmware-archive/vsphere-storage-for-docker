---
title: Docker Compatibility
---
![Image](images/docker-cert.jpeg)

VDVS is fully compatible with Docker and this section demonstrates working examples with docker's commands and concepts.

## Docker Certified Plugin
VDVS is a docker certified plugin and it integrates with Docker volume plugin framework. You can also find details on the [Docker store](https://store.docker.com/plugins/vsphere-docker-volume-service)

## Docker Volume

VDVS plugin is fully compatible with docker volume command, you can find details and working example in [Management of docker volumes](http://vmware.github.io/vsphere-storage-for-docker/documentation/docker-volume-cli.html)

## Docker-engine in Swarm mode

VDVS can be used with Docker Engine in swarm mode to build highly available and fault tolerant application with stateful data requirements. You can find an example and details on page [HA DB in docker swarm mode](http://vmware.github.io/vsphere-storage-for-docker/documentation/demo-ha-swarm.html)

## Docker Service

Docker service is used to deploy applications that can run on docker engine in swarm mode in a distributed cluster. A docker service is typically part of a larger set of applications for example you might deploy an application server and database as a service. VDVS is fully compatible with docker service command and you can see working examples and details on [Docker Service with vSphere volumes page](http://vmware.github.io/vsphere-storage-for-docker/documentation/docker-service.html)

## Docker Stack

A stack in docker terms is a collection of services. A stack can be used to define a complete application composed of multiple services. Docker stack enables defining dependencies between services, configuration parameters in one place for a complete application. VDVS is fully compatible with docker stack commands and working examples and details can be found [Docker Service with vSphere volumes page](http://vmware.github.io/vsphere-storage-for-docker/documentation/docker-stacks.html)