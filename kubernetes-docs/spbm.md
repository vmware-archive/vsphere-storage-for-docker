---
title: Storage Policy Based Management for dynamic provisioning of volumes
---

## Overview

One of the most important features of vSphere for Storage Management is Policy based Management. Storage Policy Based Management (SPBM) is a storage policy framework that provides a single unified control plane across a broad range of data services and storage solutions. SPBM enables vSphere administrators to overcome upfront storage provisioning challenges, such as capacity planning, differentiated service levels and managing capacity headroom
 
As we discussed in previously StorageClass specifies provisioner and parameters. And using these parameters you can define the policy for that particular PV which will be dynamically provisioned. 
 
You can specify the existing vCenter Storage Policy Based Management (SPBM) policy to configure a persistent volume with SPBM policy. storagePolicyName parameter is used for this.
 
**Note:**

* SPBM policy based provisioning of persistent volumes will be available in 1.7.x release.**
* All the example yamls can be found [here](https://github.com/kubernetes/kubernetes/tree/master/examples/volumes/vsphere) unless otherwise specified. Please download these examples.

**Create Storage Class**

```
#sphere-volume-spbm-policy.yaml

kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fast
provisioner: kubernetes.io/vsphere-volume
parameters:
    diskformat: zeroedthick
    storagePolicyName: gold
```

The admin specifies the SPBM policy - "gold" as part of storage class definition for dynamic volume provisioning. When a PVC is created, the persistent volume will be provisioned on a compatible datastore with maximum free space that satisfies the "gold" storage policy requirements.


```
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fast
provisioner: kubernetes.io/vsphere-volume
parameters:
    diskformat: zeroedthick
    storagePolicyName: gold
    datastore: VSANDatastore
```

The admin can also specify a custom datastore where he wants the volume to be provisioned along with the SPBM policy name. When a PVC is created, the vSphere Cloud Provider checks if the user specified datastore satisfies the "gold" storage policy requirements. If yes, it will provision the persistent volume on user specified datastore. If not, it will error out to the user that the user specified datastore is not compatible with "gold" storage policy requirements.

## Virtual SAN policy support

Vsphere Infrastructure(VI) Admins will have the ability to specify custom Virtual SAN Storage Capabilities during dynamic volume provisioning. You can now define storage requirements, such as performance and availability, in the form of storage capabilities during dynamic volume provisioning. The storage capability requirements are converted into a Virtual SAN policy which are then pushed down to the Virtual SAN layer when a persistent volume (virtual disk) is being created. The virtual disk is distributed across the Virtual SAN datastore to meet the requirements.

The official vSAN policy documentation describes in detail about each of the individual storage capabilities that are supported by vSAN. The user can specify these storage capabilities as part of storage class definition based on his application needs.
 
For vSAN policies you can few additional parameters in StorageClass can be specified:

* **cacheReservation:** Flash capacity reserved as read cache for the container object. Specified as a percentage of the logical size of the virtual machine disk (vmdk) object. Reserved flash capacity cannot be used by other objects. Unreserved flash is shared fairly among all objects. Use this option only to address specific performance issues.

* **diskStripes:** The minimum number of capacity devices across which each replica of a object is striped. A value higher than 1 might result in better performance, but also results in higher use of system resources. Default value is 1. Maximum value is 12.

* **forceProvisioning:** If the option is set to Yes, the object is provisioned even if theNumber of failures to tolerate, Number of disk stripes per object, and Flash read cache reservation policies specified in the storage policy cannot be satisfied by the datastore

* **hostFailuresToTolerate:** Defines the number of host and device failures that a virtual machine object can tolerate. For n failures tolerated, each piece of data written is stored in n+1 places, including parity copies if using RAID 5 or RAID 6.

* **iopsLimit:** Defines the IOPS limit for an object, such as a VMDK. IOPS is calculated as the number of I/O operations, using a weighted size. If the system uses the default base size of 32 KB, a 64-KB I/O represents two I/O operations

* **objectSpaceReservation:** Percentage of the logical size of the virtual machine disk (vmdk) object that must be reserved, or thick provisioned when deploying virtual machines. Default value is 0%. Maximum value is 100%.
 
 
**Note:** 

* Here you don't need to create persistent volume it is created dynamically
* vSAN storage capability based provisioning of persistent volumes is available in 1.6.5 release.


**Create Storage Class**

```
#vsphere-volume-sc-vsancapabilities.yaml

kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fast
provisioner: kubernetes.io/vsphere-volume
parameters:
    diskformat: zeroedthick
    hostFailuresToTolerate: "2"
    cachereservation: "20"
```

Here a persistent volume will be created with the Virtual SAN capabilities - hostFailuresToTolerate to 2 and cachereservation is 20% read cache reserved for storage object. Also the persistent volume will be zeroedthickdisk.  

The official vSAN policy documentation describes in detail about each of the individual storage capabilities that are supported by vSAN and can be configured on the virtual disk.
You can also specify the datastore in the Storageclass as shown in above example. The volume will be created on the datastore specified in the storage class. This field is optional. If not specified as shown in example 1, the volume will be created on the datastore specified in the vsphere config file used to initialize the vSphere Cloud Provider.

```
#vsphere-volume-sc-vsancapabilities.yaml

kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: fast
provisioner: kubernetes.io/vsphere-volume
parameters:
    diskformat: zeroedthick
    datastore: VSANDatastore
    hostFailuresToTolerate: "2"
    cachereservation: "20"
```

**Note:** If you do not apply a storage policy during dynamic provisioning on a vSAN datastore, it will use a default Virtual SAN policy.

**Create the storageclass**

```
$ kubectl create -f vsphere-volume-sc-vsancapabilities.yaml
```

**Verify storage class is created**

```
$ kubectl describe storageclass fast
Name:		fast
Annotations:	<none>
Provisioner:	kubernetes.io/vsphere-volume
Parameters:	diskformat=zeroedthick, hostFailuresToTolerate="2", cachereservation="20"
No events.
Create Persistent Volume Claim.
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: pvcsc-vsan
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
$ kubectl create -f vsphere-volume-pvcsc.yaml
```

**Verifying persistent volume claim is created**

```
$ kubectl describe pvc pvcsc-vsan
Name:		pvcsc-vsan
Namespace:	default
Status:		Bound
Volume:		pvc-80f7b5c1-94b6-11e6-a24f-005056a79d2d
Labels:		<none>
Capacity:	2Gi
Access Modes:	RWO
No events.
```

Persistent Volume is automatically created and is bounded to this pvc

**Verify if persistent volume claim is created**

```
$ kubectl describe pv pvc-80f7b5c1-94b6-11e6-a24f-005056a79d2d
Name:		pvc-80f7b5c1-94b6-11e6-a24f-005056a79d2d
Labels:		<none>
Status:		Bound
Claim:		default/pvcsc-vsan
Reclaim Policy:	Delete
Access Modes:	RWO
Capacity:	2Gi
Message:
Source:
    Type:	vSphereVolume (a Persistent Disk resource in vSphere)
    VolumePath:	[VSANDatastore] kubevols/kubernetes-dynamic-pvc-80f7b5c1-94b6-11e6-a24f-005056a79d2d.vmdk
    FSType:	ext4
No events.
```

**Note:** VMDK is created inside kubevols folder in datastore which is mentioned in 'vsphere' cloudprovider configuration. The cloudprovider config is created during setup of Kubernetes cluster on vSphere.

**Create Pod which uses Persistent Volume Claim with storage class**

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
      mountPath: /test
  volumes:
  - name: test-volume
    persistentVolumeClaim:
      claimName: pvcsc-vsan
```

**Create the pod**

```
$ kubectl create -f vsphere-volume-pvcscpod.yaml
Verifying pod is created:
$ kubectl get pod pvpod
NAME      READY     STATUS    RESTARTS   AGE
pvpod       1/1     Running   0          48m
```
