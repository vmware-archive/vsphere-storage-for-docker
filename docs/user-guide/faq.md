[TOC]

# General

## Where do I get the binaries ? What about the source ?
Please look at [GitHub Releases](https://github.com/vmware/docker-volume-vsphere/releases) for binaries. Github releases allow downloading of source for a release in addition to git clone.

## How to install and use the driver?
Please see README.md in the for the release by clicking on the tag for the release. Example: [README](https://github.com/vmware/docker-volume-vsphere/tree/0.1.0.tp.2)

## How do I run the setup on my laptop?
Follow the [guide on the wiki](https://github.com/vmware/docker-volume-vsphere/wiki/Using-laptop-for-running-the-entire-stack)

# Troubleshooting

## Docker Plugin to ESX Backend Communication.

### What is VMCI and vSock and why is it needed?

The docker volume plugin for vSphere uses VMCI and vSock to communicate with the hypervisor to implement the volume operations. It comes installed on Photon OS and on Ubuntu follow [VMware tools installation](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.vm_admin.doc/GUID-08BB9465-D40A-4E16-9E15-8C016CC8166F.html#GUID-08BB9465-D40A-4E16-9E15-8C016CC8166F) or use open vmtools 
```apt-get install open-vm-tools```. 
Additional reading for differences between VMware tools and open vm tools: 

* [Open-VM-Tools (OVT): The Future Of VMware Tools For Linux](http://blogs.vmware.com/vsphere/2015/09/open-vm-tools-ovt-the-future-of-vmware-tools-for-linux.html) 
* [VMware Tools vs Open VM Tools](http://superuser.com/questions/270112/open-vm-tools-vs-vmware-tools)

### I see "connection reset by peer (errno=104)" in the [plugin logs](https://github.com/vmware/docker-volume-vsphere#logging), what is the cause?
104 is a standard linux error (```#define ECONNRESET      104     /* Connection reset by peer */```)

It occurs if the Docker volume plugin cannot communicate to the ESX back end. This can happen if:
   * VMCI and/or vSock kernel modules are not loaded or the kernel does not support VMCI and vSock. Please read "What is VMCI and vSock and why is it needed?" above.
   * ESX service is not running. ```/etc/init.d/vmdk-opsd status```. Check [ESX Logs](https://github.com/vmware/docker-volume-vsphere#logging)
   * ESX service and the plugin are not communicating on the same port. ```ps -c | grep vmdk #On ESX``` and ```ps aux| grep docker-volume-vsphere # On VM``` check the port param passed in and make sure they are the same

### I see "address family not supported by protocol (errno=97)" in the [plugin logs](https://github.com/vmware/docker-volume-vsphere#logging), what is the cause?
97 is a standard linux error (```#define EAFNOSUPPORT    97      /* Address family not supported by protocol */```)

It occurs if the linux kernel does not know about the AF family used for VMCI communication. Please read ["What is VMCI and vSock and why is it needed?"](https://vmware.github.io/docker-volume-vsphere/user-guide/faq/#what-is-vmci-and-vsock-and-why-is-it-needed) above.
