---
title: Kubernetes Anywhere
---

There are several deployment mechanisms available to deploy Kubernetes. Kubernetes-Anywhere is one of them. Please refer this [link](https://github.com/Kubernetes/Kubernetes-anywhere) 
 
We have simplified deployment with creating pre-build Kubernetes-Anywhere [docker image](https://hub.docker.com/r/cnastorage/kubernetes-anywhere/)
 
User can just pull this image and run a container with this image to get into deployment wizard. User can fill in the questionnaires, Kubernetes-Anywhere will create a terraform deployment script to install Kubernetes cluster on vSphere.
After the cluster is deployed, cluster config file is available at /opt/Kubernetes-anywhere/phase1/vsphere/.tmp/kubeconfig.json
 
Make sure to copy this file, before stopping the deployment container. This file is used to access Kubernetes cluster using kubectl.
 
Please refer installation steps mentioned in the getting started guide for deploying Kubernetes using [Kubernete-Anywhere](https://github.com/Kubernetes/Kubernetes-anywhere/blob/master/phase1/vsphere/README.md#prerequisites) 
 
