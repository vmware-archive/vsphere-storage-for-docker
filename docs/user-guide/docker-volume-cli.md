# Using the Service in Docker

The Docker volume commands are supported for both the vSphere and Photon platforms with minor differences in capabilities. Features that are specific to either of the platforms are mentioned explicitly below.
<script type="text/javascript" src="https://asciinema.org/a/80417.js" id="asciicast-80417" async></script>

## Docker volume create options
### size

```
docker volume create --driver=<vsphere/photon> --name=MyVolume -o size=10gb
```

The volume units can be ```mb, gb and tb```

The default volume size is 100mb

### vsan-policy-name (vSphere only)

```
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o vsan-policy-name=allflash
```

Policy needs to be created via the vmdkops-admin-cli. Once policy is created, it can be addressed during create by passing the ```-o vsan-policy-name``` flag.

### diskformat (vSphere only)
```
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o diskformat=zeroedthick
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o diskformat=thin
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o diskformat=eagerzeroedthick
```

Docker volumes are backed by VMDKs. VMDKs support multiple [types](https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=1022242)

Currently the following are supported

1. Thick Provision Lazy Zeroed ([zeroedthick]((https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=1022242)))
2. Thin Provision ([thin]((https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=1022242)))
3. Thick Provision Eager Zeroed ([eagerzeroedthick]((https://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=1022242)))

### attach-as (vSphere only)
```
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o attach-as=independent_persistent
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o attach-as=persistent
```
Docker volumes are backed by VMDKs. VMDKs are attached to the VM in which Docker is requesting for a volume during Docker run. VMDKs can be attached in [different modes.](http://cormachogan.com/2013/04/16/what-are-dependent-independent-disks-persistent-and-non-persisent-modes/)

Currently the following are supported

1. [persistent](http://cormachogan.com/2013/04/16/what-are-dependent-independent-disks-persistent-and-non-persisent-modes/): If the VMDK is attached as persistent it will be part of a VM snapshot. If a VM snapshot has been taken while the Docker volume is attached to a VM, the Docker volume then continues to be attached to the VM that was snapshotted.
2. [independent_persistent](http://cormachogan.com/2013/04/16/what-are-dependent-independent-disks-persistent-and-non-persisent-modes/): If the VMDK is attached as independent_persistent it will not be part of a VM snapshot. The Docker volume can be attached to any VM that can access the datastore independent of snapshots.

### access (vSphere only)
```
docker volume create --driver=vsphere --name=MyVolume -o access=read-only -o diskformat=thin
docker volume create --driver=vsphere --name=MyVolume -o access=read-write -o diskformat=thin (default)
```

The access mode determines if the volume is modifiable by containers in a VM. The access mode allows to first create a volume with write access and initialize it with binary images, libraries (for exmple), and subsequently change the access to "read-only" (via the admin CLI). Thereby, creating content sharable by all containers in a VM.

### fstype
```
docker volume create --driver=<vsphere/photon> --name=MyVolume -o size=10gb -o fstype=xfs
docker volume create --driver=<vsphere/photon> --name=MyVolume -o size=10gb -o fstype=ext4 (default)
```

Specifies which filesystem will be created on the new volume. vSphere Docker Volume Service will search for a existing /sbin/mkfs.**fstype** on the docker host to create the filesystem, and if not found it will return a list of filesystems for which it has found a corresponding mkfs. The specified filesystem must be supported by the running kernel and support labels (-L flag for mkfs). Defaults to ext4 if not specified. 

### clone-from (vSphere only)
```
docker volume create --driver=vsphere --name=CloneVolume -o clone-from=MyVolume -o access=read-only
docker volume create --driver=vsphere --name=CloneVolume -o clone-from=MyVolume -o diskformat=thin (default)
```

Specifies a volume to be cloned when creating a new volume. The created clone is completely independent from the original volume and will inherit the same options, which can be changed with the exception of the size and fstype.
 
### flavor (Photon only)
```
docker volume create --driver=vsphere --name=CloneVolume -o flavor=<Photon persistent disk flavor name>
```

The flavor specifies the name of the persistent disk flavor that must have already been created in the Photon Controller. The flavor indicats the resource limits that are applied to the volume being created.

## Docker volume list
```
docker volume ls
DRIVER              VOLUME NAME
vsphere                MyVolume@vsanDatastore
vsphere                minio1@vsanDatastore
vsphere                minio2@vsanDatastore
vsphere                redis-data@vsanDatastore
```
## Docker volume inspect
You can use `docker volume inspect` command to see vSphere attributes of a particular volume.
```
docker volume create —driver=vmdk —name=MyVolume -o size=2gb -o vsan-policy-name=myPolicy -o fstype=xfs
```
```
docker volume inspect MyVolume
[
    {
        "Driver": "vmdk",
        "Labels": {},
        "Mountpoint": "/mnt/vmdk/MyVolume",
        "Name": "MyVolume",
        "Options": {
            "fstype": "xfs",
            "size": "2gb",
            "vsan-policy-name": "myPolicy"
        },
        "Scope": "global",
        "Status": {
            "access": "read-write",
            "attach-as": "independent_persistent",
            "capacity": {
                "allocated": "32MB",
                "size": "2GB"
            },
            "clone-from": "None",
            "created": "Wed Mar  1 20:06:02 2017",
            "created by VM": "esx1_swarm01",
            "datastore": "vsanDatastore",
            "diskformat": "thin",
            "fstype": "xfs",
            "status": "detached",
            "vsan-policy-name": "myPolicy"
        }
    }
]
```
Note: For disk formats zeroedthick and zeroedthick, the allocated size would be total size plus the size of replicas.

## Docker Compose
```
cat nginx-stack-vsphere.yaml 
version: "3"
services:
  nginx:
    image: nginx
    ports:
      - "5000:80"
    volumes:
      - log:/var/log/nginx
    deploy:
      replicas: 1 
      restart_policy:
        condition: on-failure

volumes:
   log:
      driver: vsphere
```
```
docker stack deploy -c  nginx-stack-vsphere.yaml nginx
```
