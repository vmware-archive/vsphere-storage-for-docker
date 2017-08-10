--- 
title: Prerequisites
---

Following is the list of prerequisites for running Kubernetes with vSphere Cloud Provider:

* We recommend Kubernetes version 1.6.5 and vSphere version 6.0.x
* We support Photon, Ubuntu, CoreOs and Centos. it has been tested on Photon 1.0 GA, Ubuntu 16.04, CoreOS 4.11.2 and Centos 7.3
* vSphere setup to deploy the Kubernetes cluster.
* vSphere shared storage. It can be any one of VSAN, sharedVMFS, sharedNFS. 
   - Shared storage makes sure all the nodes in the Kubernetes cluster have access to the storage blocks.
* vCenter user with required set of privileges.
   - To know privileges required to install Kubernetes Cluster using Kubernetes-Anywhere, please check this [link](https://github.com/Kubernetes/Kubernetes-anywhere/blob/master/phase1/vsphere/README.md#prerequisites) 
   - If you are not installing Kubernetes cluster using Kubernetes-Anywhere and just enabling vSphere Cloud Provider on Kubernetes cluster already deployed on vSphere, Please refer this [link](https://kubernetes.io/docs/getting-started-guides/vsphere/#enable-vsphere-cloud-provider) to know about privileges required for Cloud Provider.
* VMware Tools needs to be installed on the guest operating system on each Node VM. Please refer this [link](https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.html.hostclient.doc/GUID-ED3ECA21-5763-4919-8947-A819A17980FB.html) for instruction on installing VMware tools.
* Node VM name requirements
    - VM names can not begin with numbers.
    - VM names can not have capital letters, any special characters except `.` and `-`.
    - VM names can not be shorter than 3 chars and longer than 63
* The disk.EnableUUID parameter must be set to "TRUE" for each Node VM.
