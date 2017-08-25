---
title: Installation
---

The installation has two parts – installation of the vSphere Installation Bundle (VIB) on ESXi and installation of Docker plugin on the hosts where you plan to run containers with storage needs.

## Installation on ESXi

[VIB](http://blogs.vmware.com/vsphere/2011/09/whats-in-a-vib.html) and [Offline Depot](https://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html?resultof=%2522%256f%2566%2566%256c%2569%256e%2565%2522%2520%2522%256f%2566%2566%256c%2569%256e%2522%2520%2522%2564%2565%2570%256f%2574%2522%2520) are the packages built to install the backend for the service on ESX. The backend can be installed using esxcli or vmware tools such as [VUM](http://pubs.vmware.com/vsphere-60/topic/com.vmware.ICbase/PDF/vsphere-update-manager-601-install-administration-guide.pdf)

### VUM based VIB installation
ESXi component for vDVS is available in the form of an [Offline Depot](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html) and available to download latest release as [ZIP archive](https://bintray.com/vmware/vDVS/VIB/_latestVersion).

After downloading VIB bundle as zip format, installation is performed using VUM using the following steps.

1. Go to *admin view* in update manager tab.
2. Import zip bundle using *import zip* option.
3. Create a baseline and include zip bundle.
4. Scan your host against baseline.
5. *Remediate* host.

More information can be found at [VUM User guide](https://featurewalkthrough.vmware.com/#!/vsphere-6-5/vsphere-update-manager-overview-cluster-upgrade/1).

### VIB installation through esxcli/localcli

ESXi component for vDVS is available in the form of a [VIB](https://blogs.vmware.com/vsphere/2011/09/whats-in-a-vib.html). VIB stands for vSphere Installation Bundle. At a conceptual level a VIB is similar to a tarball or ZIP archive in that it is a collection of files packaged into a single archive to facilitate distribution.

Log into ESXi host and download the [latest release](https://bintray.com/vmware/vDVS/VIB/_latestVersion) of vDVS driver VIB on the ESXi. Assuming that you have downloaded the VIB at /tmp location you can run the below command to install it on ESXi. You will need ESXi version **6.0 or above**

```
# esxcli software vib install -v /tmp/vmware-esx-vmdkops-0.12.ccfc38f.vib
Installation Result
   Message: Operation finished successfully.
   Reboot Required: false
   VIBs Installed: VMWare_bootbank_esx-vmdkops-service_0.12.ccfc38f-0.0.1
   VIBs Removed:
   VIBs Skipped:
```

**Note**: To make admin commandset available on ESX host, please restart hostd after vib installation.
```
/etc/init.d/hostd restart
```

## Installation on Docker Hosts

vDVS plugin can be installed on Docker hosts like any docker plugin installation. You will need docker version **1.13/17.03 or above** on the VM. In a large pool of nodes, you can push the plugin installation to multiple VM through a configuration management tool such as Ansible/Salt or using a remote shell session. The installation of plugin is really simple and we will walk through the steps to install/uninstall, enable and verify the plugin installation.

The plugin is available as a docker image on the public docker registry but if you are using a private registry, you will have to point to the appropriate URL of the image.

<div class="well">
Note: We have discontinued the DEB/RPM based installation of the Docker plugin.
</div>

* **To install the plugin**

```
~# docker plugin install --grant-all-permissions --alias vsphere vmware/docker-volume-vsphere:latest
latest: Pulling from vmware/docker-volume-vsphere
f07d34084e57: Download complete
Digest: sha256:e1028b8570f37f374e8987d1a5b418e3c591e2cad155cc3b750e5e5ac643fb31
Status: Downloaded newer image for vmware/docker-volume-vsphere:latest
Installed plugin vmware/docker-volume-vsphere:latest
```

* **To verify that it was installed and is listed**

```
~# docker plugin ls
ID                  NAME                DESCRIPTION                           ENABLED
831fd45343af        vsphere:latest      VMWare vSphere Docker Volume plugin   true
```

* **You can enable or disable the plugin if needed**

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

* **And finally to remove the plugin from a given Docker host:**

```
~# docker plugin rm vsphere
vsphere
~# docker plugin ls
ID                  NAME                DESCRIPTION         ENABLED
```

