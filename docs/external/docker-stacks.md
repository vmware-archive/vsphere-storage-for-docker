---
title: Deploy stack with Docker Stacks on vSphere volume
---


Docker stacks are a set of services which make up a complete application in a given environment. The Docker stack is written in the docker-compose format (v3). We will use a stack as an example in which one of services requires data persistence. You can checkout the original [Example App](https://github.com/docker/example-voting-app) which we are using in this demo. We have modified the volume part so that the volumes are provisioned using the vSphere driver.

```
version: "3"

services:

  redis:
    image: redis:3.2-alpine
    ports:
      - "6379"
    networks:
      - voteapp
    deploy:
      placement:
        constraints: [node.role == manager]

  db:
    image: postgres:9.4
    volumes:
      - db-data:/var/lib/postgresql
    networks:
      - voteapp
    deploy:
      placement:
        constraints: [node.role == manager]

  voting-app:
    image: gaiadocker/example-voting-app-vote:good
    ports:
      - 5000:80
    networks:
      - voteapp
    depends_on:
      - redis
    deploy:
      mode: replicated
      replicas: 2
      labels: [APP=VOTING]
      placement:
        constraints: [node.role == worker]

  result-app:
    image: gaiadocker/example-voting-app-result:latest
    ports:
      - 5001:80
    networks:
      - voteapp
    depends_on:
      - db

  worker:
    image: gaiadocker/example-voting-app-worker:latest
    networks:
      voteapp:
        aliases:
          - workers
    depends_on:
      - db
      - redis
    # service deployment
    deploy:
      mode: replicated
      replicas: 2
      labels: [APP=VOTING]
      # service resource management
      resources:
        # Hard limit - Docker does not allow to allocate more
        limits:
          cpus: '0.25'
          memory: 512M
        # Soft limit - Docker makes best effort to return to it
        reservations:
          cpus: '0.25'
          memory: 256M
      # service restart policy
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 120s
      # service update configuration
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: continue
        monitor: 60s
        max_failure_ratio: 0.3
      # placement constraint - in this case on 'worker' nodes only
      placement:
        constraints: [node.role == worker]

networks:
    voteapp:

volumes:
   db-data:
      driver: vsphere
      driver_opts:
         size: 1Gb
```

Lets create the docker stack

```
# docker stack deploy -c docker-stack.yml  vote
Creating network vote_voteapp
Creating service vote_result-app
Creating service vote_worker
Creating service vote_redis
Creating service vote_db
Creating service vote_voting-app
```
We can verify that all 5 services in the stack have been created:


```
# docker stack ls
NAME  SERVICES
vote  5
```

We can get details of all services runnig in the stack


```
# docker stack ps vote
ID            NAME               IMAGE                                        NODE     DESIRED STATE  CURRENT STATE               ERROR                      PORTS
alced2bhauf3  vote_voting-app.1  gaiadocker/example-voting-app-vote:good      photon5  Running        Running 51 seconds ago
vtk98gwadk3n  vote_db.1          postgres:9.4                                 photon6  Running        Running 54 seconds ago
jrho9mw52mbh  vote_redis.1       redis:3.2-alpine                             photon6  Running        Running about a minute ago
makxglhal9m4  vote_worker.1      gaiadocker/example-voting-app-worker:latest  photon5  Running        Running 39 seconds ago
5r18yg6mufsy  vote_result-app.1  gaiadocker/example-voting-app-result:latest  photon6  Running        Running about a minute ago
9i3nx7kq8n95  vote_voting-app.2  gaiadocker/example-voting-app-vote:good      photon4  Running        Running 49 seconds ago
g2aqnsd1w1xv  vote_worker.2      gaiadocker/example-voting-app-worker:latest  photon4  Running        Running 51 seconds ago
```

And to get service specific information:

```
# docker stack services vote
ID            NAME             MODE        REPLICAS  IMAGE
2t66jike2cgz  vote_db          replicated  1/1       postgres:9.4
6hwzg8avyrme  vote_worker      replicated  2/2       gaiadocker/example-voting-app-worker:latest
blw093ace9li  vote_result-app  replicated  1/1       gaiadocker/example-voting-app-result:latest
lkqrihcw1bjw  vote_redis       replicated  1/1       redis:3.2-alpine
vc93m03txr74  vote_voting-app  replicated  2/2       gaiadocker/example-voting-app-vote:good
```

Finally to remove the stack:

```
# docker stack rm vote
Removing service vote_db
Removing service vote_worker
Removing service vote_result-app
Removing service vote_redis
Removing service vote_voting-app
Removing network vote_voteapp
```

Before Docker 17.09-ce release, volume name specified in compose file cannot contain special character like "@". With this limitation, user cannot specify volume with a fullname like "vol@datastore" in the compose file when using the vSphere driver. This limitation has been fixed in Docker 17.09-ce release and Docker compose 3.4. With this fix, user can specify volume with a fullname in the compose file with vSphere driver.  Please see the following example for detail:

```
services:
  postgres:
    image: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_vol:/var/lib/data
    environment:
      - "POSTGRES_PASSWORD=secretpass"
      - "PGDATA=/var/lib/data/db"
    deploy:
      restart_policy:
        condition: on-failure
      placement:
        constraints:
          - node.role == worker
volumes:
   postgres_vol:
      name: "postgres_vol@sharedVmfs-1"
      driver: "vsphere"
      driver_opts:
        size: "1GB"
```
The above YAML file can be used to deploy a PostGres service, the volume used in this service is created by vSphere driver, and the name of the volume can be specified with fullname format.

Now, let us deploy the stack using this YAML file:

```
root@sc-rdops-vm02-dhcp-52-237:~# docker stack deploy -c postgres.yml postgres
Creating network postgres_default
Creating service postgres_postgres
```

After stack deploy, volume "postgres_vol@sharedVmfs-1" has been created by vSphere driver as expected.

```
root@sc-rdops-vm02-dhcp-52-237:~# docker volume ls
DRIVER              VOLUME NAME
vsphere:latest      postgres_vol@sharedVmfs-1

root@sc-rdops-vm02-dhcp-52-237:~# docker volume inspect postgres_vol@sharedVmfs-1
[
    {
        "CreatedAt": "0001-01-01T00:00:00Z",
        "Driver": "vsphere:latest",
        "Labels": null,
        "Mountpoint": "/mnt/vmdk/postgres_vol@sharedVmfs-1/",
        "Name": "postgres_vol@sharedVmfs-1",
        "Options": {},
        "Scope": "global",
        "Status": {
            "access": "read-write",
            "attach-as": "independent_persistent",
            "attached to VM": "worker2-VM2.0",
            "attachedVMDevice": {
                "ControllerPciSlotNumber": "160",
                "Unit": "0"
            },
            "capacity": {
                "allocated": "79MB",
                "size": "1GB"
            },
            "clone-from": "None",
            "created": "Mon Sep 25 21:35:24 2017",
            "created by VM": "worker2-VM2.0",
            "datastore": "sharedVmfs-1",
            "diskformat": "thin",
            "fstype": "ext4",
            "status": "attached"
        }
    }
]
```

Wait for a while, check the service, and the service start running successfully.

```
root@sc-rdops-vm02-dhcp-52-237:~# docker service ps postgres_postgres
ID                  NAME                  IMAGE               NODE                        DESIRED STATE       CURRENT STATE            ERROR               PORTS
vcqjxzyz9ff5        postgres_postgres.1   postgres:latest     sc-rdops-vm02-dhcp-52-237   Running             Running 24 seconds ago

root@sc-rdops-vm02-dhcp-52-237:~#
root@sc-rdops-vm02-dhcp-52-237:~# docker service ls
ID                  NAME                MODE                REPLICAS            IMAGE               PORTS
2y5d7jfsd6cu        postgres_postgres   replicated          1/1                 postgres:latest     *:5432->5432/tcp

```

