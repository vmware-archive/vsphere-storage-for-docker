# Introduction

vSphere Docker Volume Service is simple to install. It has zero configuration and and zero credential management post install.

[Tagged releases](https://github.com/vmware/docker-volume-vsphere/releases) include the software bundles to install on ESX and on the VM.

In addition the ```make build-all``` will generate the packages.

# Install on ESX [![VIB_Download](https://api.bintray.com/packages/vmware/vDVS/VIB/images/download.svg)](https://bintray.com/vmware/vDVS/VIB/_latestVersion)

[VIB](http://blogs.vmware.com/vsphere/2011/09/whats-in-a-vib.html) and [Offline Depot](https://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html?resultof=%2522%256f%2566%2566%256c%2569%256e%2565%2522%2520%2522%256f%2566%2566%256c%2569%256e%2522%2520%2522%2564%2565%2570%256f%2574%2522%2520) are the packages built to install the backend for the service on ESX. The backend can be installed using esxcli or vmware tools such as [VUM](http://pubs.vmware.com/vsphere-60/topic/com.vmware.ICbase/PDF/vsphere-update-manager-601-install-administration-guide.pdf)

Log into ESXi host and download the [latest release](https://bintray.com/vmware/vDVS/VIB/_latestVersion) of vDVS driver VIB on ESXi and initiate the install by specifying the full path to the VIB.

```
# esxcli software vib install -v /tmp/vmware-esx-vmdkops-0.12.ccfc38f.vib
Installation Result
   Message: Operation finished successfully.
   Reboot Required: false
   VIBs Installed: VMWare_bootbank_esx-vmdkops-service_0.12.ccfc38f-0.0.1
   VIBs Removed:
   VIBs Skipped:
```

# Install on VM

**vDVS managed plugin**
* **Prerequisite**: Docker 1.13/17.03 and above

**vDVS managed plugin life cycle**

* **Installation**
```
~# docker plugin install --grant-all-permissions --alias vsphere vmware/docker-volume-vsphere:latest
latest: Pulling from vmware/docker-volume-vsphere
f07d34084e57: Download complete
Digest: sha256:e1028b8570f37f374e8987d1a5b418e3c591e2cad155cc3b750e5e5ac643fb31
Status: Downloaded newer image for vmware/docker-volume-vsphere:latest
Installed plugin vmware/docker-volume-vsphere:latest

~# docker plugin ls
ID                  NAME                DESCRIPTION                           ENABLED
831fd45343af        vsphere:latest      VMWare vSphere Docker Volume plugin   true
```

* **Enabling/Disabling plugin**

```
~# docker plugin disable vsphere
vsphere
~# docker plugin ls
ID                  NAME                DESCRIPTION                           ENABLED
831fd45343af        vsphere:latest      VMWare vSphere Docker Volume plugin   false

~# docker plugin enable vsphere
 vsphere
~# docker plugin ls
ID                  NAME                DESCRIPTION                           ENABLED
831fd45343af        vsphere:latest      VMWare vSphere Docker Volume plugin   true
```

* **Removing plugin**

```
~# docker plugin rm vsphere
vsphere
~# docker plugin ls
ID                  NAME                DESCRIPTION         ENABLED
```

**vDVS DEB/RPM based installation**

**Note** DEB/RPM packages will be deprecated going forward and will not be available.

```
- sudo dpkg -i <name>.deb # Ubuntu or deb based distros
- sudo rpm -ivh <name>.rpm # Photon or rpm based distros
```