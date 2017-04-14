---
title: Using the Service in Docker
---
## Using the Service in Docker

## Creation and management of docker volumes
The docker volume commands are supported but there is certain minor difference in what is supported in vSphere vs. what is supported in Photon. Let’s cover configurations common to both drivers and then the individual driver specific commands.


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

#### vsan-policy-name
For the vSphere driver you can specify the vsan policy name. The policy itself must be created or should be present before using this in volume creation. You can use vmdkops-admin-cli for creation of policy. The syntax for passing policy name while creating volume looks like this:
```
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o vsan-policy-name=allflash

```

#### Disk Format (diskformat)
The docker volumes are backed by VMDK and there are types of VMDK. At the moment following types of VMDKs are supported:

<table class="table table-striped table-hover ">
  <thead>
    <tr>
      <th>VMDK Type </th>
      <th>Abbreviation to be used in volume command</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Thick Provision Lazy Zeroed</td>
      <td>zeroedthick</td>
    </tr>
    <tr>
      <td>Thin Provision</td>
      <td>thin</td>
    </tr>
    <tr>
      <td>Thick Provision Eager Zeroed</td>
      <td>eagerzeroedthick</td>
    </tr>
</tbody>
</table>


#### Disk Modes (attach-as)
Docker volumes used in vDVS are backed by VMDKs. VMDKs are attached to hosts on which containers are running. These VMDKs can be attached in [different modes.](http://cormachogan.com/2013/04/16/what-are-dependent-independent-disks-persistent-and-non-persisent-modes/)

1. Persistent mode: In persistent mode the VMDK becomes part of VM the snapshot if a snapshot of the VM is taken while VMDK was attached.
2. Independent Persistent Mode: In the independent persistent mode the VMDK snapshotting is indepedent of the VM snapshot lifecycle. Even if a snapshot of the VM is taken while the Volume is attached to host, the volume VMDK does not become part of the VM snapshot.

```
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o attach-as=independent_persistent
docker volume create --driver=vsphere --name=MyVolume -o size=10gb -o attach-as=persistent
```

#### Volume Cloning (clone-from)

When creating a new volume, you can specificy a volume to clone and create a new one. This is a complete new volume of which you can change all parameters except size and fstype.


```
docker volume create --driver=vsphere --name=CloneVolume -o clone-from=MyVolume -o access=read-only
docker volume create --driver=vsphere --name=CloneVolume -o clone-from=MyVolume -o diskformat=thin (default)
```

## Listing Volumes
Docker volume list can be used to volume names & their DRIVER type
```
docker volume ls
DRIVER              VOLUME NAME
vsphere                MyVolume@vsanDatastore
vsphere                minio1@vsanDatastore
vsphere                minio2@vsanDatastore
photon                 redis-data@vsanDatastore
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
Note: For disk formats zeroedthick and thin, the allocated size would be total size plus the size of replicas.