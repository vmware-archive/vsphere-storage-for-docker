---
title: FAQs
---

## Is vSphere Cloud Provider ready for production ?
vSphere Cloud Provider is still in beta.
 
## What is the biggest Kubernetes cluster it has been tested for ?
It has been tested on eight node cluster so far.
 
## Where can I find the list of vSAN, VMFS and NFS features supported by vSphere Cloud Provider ?
Please refer to this [section](/spbm.html). Please report in case you find any features are missing.
 
## How is running containers on vSphere Integrated Containers different from running them on Kubernetes on vSphere ? 
VIC is infrastructure platform to run containerized workloads alongside traditional applications whereas vSphere Cloud provider provides an interface to run and take advantage of vSphere storage for workloads running on Kubernetes
 
## Can I run it on a single node cluster on my laptop ?
Yes as long as laptop supports nested virtualization you can try it on your laptop.
 
## Which Kubernetes distribution is supported ?
vSphere Cloud Provider is available in vanilla Kubernetes and all distributions using Kubernetes v1.5 and above should support it. Please refer this [section.](/docker-volume-vsphere/kubernetes/prereq.html)  
 
## Can we deploy multiple Kubernetes Cluster on one vCenter? 
Yes. Please refer this [section.](/docker-volume-vsphere/kubernetes/existing.html)  
 
 
## Can Kubernetes Cluster access storage from another vCenter? 
Yes. Please refer this [section.](/docker-volume-vsphere/kubernetes/existing.html)
 
## Which Operating System are supported ? 
We support Photon, Ubuntu, Core OS, please check this section for [details](/docker-volume-vsphere/kubernetes/prereq.html)
 
## How Kubernetes volumes can be made resilient to failures on vSAN datastore?
Please check the HA section for [details.](/docker-volume-vsphere/kubernetes/ha.html)
 
## Can I enable SDRS on VMs hosting kubernetes cluster?
No.
 
## Can we have a setting to ensure all dynamic PVs with have default policy Retain (instead of delete)? Or can we request the desired policy from the moment we request the PV via the PVC?
If the volume was dynamically provisioned, then the default reclaim policy is set to “delete”.  This means that, by default, when the PVC is deleted, the underlying PV and storage asset will also be deleted.
If you want to retain the data stored on the volume, then you must change the reclaim policy from “delete” to “retain” after the PV is provisioned. You cannot directly set to retain PV from PVC request for dynamic volumes. [Details](https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/)
 
## How do we resize the existing dynamic volumes? If we update the PVC with the new desired size, is it enough?
Support for resizing existing dynamic volume is not yet there.
Proposal is out for [review.](https://github.com/gnufied/community/blob/91b41028182a5291b4eccbf88f8065f66b2b7eed/contributors/design-proposals/grow-volume-size.md)
 
## Can we create ReadWriteMany volumes with VSphere storage, with pods on different machines?
ReadWriteMany is not supported with Pods on different machine. This is supported on the collocated pods on the same node.
 
## Is it mandatory to have all the machines in the same datastore ? If so, it's a very strong limitation for us.
It is not mandatory to keep all node VMs on the same datastore.  But make sure node VM has access to volumes datastores.
 
 
## Can we have the disk uuid enabled by default, so we don't need to do it machine by machine? What would be the risks of having it by default. 
You can enable disk UUID by default, while creating VM. Just add this parameter at
Customize hardware -> VM Options -> Configuration Parameters -> Edit Configuration
 
## Is it mandatory to have all the machines in the same directory?
Current code base requires Master and Node VMs to be present under one VM folder. Each Kubernetes Cluster deployed in vSphere, should be placed in their respective VM folder, else under root folder.
