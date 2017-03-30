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

After installing the new build, type command “vmgroup ls”
Check for failure to connect to auth DB.

```
/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Failed to connect auth DB(DB connection error /etc/vmware/vmdkops/auth-db)
```

The corresponding errors in the vmdk_ops.log file.

```
[root@localhost:~] cat /var/log/vmware/vmdk_ops.log

08/29/16 08:20:23 297059 [MainThread] [ERROR  ] version 0.0 in auth-db does not match latest DB version 1.0
08/29/16 08:20:23 297059 [MainThread] [ERROR  ] DB upgrade is not supported. Please remove the DB file at /etc/vmware/vmdkops/auth-db. All existing configuration will be removed and need to be recreated after removing the DB file.
```

### How to handle the upgrade manually?

#### Case 1: No vmgroup configured before

If no vmgroup has been configured, user just needs to delete the auth-db file

Step 1: Remove  auth-db file at /etc/vmware/vmdkops/auth-db

```
[root@localhost:/etc/vmware/vmdkops]rm /etc/vmware/vmdkops/auth-db
```

Step 2: Verify “vmgroup ls” command
```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name       Description                 Default_datastore  VM_list
------------------------------------  ---------  --------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT   This is a default vmgroup

```

After this point, the manually upgrade is done, and tenancy operations will succeed.

#### Case2: Has vmgroup configured before
Step 1: Backup data manually.

Example below has a vmgroup ```vmgroup1``` with VM ```photon-6``` assigned to this vmgroup1 and one volumes: vol1@datastore1 created.

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

Step 3: Verify “vmgroup ls” command, now only  ```_DEFAULT``` should be listed.

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name      Description                 Default_datastore  VM_list
------------------------------------  --------  --------------------------  -----------------  -------
11111111-1111-1111-1111-111111111111  _DEFAULT  This is a default vmgroup


```

Step 4: Recreate the vmgroup configuration with new name “new-vmgroup1” (associate the same VM photon-6 to this new-vmgroup1), see the following example:

***Note: Please DO NOT create the vmgroup with the old name “vmgroup1”!!!***

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup create --name=new-vmgroup1  --vm-list=photon-6
vmgroup 'new-vmgroup1' is created.  Do not forget to run 'vmgroup vm add' and 'vmgroup access add' commands to enable access control.
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup access add --name=new-vmgroup1 --datastore=datastore1  --volume-maxsize=500MB --volume-totalsize=1GB --allow-create
vmgroup access add succeeded

[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py vmgroup ls
Uuid                                  Name           Description                 Default_datastore  VM_list
------------------------------------  -------------  --------------------------  -----------------  --------
11111111-1111-1111-1111-111111111111  _DEFAULT       This is a default vmgroup
5c0927fb-86b5-4034-87db-8bdfa24018d4  new-vmgroup1                              datastore1         photon-6



```

Step 4: Run “docker volume ls” from VM “photon-6”,  volume which belongs to “vmgroup1” which was created before will not be visible
```
root@photon-JQQBWNwG6 [ ~ ]# docker volume ls
DRIVER              VOLUME NAME
```

Step 5: Run “docker volume create”  to create a new volume “new-vol1” and run “docker volume ls”,   should only able to see this volume which was just created
```
root@photon-KwqUODFXp [ ~ ]# docker volume create --driver=vsphere --name=new-vol1 -o size=100MB
new-vol1
root@photon-KwqUODFXp [ ~ ]# docker volume ls
DRIVER              VOLUME NAME
vsphere             new-vol1@datastore1
```

Volume “vol1” which was created before still exists, and can be seen from the following AdminCLI command

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py volume ls
Volume    Datastore   VMGroup       Capacity  Used  Filesystem  Policy  Disk Format  Attached-to  Access      Attach-as               Created By  Created Date
--------  ----------  ------------  --------  ----  ----------  ------  -----------  -----------  ----------  ----------------------  ----------  ------------------------
vol1      datastore1  N/A           100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Wed Sep 14 16:20:30 2016
new-vol1  datastore1  new-vmgroup1  100MB     13MB  ext4        N/A     thin         detached     read-write  independent_persistent  photon-6    Wed Sep 14 16:22:58 2016


```

Step6: Manually copy the data from backup to the new volume "new-vol1@datastore1".
The path which stores this new volume is "/vmfs/volumes/datastore1/dockvols/new-vmgroup1".
