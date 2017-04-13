
# Installation

The installation has two parts – installation of the backend on ESX and installation of Docker plugin on the hosts where you plan to run containers with storage needs.
 
# Install on ESX [![VIB_Download](https://api.bintray.com/packages/vmware/vDVS/VIB/images/download.svg)](https://bintray.com/vmware/vDVS/VIB/_latestVersion)


ESX component for vDVS is available in the form of a [VIB](https://blogs.vmware.com/vsphere/2011/09/whats-in-a-vib.html). VIB stands for vSphere Installation Bundle. At a conceptual level a VIB is somewhat similar to a tarball or ZIP archive in that it is a collection of files packaged into a single archive to facilitate distribution. 

Log into ESXi host and download the [latest release](https://bintray.com/vmware/vDVS/VIB/_latestVersion) of vDVS driver VIB on the ESXi. Assuming that you have downloaded the VIB at /tmp location you can run the below command to install it on ESXi

```
# esxcli software vib install -v /tmp/vmware-esx-vmdkops-0.12.ccfc38f.vib
Installation Result
   Message: Operation finished successfully.
   Reboot Required: false
   VIBs Installed: VMWare_bootbank_esx-vmdkops-service_0.12.ccfc38f-0.0.1
   VIBs Removed:
   VIBs Skipped:
```

# Install on VMs

vDVS plugin can be installed on VMs like any docker plugin installation. You will need docker version **1.13/17.03 or above** on the VM. In a large pool of nodes, you can push the plugin installation to multiple VM through a configuration management tool such as Ansible/Salt or using a remote shell session. The installation of plugin is really simple and we will walk through the steps to install/uninstall, enable and verify the plugin installation. 

The plugin is available as a docker image on the public docker registry but if you are using a private registry, you will have to point to the appropriate URL of the image.

<div class="well">
We have discontinued the DEB/RPM based installation of the vDVS plugin.
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

* **And finally to remove the plugin from a given VM, you can use:
**

```
~# docker plugin rm vsphere
vsphere
~# docker plugin ls
ID                  NAME                DESCRIPTION         ENABLED
```

