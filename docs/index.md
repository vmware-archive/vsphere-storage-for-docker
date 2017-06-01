###...
# vSphere Docker Volume Service

## Introduction

Containers have changed the way applications and packaged and deployed. Not only containers are efficient from infrastructure utilization point of view, they also provide strong isolation between process on same host, are lightweight and once packaged, can run anywhere. Docker being the most commonly used container technology as of today. 

## Persistent Storage in Container World

Although it is relatively easy to run stateless Microservices using container technology, stateful applications require slightly different treatment. There are multiple factors which need to be considered when you think about handing persistent data using containers such as:

* Containers are ephemeral by nature, so the data that needs to be persisted needs to survive the restart/re-scheduling of a container.
* When containers are re-scheduled, they can die on one host and might get scheduled on a different host	. In such a case the storage should be available and mounted on new host for the container to start gracefully.
* The application should not need to worry about un-mounting the volume from one host and mounting on another host and underlying infrastructure should handle this.
* Certain applications have a strong sense of identity (For example. Kafka, Elastic etc.) and the disk used by a container with certain identity is tied to it. It is important that if a container with a certain ID gets re-scheduled for some reason then the disk only associated with that ID is re-attached on a new host.

## vSphere docker volume service
vSphere docker volume service technology enables running stateful containers backed by storage technology of choice in a vSphere environment. It is easy to install and use from end user perspective while retaining visibility and control in hands of a vSphere administrator.

Some of key features of the vDVS are:

* Enterprise class storage: You can use vDVS with proven enterprise class technologies such as VMWare vSAN, VMFS, NFS etc. 
* Security & RBAC: The vSphere admin can manage the security and access controls on the underlying hosts on a set of hosts or data centers.
* Simple to use: It is as easy to use as other docker APIs and from end user’s perspective there is no need for additional configuration etc.

This service is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/plugins_volume/). It does not need credential management or configuration management. 

<div class="panel panel-info">
  <div class="panel-heading">
    <h3 class="panel-title">Feedback</h3>
  </div>
  <div class="panel-body">
        On going work and feature requests are tracked using <a href="https://github.com/vmware/docker-volume-vsphere/issues">GitHub Issues</a>. Please feel free to file <a href="https://github.com/vmware/docker-volume-vsphere/issues">issues</a> or email <a href ="mailto:containers@vmware.com">containers@vmware.com</a>
  </div>
</div>

** Documentation Version **

The documentation here is for the latest release. The current master documentation can be found in markdown format in the [documentation folder](https://github.com/vmware/docker-volume-vsphere/tree/master/docs). For older releases, browse to [releases](https://github.com/vmware/docker-volume-vsphere/releases) select the release, click on the tag for the release and browse the docs folder.

