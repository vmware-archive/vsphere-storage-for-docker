# S3 Object Storage with vSphere Volumes

[Minio](https://www.minio.io/) is lightweight AWS S3 compatible object storage server built for cloud applications.

vSphere Docker Volume Service (vDVS) enables running stateful containers backed by proven enterprise class storage. 

With Minio’s simple object storage layer and VMware’s highly available vSphere Docker Volumes, you can reliably run any application that requires object storage with vDVS for e.g. Docker Trusted Registry.


## Launch Minio with vDVS and Docker Swarm

Minio Stack (vMinio.yml):
```
version: "3"

services:

  minio:
    image: minio/minio
    ports:
      - "9000:9000"
    volumes:
      - export:/export
      - config:/root/.minio
    environment:
      MINIO_ACCESS_KEY: admin
      MINIO_SECRET_KEY: admin123
    command: server /export 
    deploy:
      restart_policy:
        condition: on-failure

volumes:
   export:
      driver: "vsphere"
      driver_opts:
        size : "5GB"
        fstype : "xfs"
   config:
      driver: "vsphere"
      driver_opts:
        size: "1GB"
        fstype : "xfs"
```

Deploy Minio Stack:
```
docker stack deploy -c vMinio.yml minio
```

Check Minio has started successfully:
```
docker stack ps minio
```

This will create 2 vSphere volumes:
- minio_export: Used to store data
- minio_config: Used to store configuration

```
docker volume ls -f driver=vsphere
DRIVER              VOLUME NAME
vsphere:latest      minio_config@datastore1
vsphere:latest      minio_export@datastore1
```

## Accessing Minio Object Storage:
-	Point your browser to [http://Swarm-Master-IP:9000](http://Swarm-Master-IP:9000) and login using credentials specified in compose file.

-	You can also access Minio Object Storage using [CLI, API](http://docs.minio.io/)
