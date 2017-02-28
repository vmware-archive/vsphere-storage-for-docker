# Short Volume names
Short Volume Name can contain up to a 100 characters.
All characters are allowed, but the last '@' will be interpreted as a separator
between volume and datastore name (see below).
As an exception, we do not allow volume names ending with '-NNNNNN', where
N is a digit (this limitation is to prevent conflicts with ESXi deltadisks used
during VM snapshots). Examples: `MyVolume_fast.12`,  `_12SuperStorage`,
`super-volume-200` are all valid names, while `myVolume-000012` is not valid.


# Full volume names
Full volume name is a combination of Short Volume name and datastore name,
separated by @ symbol.
Datastore name is a vSphere Datastore , e.g. vsanDatastore.
Examples: `BigFatDisk@vsanDatastore` or `myVol123.33@datastore255`.
Datastore name length is limited to 100 characters.


# Default datastore 

Our approach is `if you do not care about specific datastore, use just the volume name and the rest will be taken care of`.

If datastore name is omitted, the name of the datastore where Docker VM is located will be used.
E.g. if Docker VM is located on `vsanDatastore`, then 
`myVolume@vsanDatastore` is the same as `myVolume`. 

Full volume names are unique, but short volume names are unique only within a specific Docker VM. For example, if there is a Docker VM1 located on
`vsanDatastore`, and Docker VM2 located on `datastore266`, then in VM1 `myVolume` is referring to `myVolume@vsanDatastore`, while in VM2 'myVolume' is referring
to `myVolume@datastore266` - which are different volumes. However, in both VMs `myVolume@vsanDatastore` and `myvolume@datastore266` can be used to 
uniquely identify the volume.

Note that `docker volume ls` command will strip datastore name from the volumes on the Docker VM datastore. For example, if the Docker VM runs on `datastore266`
and there are 2 volumes on this datastore, and there are also 2 volumes with the same names on vsanDatastore, then 'docker volume ls' will show
the following:
```
$ docker volume ls
DRIVER       VOLUME NAME
vsphere         volume1
vsphere         volume2
vsphere         volume1@vsanDatastore
vsphere         volume2@vsanDatastore
```

# Volume name usage
volume@datastore can be used anywhere where `volume name` is needed - i.e. in 'docker run -v **volume**:/path ...' ,
'docker volume create', 'docker volume rm', docker volume inspect'. In all cases, using short volume name (with no datastore) will refer to the volume 
with the given name located on the same datastore where the Docker VM is located.

