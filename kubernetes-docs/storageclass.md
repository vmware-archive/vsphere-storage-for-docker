---
title: Dynamic Provisioning and StorageClass API
---

With PV and PVCs one can only provision storage statically i.e. PVs first needs to be created before a Pod claims it. However with StorageClass API Kubernetes enables dynamic volume provisioning which is very unique to Kubernetes. This avoids pre-provisioning of storage and storage is provisioned automatically when a user requests it. 
 
StorageClass API object specifies a provisioner and parameters  which are used to decide which volume plugin to be used and provisioner specific parameters.
Provisioner could be AWS EBS, vSphere, OpenStack and so on.
 
vSphere is one of the provisioners and it allows following parameters:

* **diskformat** which can be thin(default), zeroedthick and eagerzeroedthick

* **datastore** is an optional field which can be VMFSDatastore or VSANDatastore. This allows user to select the datastore to provision PV from, if not specified the default datastore from vSphere config file is used.

* **storagePolicyName** is an optional field which is the name of the SPBM policy to be applied. The newly created persistent volume will have the SPBM policy configured with it.
vSAN storage capability parameters which you can specify explicitly. The newly created persistent volume will have these vSAN storage capabilities configured with it. There are additional parameters which are covered in [Storage Policy Management section](/docker-volume-vsphere/kubernetes/policy-based-mgmt.html).

**Note:**

All the example yamls can be found [here](https://github.com/Kubernetes/Kubernetes/tree/master/examples/volumes/vsphere) unless otherwise specified. Please download these examples.

Let us look at an example of how to use StorageClass for dynamic provisioning.

**Note:** Here you don't need to create vmdk it is created dynamically.

**Create Storage Class**

```
#vpshere-volume-sc-fast.yaml

kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fast
provisioner: kubernetes.io/vsphere-volume
parameters:
    diskformat: zeroedthick
    fstype:     ext3
```

You can also specify the datastore in the Storageclass as shown below YAML for dynamic provisioning. The volume will be created on the datastore specified in the storage class. This field is optional. If not specified as shown in above YAML, the volume will be created on the datastore specified in the vsphere config file used to initialize the vSphere Cloud Provider.

```
#vsphere-volume-sc-slow.yaml

kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fast
provisioner: kubernetes.io/vsphere-volume
parameters:
    diskformat: zeroedthick
    datastore: VSANDatastore
```

If datastore is member of DataStore Cluster or within some sub folder, the datastore folder path needs to be provided in the datastore parameter as below.

```
   datastore:	DatastoreCluster/VSANDatastore
```

**Create the Storageclass**

```
$ kubectl create -f examples/volumes/vsphere/vsphere-volume-sc-fast.yaml
```

**Verifying storage class is created**

```
$ kubectl describe storageclass fast 
Name:           fast
IsDefaultClass: No
Annotations:    <none>
Provisioner:    kubernetes.io/vsphere-volume
Parameters:     diskformat=zeroedthick,fstype=ext3
No events.
```

**Create Persistent Volume Claim**

```
Vsphere-volume-pvcsc.yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: pvcsc001
  annotations:
    volume.beta.kubernetes.io/storage-class: fast
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
```

**Create the persistent volume claim**

```
$ kubectl create -f examples/volumes/vsphere/vsphere-volume-pvcsc.yaml
```

**Verify persistent volume claim is created**

```
$ kubectl describe pvc pvcsc001
Name:           pvcsc001
Namespace:      default
StorageClass:   fast
Status:         Bound
Volume:         pvc-83295256-f8e0-11e6-8263-005056b2349c
Labels:         <none>
Capacity:       2Gi
Access Modes:   RWO
Events:
  FirstSeen LastSeen Count  From        SubObjectPath   Type  Reason Message
  -----------------------------------------------------------
  1m          1m      1   persistentvolume-controller  Normal  Provisioning Succeeded   

  Successfully provisioned volume pvc-83295256-f8e0-11e6-8263-005056b2349c using Kubernetes.io/vsphere-volume
```

Persistent Volume is automatically created and is bounded to this pvc.

**Verify persistent volume claim is created**

```
$ kubectl describe pv pvc-83295256-f8e0-11e6-8263-005056b2349c
Name:           pvc-83295256-f8e0-11e6-8263-005056b2349c
Labels:         <none>
StorageClass:   fast
Status:         Bound
Claim:          default/pvcsc001
Reclaim Policy: Delete
Access Modes:   RWO
Capacity:       2Gi
Message:
Source:
    Type:       vSphereVolume (a Persistent Disk resource in vSphere)
    VolumePath: [datastore1] kubevols/kubernetes-dynamic-pvc-83295256-f8e0-11e6-8263-005056b2349c.vmdk
    FSType:     ext3
No events.
```

**Note:** VMDK is created inside kubevols folder in datastore which is mentioned in 'vsphere' cloudprovider configuration. The cloudprovider config is created during setup of Kubernetes cluster on vSphere.

**Create Pod which uses Persistent Volume Claim with storage class.**

```
#vsphere-volume-pvcscpod.yaml

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
      claimName: pvcsc001
```

**Create the pod**

```
$ kubectl create -f vsphere-volume-pvcscpod.yaml
```

**Verify pod is created**

```
$ kubectl get pod pvpod
NAME      READY     STATUS    RESTARTS   AGE
pvpod       1/1      Running   0          48m
```