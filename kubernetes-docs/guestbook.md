---
title: Running Stateful application - Guestbook App
--- 
 
Guestbook is PHP application with Redis as backend. In this section we will demonstrate how to use  Kubernetes deployed on vSphere  to run Guestbook Application with persistent storage. At the end of this demo, you will have a sample guestbook app running inside Kubernetes where the data is resident inside VMDKs managed by vSphere.

The data in the VMDK is independent of the lifecycle of the pods and persists even if pods are deleted.					
						
**Storage Setup** 
							
The backing VMDK files are needed when dynamic provisioning is not used. The VMDK files need to exist before creating the service in k8s that uses them.
							
Login to ESX (if more then one ESX is used make sure to have a common/shared datastore) and create the volumes 
		 	 	 		
```					
# VMFS
cd /vmfs/volumes/datastore1/
# VSAN
cd /vmfs/volumes/vsanDatastore/
						
# VMFS
mkdir kubevols # Not needed but good hygiene
# VSAN
/usr/lib/vmware/osfs/bin/osfs-mkdir kubevols # Needed
						
cd kubevols
						
vmkfstools -c 2G redis-slave.vmdk 
vmkfstools -c 2G redis-master.vmdk 
```				
					
						
## Guestbook with statically provisioning

In this example we will provision PVs and Guestbook pods will claim these PVs
 

**Create a record for the persistent Single command [TODO]**

Download all files in the PVC PV Demo from [here](https://github.com/vmware/kubernetes/tree/kube-examples/kube-examples/guestbook)
						
**Create all the records in one**

```
$ kubectl create -f guestbook-pvc # Folder containing all the yamls for the pvc demo 
```
						
**Individual Steps**
						
Download the yaml files
					
				
						
Create the persistent volume records

```						
$ kubectl create -f redis-master-pv.yaml create -f redis-slave-pv.yaml
```						
					
Check the persistent volume records

```			
$ kubectl get pv
```

						
Create the PVC

```
$ kubectl create -f redis-slave-claim.yaml 
$ kubectl create -f redis-master-claim.yaml
```

Check that the volumes are bound to the claims 

``` 
$ kubectl get pv
 
NAME             CAPACITY ACCESSMODES RECLAMIPOLICY STATUS CLAIM REASON AGE
redis-master-pv 2Gi           RWO           Retain           Bound     default/redis-master-claim     34s 
redis-slave-pv 2Gi           RWO              Retain               Bound     default/redis-slave-claim     34s 	 		
```

Create the guestbook

```
kubectl create -f guestbook-all-in-one.yaml
``` 
 
## Guestbook Application with Dynamic Provisioning
					
						
Download the yaml files for storage class based guestbook 
 
Create the storage class

```						
$ kubectl create -f redis-sc.yaml
```

Create the claims that use the storage class

```					
$kubectl create -f redis-master-claim.yaml create -f redis-slave-claim.yaml
```

This should trigger storage volumes getting created on the datastore configured on the vsphere cloud provider 					
 
```						
[root@promc-2n-dhcp41:/vmfs/volumes/57f5768f-856aa050-fde c-e0db55248054/kubevols] ls *-pvc-* kubernetes-dynamic-pvc-91c9cd60-9a7c-11e6-a431-00505690d6 4d-flat.vmdk kubernetes-dynamic-pvc-a6abf43a-9a7c-11e6-a 431-00505690d64d-flat.vmdk kubernetes-dynamic-pvc-91c9cd60-9a7c-11e6-a431-00505690d6 4d.vmdk kubernetes-dynamic-pvc-a6abf43a-9a7c-11e6-a 431-00505690d64d.vmdk 
```

All the dynamic provisioned volumes will be created in the kubevols directory inside the datastore.				
		 	 	 		
Create the guestbook app consuming the claims 

```
$ kubectl create -f guestbook-all-in-one.yaml 
```

Verify Guestbook Application
The following steps verify that the Pods, storage and the application are running correctly
 
Wait till the pods are ready

``` 
$ kubectl get pods
NAME                                 READY STATUS RESTARTS AGE
frontend-88237173-77e6n 1/1         Running  0                   43m
redis-master-1145698066-2hbbk 1/1         Running  0                   43m
redis-slave-5845680045-3ffdg 1/1         Running  0                   43m		
``` 
 
Get the port on which the front end is accessible (kube-up does not configure an external load balancer or ingress and guestbook uses NodePort mapping)

```					
$ kubectl describe service frontend | grep NodePort
Type: NodePort
NodePort: <unset> 31531/TCP 
```		
					
Check the nodes on which the frontend is running 
 
		 	 	 		
						
Get the IP address of one of the nodes (This can also be obtained from vCenter)
Also possible from vCenter or the console output kube-up
		
```
$ kubectl describe node 
$ kubernetes-minion-1|grep -i Address Addresses: 10.20.105.59 
```					
		 	 	 		
				
Combine the IP and Port and head to the URL (Ex:
http://10.20.105.59:31531 ). Try out the app and leave a few messages.
To check the attach status of VMDKs on ESX head to vCenter (Settings page or Recent Tasks window)
					
			
		
## Lifecycle of volume  independent of Pod
						
To test the persistence, delete the guestbook and recreate it using the same volumes, the old messages on the guest book should show up. 
					
				
$ kubectl delete -f guestbook-all-in-one.yaml 
$ kubectl create -f guestbook-all-in-one.yaml
					
						
After create, the port mapping will change. 
				
```						
# Get the new port
$ kubectl describe service frontend| grep NodePort 
Type:                 NodePort
NodePort:      <unset> 32400/TCP 
```

Check the nodes on which the frontend is running 				
 
 
