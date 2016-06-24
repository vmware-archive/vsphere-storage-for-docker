# Short Volume names
Short Volume Name can contain alphanumeric characters, underscores and dots. Examples: `MyVolume_fast.12` or `_12SuperStorage`.
Volume name is limited to 128 characters. 

# Full volume names
Full volume name is a combination of Short Volume name and Datastore name, separated by @ sign.
Datastore name is a vSphere datastore , e.g. vsanDatastore. Examples: `BigFatDisk@vsanDatastore` or `myVol123.33@datastore255`


# Default datastore 

Our approach is `if you do not care about specific datastore, use just the volume name and the rest will be taken care of`.

If Datastore Name is ommited, the name of the datastore where Docker VM is located will be used.
E.g. if Docker VM is located on `vsanDatastore`, then 
`myVolume@vsanDatastore` is the same as `myVolume`. 

Full volume names are unique, but short volume names are unique only within a specific Docker VM. For example, if there is a Docker VM1 located on
`vsanDatastore`, and Docker VM2 located on `datastore266`, then in VM1 `myVolume` is referring to `myVolume@vsanDatastore`, while in VM2 'myVolume' is referring
to `myVolume@datastore266` - which are different volumes. However, in both VMs `myVolume@vsanDatastore` and `myvolume@datastore266` can be used to 
uniquely identify the volume.

Note that `docker volume ls` command will strip Datastore from the volumes on the VM datastore. For example, if the Docker VM runs on `datastore266`
and there are 2 volumes on this datastore, and there are also 2 volumes with the same names on vsanDatastore, then 'docker volume ls' will show
the following:
```
$ docker volume ls
DRIVER       VOLUME NAME
vmdk         volume1
vmdk         volume 2
vmdk         volume1@vsanDatastore
vmdk         volume2@vsanDatastore
```

# Volume name usage
volume@datastore can be used anywhere where `volume name` is needed - i.e. in 'docker run -v **volume**:/path ...' ,
'docker volume create', 'docker volume rm', docker volume inspect'. In all cases, using short volume name (with no datastore) will refer to the volume 
with the given name located on the same datastore where the Docker VM is located

