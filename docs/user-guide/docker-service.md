###...

# Docker Service

Docker service is way to define a container based application with all metadata such as scaling, volume definitions etc. In this example we will create a docker service and inspect the volumes which are auto created. This will also serve as an example of usage of docker service command with vSphere suppoted volumes.

Let's create a MongoDB service with replica count as 1 and vSphere based volume.
```
# docker service create --replicas 1 --mount type=volume,source=mongodata,target=/data/db,volume-driver=vsphere,volume-opt=size=1Gb  --mount type=volume,source=mongoconfig,target=/da
ta/configdb,volume-driver=vsphere --name mongo1 mongo:3.2 mongod
g4ckq563yw51f2xeixgy4nzh9
```

After successful creation of the service, we can check the details of service created:

```
# docker service ls
ID            NAME    MODE        REPLICAS  IMAGE
g4ckq563yw51  mongo1  replicated  1/1       mongo:3.2

# docker service ps mongo1
ID            NAME      IMAGE      NODE     DESIRED STATE  CURRENT STATE          ERROR  PORTS
vm97k26fzcg6  mongo1.1  mongo:3.2  photon6  Running        Running 6 minutes ago


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

<div class="panel panel-info">
  <div class="panel-heading">
    <h3 class="panel-title">Scaling Persistent Services</h3>
  </div>
  <div class="panel-body">
    Scaling persistent services is not the same as scaling a stateless service. Typically you create the volumes needed by each instance of a database and attach them to the respective container, and implementations might vary from databse to database. For example in case of MongoDB one of patterns is to create one service for each instance of MongoDB and run them as separate services, while use MongoBD's replica set feature to form a scaled service.
  </div>
</div>

We can also assign CPU limits on the databse service & finally remove it.

```
# docker service update --limit-cpu 2 mongo1
mongo1

# docker service rm mongo1
mongo1
```