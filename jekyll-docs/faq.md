---
title: FAQs
---

## General

### Where do I get the binaries? What about the source?
Please look at [GitHub Releases](https://github.com/vmware/vsphere-storage-for-docker/releases) for binaries. Github releases allow downloading of source for a release in addition to git clone.

### How to install and use the driver?
Please follow the instructions at [installation-on-esxi](http://vmware.github.io/vsphere-storage-for-docker/documentation/install.html#installation-on-esxi).

### How do I run the setup on my laptop?
Follow the [guide on the wiki](https://github.com/vmware/vsphere-storage-for-docker/wiki/Using-laptop-for-running-the-entire-stack)

### How do I bind a volume to a particular container host?
This can be achieved via [Tenancy](http://vmware.github.io/vsphere-storage-for-docker/documentation/tenancy.html).

### How do I mount vSphere volume when there are volumes with the same name created on the different datastores?
If there are volumes with the same name on different datastores then use the long name (e.g. `myVolume@vsanDatastore`). Let's take an example as shown below, full volume name should be passed while mounting the volume.

```
root@sc-rdops-vm02-dhcp-52-237:~# docker volume ls
DRIVER              VOLUME NAME
vsphere:latest      myVolume@sharedVmfs-0
vsphere:latest      myVolume@vsanDatastore
```

### Can I migrate data between Linux and Windows containers?
Volumes created via the Linux plugin are formatted with ext4 by default, and the ones created via the Windows plugin are formatted with NTFS. While it is possible to cross-mount such volumes, the vSphere Storage for Docker plugin doesn't support such cases, nor does it provide any explicit help.

### Which release I should use?
You can choose release based on your needs. There will be two types of releases: edge release and stable release.

Edge release: A monthly release with new features and bug fixes.

Stable release: A quarterly release with reliable/stable updates.

Please refer [release convention](https://github.com/vmware/vsphere-storage-for-docker/blob/master/CONTRIBUTING.md#release-naming-convention) for more details.

### Can I use the full volume like "vol@datastore" in compose file with VDVS?
Yes, it is supported starting from Docker 17.09-ce release and compose 3.4. Please refer to this [example](https://github.com/vmware/vsphere-storage-for-docker/blob/master/docs/external/docker-stacks.md).

## Troubleshooting

### Docker Service to ESX Backend Communication.

#### What is VMCI and vSock and why is it needed?

vSphere Docker Volume Service uses VMCI and vSock to communicate with the hypervisor to implement the volume operations. It comes installed on Photon OS and on Ubuntu follow [VMware tools installation](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.vm_admin.doc/GUID-08BB9465-D40A-4E16-9E15-8C016CC8166F.html#GUID-08BB9465-D40A-4E16-9E15-8C016CC8166F) or use open vmtools
```apt-get install open-vm-tools```.
Additional reading for differences between VMware tools and open vm tools:

* [Open-VM-Tools (OVT): The Future Of VMware Tools For Linux](http://blogs.vmware.com/vsphere/2015/09/open-vm-tools-ovt-the-future-of-vmware-tools-for-linux.html)
* [VMware Tools vs Open VM Tools](http://superuser.com/questions/270112/open-vm-tools-vs-vmware-tools)

#### I see "connection reset by peer (errno=104)" in the [service's logs](https://github.com/vmware/vsphere-storage-for-docker#logging), what is the cause?
104 is a standard linux error (```#define ECONNRESET      104     /* Connection reset by peer */```)

It occurs if the Docker volume service cannot communicate to the ESX back end. This can happen if:
   * VMCI and/or vSock kernel modules are not loaded or the kernel does not support VMCI and vSock. Please read "What is VMCI and vSock and why is it needed?" above.
   * ESX service is not running. ```/etc/init.d/vmdk-opsd status```. Check [ESX Logs](https://github.com/vmware/vsphere-storage-for-docker#logging)
   * ESX service and the docker volume service are not communicating on the same port. ```ps -c | grep vmdk #On ESX``` and ```ps aux| grep vsphere-storage-for-docker # On VM``` check the port param passed in and make sure they are the same

#### I see "address family not supported by protocol (errno=97)" in the [service's logs](https://github.com/vmware/vsphere-storage-for-docker#logging), what is the cause?
97 is a standard linux error (```#define EAFNOSUPPORT    97      /* Address family not supported by protocol */```)

It occurs if the linux kernel does not know about the AF family used for VMCI communication. Please read ["What is VMCI and vSock and why is it needed?"](https://vmware.github.io/vsphere-storage-for-docker/user-guide/faq/#what-is-vmci-and-vsock-and-why-is-it-needed) above.

#### I see "plugin not found" error message in the Docker daemon logs, what is the cause?
This is the limitation of docker and being tracked at [issue#34545](https://github.com/moby/moby/issues/34545).

Docker maintains a unique plugin ID that's assigned when the plugin is installed. Once assigned the ID remains with the plugin till its removed. All volumes that were created via the plugin will remain inaccessible if the plugin is removed. The correct way to install updates to the plugin is to upgrade the plugin using `docker plugin upgrade` as mentioned at [user guide](http://vmware.github.io/vsphere-storage-for-docker/documentation/install.html#upgrade-instructions).

#### Volume remains attached to the VM after upgrading VDVS ESX driver
If the container using volumes exits during the upgrade of ESX driver (i.e. after vib remove but before vib install), the volumes may remain attached to VM. In such a case, please disable and then enable (restart) the volume plugin to ensure volumes are properly detached.

#### I'm not able to create volume after upgrading to VDVS managed plugin, what is the cause?
```
# docker volume create -d vsphere vol5
Error response from daemon: create vol5: Post http://%2Frun%2Fdocker%2Fplugins%2Fvsphere.sock/VolumeDriver.Create: dial unix /run/docker/plugins/vsphere.sock: connect: no such file or directory
```

Restart docker service is required.

e.g.
```
systemctl restart docker
```

#### I'm not able to create volume and I see "VolumeDriver.Create: Device not found" error

From 0.19 release of VDVS, the plugin has to have a VIB that's at least 0.19 or later.  [#2023](https://github.com/vmware/vsphere-storage-for-docker/issues/2023)
Ideally it is better if you always upgrade both the plugin and the VIB to matching versions.