---
title: Deploying S3 Stateful Containers - Minio
---

This case study describes the process to deploy distributed Minio server on Kubernetes. This example uses the official Minio Docker image from Docker Hub.
 
**Create Minio Storage class**

``` 
#minio-sc.yaml
 
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: miniosc
provisioner: Kubernetes.io/vsphere-volume
parameters:
    diskformat: thin
 
```

Creating the storage class:

```
$ kubectl create -f minio-sc.yaml
```

**Create Minio headless Service**

Headless Service controls the domain within which StatefulSets are created. The domain managed by this Service takes the form: $(service name).$(namespace).svc.cluster.local (where “cluster.local” is the cluster domain), and the pods in this domain take the form: $(pod-name-{i}).$(service name).$(namespace).svc.cluster.local. This is required to get a DNS resolvable URL for each of the pods created within the Statefulset.

This is the Headless service description.

```
apiVersion: v1
kind: Service
metadata:
  name: minio
  labels:
    app: minio
spec:
  clusterIP: None
  ports:
    - port: 9000
      name: minio
  selector:
    app: minio
```

Create the headless service

```
$ kubectl create -f https://github.com/vmware/kubernetes/tree/kube-examples/kube-examples/minio/distributed/minio-distributed-headless-service.yaml?raw=true
service "minio" created
```

**Create Minio StatefulSet**

A StatefulSet provides a deterministic name and a unique identity to each pod, making it easy to deploy stateful distributed applications. To launch distributed Minio you need to pass drive locations as parameters to the minio server command. Then, you’ll need to run the same command on all the participating pods. StatefulSets offer a perfect way to handle this requirement.
This is the Statefulset description.

```
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: minio
spec:
  serviceName: minio
  replicas: 4
  template:
    metadata:
      annotations:
        pod.alpha.Kubernetes.io/initialized: "true"
      labels:
        app: minio
    spec:
      containers:
      - name: minio
        env:
        - name: MINIO_ACCESS_KEY
          value: "minio"
        - name: MINIO_SECRET_KEY
          value: "minio123"
        image: minio/minio:RELEASE.2017-05-05T01-14-51Z
        args:
        - server
        - http://minio-0.minio.default.svc.cluster.local/data
        - http://minio-1.minio.default.svc.cluster.local/data
        - http://minio-2.minio.default.svc.cluster.local/data
        - http://minio-3.minio.default.svc.cluster.local/data
        ports:
        - containerPort: 9000
          hostPort: 9000
        # These volume mounts are persistent. Each pod in the PetSet
        # gets a volume mounted based on this field.
        volumeMounts:
        - name: data
          mountPath: /data
  # These are converted to volume claims by the controller
  # and mounted at the paths mentioned above.
  volumeClaimTemplates:
  - metadata:
      name: data
      annotations:
        volume.beta.Kubernetes.io/storage-class: miniosc
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 5Gi
```

Create the Statefulset

``` 
$ kubectl create -f https://github.com/vmware/kubernetes/tree/kube-examples/kube-examples/minio/distributed/minio-distributed-statefulset.yaml?raw=true
statefulset "minio" created
```

**Create service and expose it to external traffic using NodePort**
 
Now that you have a Minio statefulset running, you may either want to access it internally (within the cluster) or expose it as a Service onto an external (outside of your cluster, maybe public internet) IP address, depending on your use case. You can achieve this using Services. 

There are 3 major service types — default type is ClusterIP, which exposes a service to connection from inside the cluster. NodePort and LoadBalancer are two types that expose services to external traffic.
 
In this example, we expose the Minio Deployment by using NodePort. This is the service description.

``` 
#minio_NodePort.yaml
 
apiVersion: v1
kind: Service
metadata:
  name: minio-service
spec:
  type: NodePort
  ports:
    - port: 9000
      nodePort: 30000
  selector:
    app: minio
```

```
$ kubectl create -f minio_NodePort.yaml
service "minio-service" created
``` 
 
**Access Minio** 
 
Find the IP addresses of master nodes

```
$ kubectl describe node master | grep Addresses
Addresses:		10.160.132.97,10.160.132.97,master
``` 
 
Find the NodePort 							

```
$ kubectl describe service minio-service | grep NodePort
Type:			NodePort
NodePort:		<unset>	30000/TCP
```

Use the following URL to access Mnio
```
http://10.160.132.97:30000/minio/login
```