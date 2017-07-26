---
title: High Availability of Kubernetes Cluster
---

Kubernetes ensures that all the the Pods are restarted in case the node goes down. The persistent storage API objects ensure that same PVs are mounted back to the new Pods on restart or if they are recreated. 

But what happens if the node/host is the VM and the physical host fails? vSphere HA  leverages multiple ESXi hosts configured as a cluster to provide rapid recovery from outages and cost-effective high availability for applications running in virtual machines. vSphere HA provides a base level of protection for your virtual machines by restarting virtual machines in the event of a host failure 
 
Applications running on Kubernetes on vSphere can take advantage of vSphere Availability ensuring resilient and highly available applications.
 
**Node VM Failure:**
Node VM failure will cause Kubernetes to recreate a new pod to run the containers. vSphere Cloud Provider will mount the disk to a live node and unmount disk from the dead node automatically. The validation description is as follows:
						
* Shutdown one Kubernetes node VM. This will cause Kubernetes to remove the node VM from the Kubernetes cluster.
* The Kubernetes cluster will recreate the pod on an idle node in the original cluster after the simulated node failure. Kubernetes vSphere Cloud Provider will:

  - Mount the disks from the shutdown node VM to the idle node.
  - Unmount the disks from the powered off node VM.
									
* Fix the issue of the node VM (if any) and power it on. Kubernetes will add the node back to the original cluster and it will be available for new pod creation. 
 
**Physical Host Failure:**
Powering off one of the ESXi hosts will cause the vSphere Availability to restart the node on one of the running ESXi servers. The node in Kubernetes cluster will temporarily change to UNKNOWN. After less than two minutes, the node will be available in the cluster. No pod recreation is required. 
 



<table class="table table-striped table-hover ">
  <thead>
    <tr>
      <th>Failure Components</th>
      <th>Recovery Time</th>
      <th>Result and Behavior</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Kubernetes node shutdown or corruption</td>
      <td> < 1 min</td>
      <td>The pod is recreated, and Kubernetes vSphere Cloud Provider automatically unmounts/mounts the disk to standby node. The standby node changes to working node</td>
    </tr>
    <tr>
      <td>ESXi host failure or powered off</td>
      <td> < 2 mins</td>
      <td>vSphere Availability restarted the Kubernetes node on another ESXi host; no node recreation required.</td>
    </tr>
</tbody>
</table>
Please note the recovery time is dependent upon the hardware. For additional details please refer this [blog.](https://blogs.vmware.com/virtualblocks/2017/06/13/recipe-vmware-vsan-persistent-storage-mongodb-containers/)
 
 
 
