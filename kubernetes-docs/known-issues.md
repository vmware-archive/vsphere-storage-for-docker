---
title: Known Issues
--- 

## Release 1.7

* Admin updating the SPBM policy name in vCenter could cause confusions/inconsistencies. [Link](https://github.com/vmware/kubernetes/issues/156)

* Two or more PVs could show different policy names but with the same policy ID. [Link](https://github.com/vmware/kubernetes/issues/157)

* Node status becomes NodeReady from NodeNotSchedulable after Failover. [Link](https://github.com/kubernetes/kubernetes/issues/45670)
 
## Release 1.6.5 

* Node status becomes NodeReady from NodeNotSchedulable after Failover. [Link]( https://github.com/kubernetes/kubernetes/issues/45670)
 
## Release 1.5.7

* Node status becomes NodeReady from NodeNotSchedulable after Failover. [Link](https://github.com/kubernetes/kubernetes/issues/45670)
 
## Kubernetes-Anywhere

* Destroying a Kubernetes cluster operation using kubernetes-anywhere is flaky [Link](https://github.com/kubernetes/kubernetes-anywhere/issues/285)
