--- 
title: Prerequisites
---

Following is the list of prerequisities for running Kubernetes on vSphere Cloud Provider:

* We recommend Kubernetes version 1.5.x and vSphere version 6.0.x
* We support Photon, Ubuntu, CoreOs and Centos. it has been tested on Photon 1.0 GA, Ubuntu 16.04, CoreOS 4.11.2 and Centos 7.3
* vSphere setup to deploy the Kubernetes cluster.
* vSphere shared storage. It can be any one of VSAN, sharedVMFS, sharedNFS. 
   - Shared storage makes sure all the nodes in the Kubernetes cluster have access to the storage blocks.
* vCenter user with required set of privileges.
   - To know privileges required to install Kubernetes Cluster using Kubernetes-Anywhere, refer https://github.com/kubernetes/kubernetes-anywhere/blob/master/phase1/vsphere/README.md#prerequisites 
   - If you are not installing kubernetes cluster using Kubernetes-Anywhere and just enabling vSphere Cloud Provider on kubernetes cluster already deployed on vSphere, refer https://kubernetes.io/docs/getting-started-guides/vsphere/#configuring-vsphere-cloud-provider to know about privileges required for Cloud Provider.
