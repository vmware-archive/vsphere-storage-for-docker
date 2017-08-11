---
title: Known Issues
--- 

## Release 1.7

* Admin updating the SPBM policy name in vCenter could cause confusions/inconsistencies. [Link](https://github.com/vmware/Kubernetes/issues/156)

* Two or more PVs could show different policy names but with the same policy ID. [Link](https://github.com/vmware/Kubernetes/issues/157)

* Node status becomes NodeReady from NodeNotSchedulable after Failover. [Link](https://github.com/Kubernetes/Kubernetes/issues/45670)
 
## Release 1.6.5 

* Node status becomes NodeReady from NodeNotSchedulable after Failover. [Link]( https://github.com/Kubernetes/Kubernetes/issues/45670)
 
## Release 1.5.7

* Node status becomes NodeReady from NodeNotSchedulable after Failover. [Link](https://github.com/Kubernetes/Kubernetes/issues/45670)

## vCenter Port other than 443
* For Kubernetes 1.6 and 1.7 releases (except Release v1.7.3 and onwards, Release v1.6.8 and onwards) vCenter Port other than 443 is not supported.
 
## Kubernetes-Anywhere

* Destroying a Kubernetes cluster operation using Kubernetes-anywhere is flaky [Link](https://github.com/Kubernetes/Kubernetes-anywhere/issues/285)
