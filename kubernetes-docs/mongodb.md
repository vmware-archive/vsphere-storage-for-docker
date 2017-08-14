---
title: Deploying Sharded MongoDB Cluster
--- 
				
This section describes the steps to create persistent storage for containers to be consumed by MongoDB services on vSAN. After these steps are completed, Cloud Provider will create the virtual disks (volumes in Kubernetes) and mount them to the Kubernetes nodes automatically. The virtual disks are created with the vSAN default policy. 
				
						
**Define StorageClass**
					
A StorageClass provides a mechanism for the administrators to describe the “classes” of storage they offer. Different classes map to quality-of-service levels, or to backup policies, or to arbitrary policies determined by the cluster administrators. The YAML format defines a “platinum” level StorageClass.
 
```						
kind: StorageClass
apiVersion: storage.k8s.io/v1beta1 
Metadata:
name: platinum
provisioner: Kubernetes.io/vsphere-volume
	diskformat: thin
```					
				
**Note:** Although all volumes are created on the same vSAN datastore, you can adjust the policy according to actual storage capability requirement by modifying the vSAN policy in vCenter Server. User can also specify VSAN storage capabilities in StorageClass definition based on this application needs. Please refer to VSAN storage capability section mentioned in vSphere CP document 
					
						
**Claim Persistent Volume**

A PersistentVolumeClaim (PVC) is a request for storage by a user. Claims can request specific size and access modes (for example, can be mounted once read/write or many times read-only). The YAML format claims a 128GB volume with read and write capability.

```						
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
name: pvc128gb
annotations:
 volume.beta.Kubernetes.io/storage-class: 
"platinum"
spec:
accessModes:
- ReadWriteOnce
resources:
requests:
storage: 128Gi 
```
	
					
**Specify the Volume to be Mounted for the Consumption by the Containers**
						
The YAML format specifies a MongoDB 3.4 image to use the volume from Step 2 and mount it to path
/data/db.

```				
spec:
     containers:
     - image: mongo:3.4
       name: mongo-ps
       ports:
       - name: mongo-ps
        containerPort: 27017
          hostPort: 27017
       volumeMounts:
       - name: pvc-128gb
             mountPath: /data/db
    volumes:
       - name: pvc-128gb
         persistentVolumeClaim:
              claimName: pvc128gb 
```

Storage was created and provisioned from vSAN for containers for the MongoDB service by using dynamic provisioning in YAML files. Storage volumes were claimed as persistent ones to preserve the data on the volumes. All mongo servers are combined into one Kubernetes pod per node.

In Kubernetes, as each pod gets one IP address assigned, each service within a pod must have a distinct port. As the mongos are the services by which you access your shard from other applications, the standard MongoDB port 27017 is assigned to them.


Please refer this [Reference Architecture](https://storagehub.vmware.com/#!/vmware-vsan/vmware-vsan-tm-as-persistent-storage-for-mongodb-in-containers) for detailed understanding of how persistent storage for containers is consumed by MongoDB services on vSAN.


Download the yaml files for deploying MondoDB on Kubernetes with vSphere Cloud Provider from [here](https://github.com/vmware/kubernetes/tree/kube-examples/kube-examples/guestbook/guestbook-storageclass)

To understand the configuration mentioned in these YAMLs please refer this [link](https://storagehub.vmware.com/#!/vmware-vsan/vmware-vsan-tm-as-persistent-storage-for-mongodb-in-containers/mongodb-deployment)  

Execute following commands to deploy Sharded MongoDB Cluster on Kubernetes with vSphere Cloud Provider.

**Create StaogeClass**

```
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/storageclass.yaml
```

**Create Storage Volumes for Shared MondoDB Cluster**

```
kubectl create -f https://github.com/vmware/kubernetes/blob/kube-examples/kube-examples/mongodb-shards/storage-volumes-node01.yaml
kubectl create -f https://github.com/vmware/kubernetes/blob/kube-examples/kube-examples/mongodb-shards/storage-volumes-node02.yaml
kubectl create -f https://github.com/vmware/kubernetes/blob/kube-examples/kube-examples/mongodb-shards/storage-volumes-node03.yaml
kubectl create -f https://github.com/vmware/kubernetes/blob/kube-examples/kube-examples/mongodb-shards/storage-volumes-node03.yaml
```

**Create Mongo DB pods**

```
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node01-deployment.yaml
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node02-deployment.yaml
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node03-deployment.yaml
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node03-deployment.yaml
```	

**Create Services**

```
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node01-service.yaml
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node02-service.yaml
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node03-service.yaml
kubectl create -f https://raw.githubusercontent.com/vmware/kubernetes/kube-examples/kube-examples/mongodb-shards/node04-service.yaml
```