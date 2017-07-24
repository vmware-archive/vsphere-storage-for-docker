---
title: Persistent Volumes & Persistent Volumes Claims
---

In case of Kubernetes Volumes we know that once the Pod is deleted the specification of the volume in the Pod is also lost. Even though VMDK file persists but from Kubernetes perspective the volume is deleted.
 
Persistent Volumes API resource solves this problem where PVs have lifecycle independent of the Pods and not created when Pod is run. PVs are unit of storage which we provision in advance, they are Kubernetes objects backed by some storage, vSphere in this case. PVs are created, deleted using kubectl commands.
 
In order to use these PVs user needs to create PersistentVolumeClaims which is nothing but a request for PVs. A claim must specify the access mode and storage capacity, once a claim is created PV is automatically bound to this claim. Kubernetes will bind a PV to PVC based on access mode and storage capacity but claim can also mention volume name, selectors and volume class for a better match.
This design of PV-PVCs not only abstract storage provisioning and consumption but also ensures security through access control. 

**Note:**

All the example yamls can be found [here](https://github.com/Kubernetes/kubernetes/tree/master/examples/volumes/vsphere) unless otherwise specified. Please download these examples.

Here is an example of how to use PV and PVC to add persistent storage to your Pods.

**Create VMDK**

First ssh into ESX and then use following command to create vmdk,

```
vmkfstools -c 2G /vmfs/volumes/datastore1/volumes/myDisk.vmdk
```

**Create Persistent Volume**

```
#vsphere-volume-pv.yaml

apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv0001
spec:
  capacity:
    storage: 2Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  vsphereVolume:
    volumePath: "[datastore1] volumes/myDisk"
    fsType: ext4
```

In the above example datastore1 is located in the root folder. If datastore is member of Datastore Cluster or located in sub folder, the folder path needs to be provided in the VolumePath as below.

```
vsphereVolume:
    VolumePath:	"[DatastoreCluster/datastore1] volumes/myDisk"
```


**Create the persistent volume**

```
$ kubectl create -f vsphere-volume-pv.yaml
```

**Verify persistent volume is created**

```
$ kubectl describe pv pv0001
Name:		pv0001
Labels:		<none>
Status:		Available
Claim:
Reclaim Policy:	Retain
Access Modes:	RWO
Capacity:	2Gi
Message:
Source:
    Type:	vSphereVolume (a Persistent Disk resource in vSphere)
    VolumePath:	[datastore1] volumes/myDisk
    FSType:	ext4
No events.
```

**Create Persistent Volume Claim**

```
#vsphere-volume-pvc.yaml

kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: pvc0001
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
```

**Create the persistent volume claim**

```
$ kubectl create -f vsphere-volume-pvc.yaml
```

**Verify persistent volume claim is created**

```
$ kubectl describe pvc pvc0001
Name:		pvc0001
Namespace:	default
Status:		Bound
Volume:		pv0001
Labels:		<none>
Capacity:	2Gi
Access Modes:	RWO
No events.
```

**Create Pod which uses Persistent Volume Claim**

```
#vpshere-volume-pvcpod.yaml

apiVersion: v1
kind: Pod
metadata:
  name: pvpod
spec:
  containers:
  - name: test-container
    image: gcr.io/google_containers/test-webserver
    volumeMounts:
    - name: test-volume
      mountPath: /test-vmdk
  volumes:
  - name: test-volume
    persistentVolumeClaim:
      claimName: pvc0001
```

**Create the pod**

```
$ kubectl create -f vsphere-volume-pvcpod.yaml
```

**Verify pod is created**

```
$ kubectl get pod pvpod
NAME      READY     STATUS    RESTARTS   AGE
pvpod       1/1     Running   0          48m
```
