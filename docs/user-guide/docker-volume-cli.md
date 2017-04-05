# Using the Service in Docker

## Creation and management of docker volumes
The docker volume commands are supported but there is certain minor difference in what is supported in vSphere vs. what is supported in Photon. Let’s cover configurations common to both drivers and then the individual driver specific commands.


### Common

#### Size
You can specify the size of volume while creating a volume. Supported units of sizes are mb, gb and tb. By default if you don’t specify the size, a 100MB volume is created.

```
docker volume create --driver=<vsphere/photon> --name=MyVolume -o size=10gb
```

#### File System Type (fstype)
You can specify the filesystem which will be used it to create the volumes. The docker plugin will look for existing filesystesm in /sbin/mkfs.fstype but if the specified filesystem is not found then it will return a list for which it has found mkfs. The default filesystem if not specified is ext4.

```
docker volume create --driver=<vsphere/photon> --name=MyVolume -o size=10gb -o fstype=xfs
docker volume create --driver=<vsphere/photon> --name=MyVolume -o size=10gb -o fstype=ext4 (default)

```

### vSphere

#### vsan-policy-name
For the vSphere driver you can specify the vsan policy name. The policy itself must be created or should be present before using this in volume creation. You can use vmdkops-admin-cli for creation of policy. The syntax for passing policy name while creating volume looks like this:
```
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o vsan-policy-name=allflash

```

#### Disk Format (diskformat)
The docker volumes are backed by VMDK and there are types of VMDK. At the moment following types of VMDKs are supported:

| VMDK Type             | Abbreviation to be used in volume command|
| ---------------------:|:------------------|
| Thick Provision Lazy Zeroed |zeroedthick|
|Thin Provision|thin|
|Thick Provision Eager Zeroed|eagerzeroedthick|

#### Attach types

#### Disk Modes



### Photon


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
