--- 
title: Prerequisites
---

Following is the list of prerequisites for running Kubernetes on vSphere Cloud Provider:

* We recommend Kubernetes version 1.6.5 and vSphere version 6.0.x
* We support Photon, Ubuntu, CoreOs and Centos. it has been tested on Photon 1.0 GA, Ubuntu 16.04, CoreOS 4.11.2 and Centos 7.3
* vSphere setup to deploy the Kubernetes cluster.
* vSphere shared storage. It can be any one of VSAN, sharedVMFS, sharedNFS. 
   - Shared storage makes sure all the nodes in the Kubernetes cluster have access to the storage blocks.
* vCenter user with required set of privileges.
   - To know privileges required to install Kubernetes Cluster using Kubernetes-Anywhere, please check this [link](https://github.com/Kubernetes/Kubernetes-anywhere/blob/master/phase1/vsphere/README.md#prerequisites) 
   - If you are not installing Kubernetes cluster using Kubernetes-Anywhere and just enabling vSphere Cloud Provider on Kubernetes cluster already deployed on vSphere, Please refer this [link](https://Kubernetes.io/docs/getting-started-guides/vsphere/#configuring-vsphere-cloud-provider) to know about privileges required for Cloud Provider.
