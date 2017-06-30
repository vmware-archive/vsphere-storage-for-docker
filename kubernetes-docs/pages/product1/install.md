# Introduction

vSphere Docker Volume Service is simple to install. It has zero configuration and and zero credential management post install.

[Tagged releases](https://github.com/vmware/docker-volume-vsphere/releases) include the software bundles to install on ESX and on the VM.

In addition the ```make build-all``` will generate the packages.

# Install on ESX

[VIB](http://blogs.vmware.com/vsphere/2011/09/whats-in-a-vib.html) and [Offline Depot](https://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html?resultof=%2522%256f%2566%2566%256c%2569%256e%2565%2522%2520%2522%256f%2566%2566%256c%2569%256e%2522%2520%2522%2564%2565%2570%256f%2574%2522%2520) are the packages built to install the backend for the service on ESX. The backend can be installed using esxcli or vmware tools such as [VUM](http://pubs.vmware.com/vsphere-60/topic/com.vmware.ICbase/PDF/vsphere-update-manager-601-install-administration-guide.pdf)

Here is a demo show casing esxcli

<script type="text/javascript" src="https://asciinema.org/a/80405.js" id="asciicast-80405" async></script>

# Install on VM

We currently package the service as a RPM and Deb package. This is to be able to start the service before Docker engine starts. We will also support Docker plugin framework once it is ready.

Here is a demo showcasing the install in a Photon OS VM.

<script type="text/javascript" src="https://asciinema.org/a/80412.js" id="asciicast-80412" async></script>