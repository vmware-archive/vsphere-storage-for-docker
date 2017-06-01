---
title: HA Database with Docker in Swarm mode
---

## HA Database with Docker in Swarm mode

In this section we will run a database in Swarm cluster and will demonsrtate high availability even if one of nodes in the cluster is killed. The demo will also show how a volume is re-attached to a new node when the application container is re-scheduled on a new node.

## Setup
We have a 3 node setup and we will create a swarm cluster with one node acting as manager and two nodes acting as workers

## Create a Swarm Cluster

We will first create a swarm cluster between the nodes. To begin with run the following command on one of worker nodes

```
# docker swarm init --advertise-addr 192.168.159.139
Swarm initialized: current node (3fkhpb54el4odc9tvw4ctxn4r) is now a manager.

To add a worker to this swarm, run the following command:

    docker swarm join \
    --token SWMTKN-1-1ltdiozt1ag7sa0mp6ivy4jpivsgal5jlu7gnlf099kd3uouix-6fq7vsef5v8l7vbvuhgvu9dp8 \
    192.168.159.139:2377
```

Now let's go to twh two worker nodes and run following command:

```
# docker swarm join \
>     --token SWMTKN-1-1ltdiozt1ag7sa0mp6ivy4jpivsgal5jlu7gnlf099kd3uouix-6fq7vsef5v8l7vbvuhgvu9dp8 \
>     192.168.159.139:2377
This node joined a swarm as a worker.

```
After the two nodes have been succesfully added to cluster we can do a quick verification and check that all nodes are listed:

```
# docker node ls
ID                           HOSTNAME  STATUS  AVAILABILITY  MANAGER STATUS
3fkhpb54el4odc9tvw4ctxn4r *  photon1   Ready   Active        Leader
6aimhdsw9fqdru54qk3owjnlp    photon5   Ready   Active
dosy6twh0fug6216wj9j3pmtl    photon4   Ready   Active

```

We can also verify that there is no volume that exists on the cluster at the moment:
```
#docker volume ls
DRIVER              VOLUME NAME

```

## Deploy a Service
Let's deploy a service with a single MySQL DB instance and a vSphere volume attached to it. Some key points to be noted here are:

- We are mentioning the the volume driver type as vSphere & size as 1GB
- The volume is attached at /var/lib/mysql - the location where MySQL DB stores the data

```
# docker service create --name db1  --replicas 1 --mount type=volume,source=db_data,target=/var/lib/mysql,volume-driver=vsphere,volume-opt=size=1Gb -p 3306:3306  --env MYSQL_ROOT_PASSWORD=word --env MYSQL_DATABASE=wordpress --env MYSQL_USER=wordpress --env MYSQL_PASSWORD=wordpress MySQLdb
5cu69bufuhjvgncp1t9eusdv7
```


We can verify that the volume has been created:
```
# docker volume ls
DRIVER              VOLUME NAME
vsphere             db_data@datastore3

```
A quick detailed inspection of the volumes also shows all the details of the volume:

```
# docker volume inspect db_data
[
    {
        "Name": "db_data",
        "Driver": "vsphere",
        "Mountpoint": "/mnt/vmdk/db_data",
        "Status": {
            "access": "read-write",
            "attach-as": "independent_persistent",
            "capacity": {
                "allocated": "39MB",
                "size": "1GB"
            },
            "clone-from": "None",
            "created": "Mon Apr 10 14:09:08 2017",
            "created by VM": "Photon5",
            "datastore": "datastore3",
            "diskformat": "thin",
            "fstype": "ext4",
            "status": "detached"
        },
        "Labels": null,
        "Scope": "global"
    }
]
```
A quick inspection of the docker container also shows that the volume is attached

```
root@photon1 [ ~ ]# docker ps -a
CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS               NAMES
d5afd6b94b99        mariadb:latest      "docker-entrypoint.sh"   3 minutes ago       Up 3 minutes        3306/tcp            db1.1.0v25jf29xmi9smul1pkw73txw
root@photon1 [ ~ ]# docker exec -it d5afd6b94b99 bash
root@d5afd6b94b99:/# df -h
Filesystem                                       Size  Used Avail Use% Mounted on
overlay                                           20G  1.1G   18G   6% /
tmpfs                                            497M     0  497M   0% /dev
tmpfs                                            497M     0  497M   0% /sys/fs/cgroup
/dev/sda1                                         20G  1.1G   18G   6% /etc/hosts
shm                                               64M     0   64M   0% /dev/shm
/dev/disk/by-path/pci-0000:03:00.0-scsi-0:0:0:0  976M  116M  794M  13% /var/lib/mysql

```


## Store the data 
Let's create an empty table in the database. In ideal scenario, even if container or the node fails, this data should still persist. 

```
root@d5afd6b94b99:/# mysql -u wordpress -p
Enter password:
Welcome to the MariaDB monitor.  Commands end with ; or \g.
Your MariaDB connection id is 3
Server version: 10.1.22-MariaDB-1~jessie mariadb.org binary distribution

Copyright (c) 2000, 2016, Oracle, MariaDB Corporation Ab and others.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

MariaDB [(none)]> use wordpress;
Database changed
MariaDB [wordpress]> show tables;
Empty set (0.01 sec)

MariaDB [wordpress]> 	
Query OK, 0 rows affected (0.17 sec)

MariaDB [wordpress]> show tables;
+---------------------+
| Tables_in_wordpress |
+---------------------+
| customers           |
+---------------------+
1 row in set (0.00 sec)

MariaDB [wordpress]>


```


## Destructive test & verification

Now to test the availability of the database, let's completely destroy the node on which the MySQL DB is running. This should cause two things to happen:

- The container should be re-scheduled on a different node (This is responsibility of Swarm)
- The volume should get attached to the container on new node (This is what vDVS plugin will provide in this case)

Once we destroy the original node, we can see that the container is scheduled on a different node.
```
# docker ps -a
CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS               NAMES
a1f07c2af9c5        mariadb:latest      "docker-entrypoint.sh"   14 seconds ago      Up 10 seconds       3306/tcp            db1.1.eljid7i5i8kiinm5z8f869kk1
```
Let's login and verify that the data also persists and the volume was re-attached

```
]# docker exec -it a1f07c2af9c5 bash
root@a1f07c2af9c5:/# df -h
Filesystem                                       Size  Used Avail Use% Mounted on
overlay                                           20G  1.1G   18G   6% /
tmpfs                                            497M     0  497M   0% /dev
tmpfs                                            497M     0  497M   0% /sys/fs/cgroup
/dev/sdb1                                         20G  1.1G   18G   6% /etc/hosts
shm                                               64M     0   64M   0% /dev/shm
/dev/disk/by-path/pci-0000:03:00.0-scsi-0:0:0:0  976M  116M  794M  13% /var/lib/mysql
root@a1f07c2af9c5:/# mysql -u wordpress -p
Enter password:
Welcome to the MariaDB monitor.  Commands end with ; or \g.
Your MariaDB connection id is 2
Server version: 10.1.22-MariaDB-1~jessie mariadb.org binary distribution

Copyright (c) 2000, 2016, Oracle, MariaDB Corporation Ab and others.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

MariaDB [(none)]> show databases;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| wordpress          |
+--------------------+
2 rows in set (0.02 sec)

MariaDB [(none)]> use wordpress;
Reading table information for completion of table and column names
You can turn off this feature to get a quicker startup with -A

Database changed
MariaDB [wordpress]> show tables;
+---------------------+
| Tables_in_wordpress |
+---------------------+
| customers           |
+---------------------+
1 row in set (0.00 sec)

MariaDB [wordpress]>

```

## Conclusion

vDVS along with Swarm mode can provide a HA capability to a cluster running stateful applications. While Swarm mode takes care of re-scheduling the application containers on nodes in the cluster, vDVS plugun ensures that the volumes are managed in a similar way and data is persisted while re-scheduling of the application containers.
