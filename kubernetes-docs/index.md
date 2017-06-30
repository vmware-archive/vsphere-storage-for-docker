---
title: Kubernetes - vSphere Cloud Provider
---


## Introduction 

Containers have changed the way applications are packaged and deployed. Not only containers are efficient from infrastructure utilization point of view, they also provide strong isolation between process on same host. They are lightweight and once packaged can run anywhere. Docker is the most commonly used container runtime technology and this user guide outlines how vSphere is compatible with Docker ecosystem. 

## Persistent Storage in Container World

Although it is relatively easy to run stateless Microservices using container technology, stateful applications require slightly different treatment. There are multiple factors which need to be considered when you think about handling persistent data using containers such as:

* Containers are ephemeral by nature, so the data that needs to be persisted has to survive through the restart/re-scheduling of a container.
* When containers are re-scheduled, they can die on one host and might get scheduled on a different host. In such a case the storage should also be shifted and made available on new host for the container to start gracefully.
* The application should not have to worry about the volume/data and underlying infrastructure should handle the complexity of unmounting and mounting etc.
* Certain applications have a strong sense of identity (For example. Kafka, Elastic etc.) and the disk used by a container with certain identity is tied to it. It is important that if a container with a certain ID gets re-scheduled for some reason then the disk only associated with that ID is re-attached on a new host.

## Kubernetes Storage Primitives

Kubernetes provides abstractions ensure that the storage details are separated from allocation and usage of storage.
Kubernetes has various API resources to provision, categorize and consume the storage. 

**API Resources**:

* Volumes
* Persistent Volumes
* Persistent Volumes Claims
* Storage Class
* Statefulsets

Please refer [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/volumes/) for details.

