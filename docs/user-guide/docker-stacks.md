---
title: Docker Stacks
---



## Docker Stacks

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