[TOC]

# General

## Where do I get the binaries ? What about the source ?
Please look at [GitHub Releases](https://github.com/vmware/docker-volume-vsphere/releases) for binaries. Github releases allow downloading of source for a release in addition to git clone.

## How to install and use the driver?
Please see README.md in the for the release by clicking on the tag for the release. Example: [README](https://github.com/vmware/docker-volume-vsphere/tree/0.1.0.tp.2)

## How do I run the setup on my laptop?
Follow the [guide on the wiki](https://github.com/vmware/docker-volume-vsphere/wiki/Using-laptop-for-running-the-entire-stack)

# Troubleshooting

## Docker Service to ESX Backend Communication.

### What is VMCI and vSock and why is it needed?

vSphere Docker Volume Service uses VMCI and vSock to communicate with the hypervisor to implement the volume operations. It comes installed on Photon OS and on Ubuntu follow [VMware tools installation](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.vm_admin.doc/GUID-08BB9465-D40A-4E16-9E15-8C016CC8166F.html#GUID-08BB9465-D40A-4E16-9E15-8C016CC8166F) or use open vmtools 
```apt-get install open-vm-tools```. 
Additional reading for differences between VMware tools and open vm tools: 

* [Open-VM-Tools (OVT): The Future Of VMware Tools For Linux](http://blogs.vmware.com/vsphere/2015/09/open-vm-tools-ovt-the-future-of-vmware-tools-for-linux.html) 
* [VMware Tools vs Open VM Tools](http://superuser.com/questions/270112/open-vm-tools-vs-vmware-tools)

### I see "connection reset by peer (errno=104)" in the [service's logs](https://github.com/vmware/docker-volume-vsphere#logging), what is the cause?
104 is a standard linux error (```#define ECONNRESET      104     /* Connection reset by peer */```)

It occurs if the Docker volume service cannot communicate to the ESX back end. This can happen if:
   * VMCI and/or vSock kernel modules are not loaded or the kernel does not support VMCI and vSock. Please read "What is VMCI and vSock and why is it needed?" above.
   * ESX service is not running. ```/etc/init.d/vmdk-opsd status```. Check [ESX Logs](https://github.com/vmware/docker-volume-vsphere#logging)
   * ESX service and the docker volume service are not communicating on the same port. ```ps -c | grep vmdk #On ESX``` and ```ps aux| grep docker-volume-vsphere # On VM``` check the port param passed in and make sure they are the same

### I see "address family not supported by protocol (errno=97)" in the [service's logs](https://github.com/vmware/docker-volume-vsphere#logging), what is the cause?
97 is a standard linux error (```#define EAFNOSUPPORT    97      /* Address family not supported by protocol */```)

It occurs if the linux kernel does not know about the AF family used for VMCI communication. Please read ["What is VMCI and vSock and why is it needed?"](https://vmware.github.io/docker-volume-vsphere/user-guide/faq/#what-is-vmci-and-vsock-and-why-is-it-needed) above.

## Upgrade to version 0.10 (Dec 2016) release

Tenancy changes in release 0.10 need a manual upgrade process enumerated below.
***Save the desired tenancy configuration before upgrade***

### How to know if auth-db upgrade is needed post install?

After installing the new build, type command “tenant ls”
Check for failure to connect to auth DB.

```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Failed to connect auth DB(DB connection error /etc/vmware/vmdkops/auth-db)
```

The corresponding errors in the vmdk_ops.log file.

```
[root@localhost:~] cat /var/log/vmware/vmdk_ops.log
 
08/29/16 08:20:23 297059 [MainThread] [ERROR  ] version 0.0 in auth-db does not match latest DB version 1.0
08/29/16 08:20:23 297059 [MainThread] [ERROR  ] DB upgrade is not supported. Please remove the DB file at /etc/vmware/vmdkops/auth-db. All existing configuration will be removed and need to be recreated after removing the DB file.
```
 
### How to handle the upgrade manually?

#### Case 1: No tenant configured before
 
If no tenant has been configured, user just needs to delete the auth-db file

Step 1: Remove  auth-db file at /etc/vmware/vmdkops/auth-db

``` 
[root@localhost:/etc/vmware/vmdkops]rm /etc/vmware/vmdkops/auth-db
```

Step 2: Verify “tenant ls” command
``` 
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list 
------------------------------------  --------  ------------------------  -----------------  ------- 
775888a6-6e98-4f41-9ff2-2ab12afd98de  _DEFAULT  This is a default tenant
```
 
After this point, the manually upgrade is done, and tenancy operations will succeed.
 
#### Case2:  Has tenant configured before
Step 1: Backup data manually.

Example below has a tenant ```tenant1``` with VM ```photon4``` assigned to this tenant and one volumes: vol1@datastore1 created.

```
root@photon-JQQBWNwG6 [ ~ ]# docker volume ls
DRIVER              VOLUME NAME
vmdk                vol1@datastore1
```

User needs to manually backup data stored in vol1@datastore1.

Step 2: Move the auth-db file at /etc/vmware/vmdkops/auth-db

```
[root@localhost:/etc/vmware/vmdkops]mv /etc/vmware/vmdkops/auth-db /etc/vmware/vmdkops/auth-db.backup.v10.upgrade
```

Step 3: Verify “tenant ls” command, now only  ```_DEFAULT``` should be listed.

``` 
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name      Description               Default_datastore  VM_list 
------------------------------------  --------  ------------------------  -----------------  ------- 
775888a6-6e98-4f41-9ff2-2ab12afd98de  _DEFAULT  This is a default tenant                             
```

Step 4: Recreate the tenant configuration with new name “new-tenant1” (associate the same VM photon4 to this new-tenant1), see the following example:

***Note: Please DO NOT create the tenant with the old name “tenant1”!!!***

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant create --name=new-tenant1  --vm-list=photon4
tenant create succeeded
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant access add --name=new-tenant1 --datastore=datastore1  --volume-maxsize=500MB --volume-totalsize=1GB --allow-create
tenant access add succeeded
 
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant ls
Uuid                                  Name         Description               Default_datastore  VM_list 
------------------------------------  -----------  ------------------------  -----------------  ------- 
775888a6-6e98-4f41-9ff2-2ab12afd98de  _DEFAULT     This is a default tenant                             
d5964623-f4bd-4fa6-af4f-b7fa7f51ba5e  new-tenant1                            datastore1         photon4 
```

Step 4: Run “docker volume ls” from VM “photon4”,  volume which belongs to “tenant1” which was created before will not be visible
``` 
root@photon-JQQBWNwG6 [ ~ ]# docker volume ls
DRIVER              VOLUME NAME
```

Step 5: Run “docker volume create”  to create a new volume “new-tenant1-vol1” and run “docker volume ls”,   should only able to see this volume which was just created
``` 
root@photon-JQQBWNwG6 [ ~ ]# docker volume create --driver=vsphere --name=new-tenant1-vol1 -o size=100MB
new-tenant1-vol1
root@photon-JQQBWNwG6 [ ~ ]# docker volume ls
DRIVER              VOLUME NAME
vmdk                new-tenant1-vol1@datastore1
```
 
Volume “vol1” which was created before still exists, and can be seen from the following AdminCLI command

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py ls
Volume            Datastore   Created By VM  Created                   Attached To VM  Policy  Capacity  Used      Disk Format  Filesystem Type  Access      Attach As              
----------------  ----------  -------------  ------------------------  --------------  ------  --------  --------  -----------  ---------------  ----------  ---------------------- 
new-tenant1-vol1  datastore1  photon4        Mon Aug 29 09:17:01 2016  detached        N/A     100.00MB  13.00MB   thin         ext4             read-write  independent_persistent 
vol1              datastore1  photon4        Mon Aug 29 09:09:18 2016  detached        N/A     100.00MB  100.00MB  thin         ext4             read-write  independent_persistent 
```

Step6: Manually copy the data from backup to the new volume "new-tenant1-vol1@datastore1". 
The path which stores this new volume is "/vmfs/volumes/datastore1/dockvols/new-tenant1".
