

## Swarm Cluster

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
