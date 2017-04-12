###...

# Docker Service


```
# docker service create --replicas 1 --mount type=volume,source=mongodata,target=/data/db,volume-driver=vsphere,volume-opt=size=1Gb  --mount type=volume,source=mongoconfig,target=/da
ta/configdb,volume-driver=vsphere --name mongo1 mongo:3.2 mongod
g4ckq563yw51f2xeixgy4nzh9
```



```
# docker service inspect mongo1
[
    {
        "ID": "g4ckq563yw51f2xeixgy4nzh9",
        "Version": {
            "Index": 2172
        },
        "CreatedAt": "2017-04-12T03:40:07.105786713Z",
        "UpdatedAt": "2017-04-12T03:40:07.105786713Z",
        "Spec": {
            "Name": "mongo1",
            "TaskTemplate": {
                "ContainerSpec": {
                    "Image": "mongo:3.2@sha256:847d2ac8eebc500522d288019900ca361b1dda8c0d720de421c60165b8001a6c",
                    "Args": [
                        "mongod"
                    ],
                    "Mounts": [
                        {
                            "Type": "volume",
                            "Source": "mongodata",
                            "Target": "/data/db",
                            "VolumeOptions": {
                                "DriverConfig": {
                                    "Name": "vsphere",
                                    "Options": {
                                        "size": "1Gb"
                                    }
                                }
                            }
                        },
                        {
                            "Type": "volume",
                            "Source": "mongoconfig",
                            "Target": "/data/configdb",
                            "VolumeOptions": {
                                "DriverConfig": {
                                    "Name": "vsphere"
                                }
                            }
                        }
                    ],
                    "DNSConfig": {}
                },
                "Resources": {
                    "Limits": {},
                    "Reservations": {}
                },
                "RestartPolicy": {
                    "Condition": "any",
                    "MaxAttempts": 0
                },
                "Placement": {},
                "ForceUpdate": 0
            },
            "Mode": {
                "Replicated": {
                    "Replicas": 1
                }
            },
            "UpdateConfig": {
                "Parallelism": 1,
                "FailureAction": "pause",
                "MaxFailureRatio": 0
            },
            "EndpointSpec": {
                "Mode": "vip"
            }
        },
        "Endpoint": {
            "Spec": {}
        },
        "UpdateStatus": {
            "StartedAt": "0001-01-01T00:00:00Z",
            "CompletedAt": "0001-01-01T00:00:00Z"
        }
    }
]
```

```
# docker service ls
ID            NAME    MODE        REPLICAS  IMAGE
g4ckq563yw51  mongo1  replicated  1/1       mongo:3.2

# docker service ps mongo1
ID            NAME      IMAGE      NODE     DESIRED STATE  CURRENT STATE          ERROR  PORTS
vm97k26fzcg6  mongo1.1  mongo:3.2  photon6  Running        Running 6 minutes ago
```

```
# docker service scale mongo1=2
mongo1 scaled to 2
```

```
# docker service update --limit-cpu 2 mongo1
mongo1

```

```
# docker service rm mongo1
mongo1
```