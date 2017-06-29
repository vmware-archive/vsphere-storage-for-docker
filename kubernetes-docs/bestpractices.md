---
title: Best Practices
---

This section describes the vSphere specific configurations:
**vSphere HA**
* vSphere Cloud Provider supports vSphere HA. To ensure high availability of node VMs, it is recommended to enable HA on the cluster.[Details](https://pubs.vmware.com/vsphere-65/index.jsp#com.vmware.vsphere.avail.doc/GUID-4BC60283-B638-472F-B1D2-1E4E57EAD213.html) 

**Use Resource Pool**
* It is recommended to place Kubernetes node VMs in the resource pool to ensure guaranteed performance. [Details](https://pubs.vmware.com/vsphere-65/index.jsp#com.vmware.vsphere.resmgmt.doc/GUID-0F6C6709-A5DA-4D38-BE08-6CB1002DD13D.html) 
