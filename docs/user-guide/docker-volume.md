###...

# Docker Volume

This section covers all volume related commands with example and known caveats if any. 

To create a volume with vSphere driver and size specifications:

```
# docker volume create --name=db_data --driver=vsphere -o size=1Gb
db_data
```

To list all volumes:
```
# docker volume ls
DRIVER              VOLUME NAME
vsphere:latest      db_data@datastore3
vsphere:latest      wordpress_db_data@datastore3

```

You can also inspect the volume:
```
# docker volume inspect db_data
[
    {
        "Driver": "vsphere:latest",
        "Labels": {},
        "Mountpoint": "/mnt/vmdk/db_data",
        "Name": "db_data",
        "Options": {
            "size": "1Gb"
        },
        "Scope": "global",
        "Status": {
            "access": "read-write",
            "attach-as": "independent_persistent",
            "capacity": {
                "allocated": "39MB",
                "size": "1GB"
            },
            "clone-from": "None",
            "created": "Tue Apr 11 12:24:37 2017",
            "created by VM": "Photon6",
            "datastore": "datastore3",
            "diskformat": "thin",
            "fstype": "ext4",
            "status": "detached"
        }
    }
]
```

<div class="panel panel-info">
  <div class="panel-heading">
    <h3 class="panel-title">Volume Prune</h3>
  </div>
  <div class="panel-body">
        Docker volume prune command is not supported at the moment
  </div>
</div>

You can remove the volume with following command
```
# docker volume rm db_data
db_data
```