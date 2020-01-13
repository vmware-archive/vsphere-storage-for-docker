---
title: Installation
---

The installation has two parts – installation of the vSphere Installation Bundle (VIB) on ESXi and installation of Docker plugin on the hosts where you plan to run containers with storage needs.

## Installation on ESXi

[VIB](http://blogs.vmware.com/vsphere/2011/09/whats-in-a-vib.html) and [Offline Depot](https://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html?resultof=%2522%256f%2566%2566%256c%2569%256e%2565%2522%2520%2522%256f%2566%2566%256c%2569%256e%2522%2520%2522%2564%2565%2570%256f%2574%2522%2520) are the packages built to install the backend for the service on ESX. The backend can be installed using esxcli or vmware tools such as [VUM](http://pubs.vmware.com/vsphere-60/topic/com.vmware.ICbase/PDF/vsphere-update-manager-601-install-administration-guide.pdf)

### VUM based VIB installation
ESXi component for VDVS is available in the form of an [Offline Depot](http://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.install.doc/GUID-29491174-238E-4708-A78F-8FE95156D6A3.html) and available to download latest release as [ZIP archive](https://bintray.com/vmware/vDVS/VIB/_latestVersion).

After downloading VIB bundle as zip format, installation is performed using VUM using the following steps.

1. Go to *admin view* in update manager tab.
2. Import zip bundle using *import zip* option.
3. Create a baseline and include zip bundle.
4. Scan your host against baseline.
5. *Remediate* host.

More information can be found at [VUM User guide](https://featurewalkthrough.vmware.com/#!/vsphere-6-5/vsphere-update-manager-overview-cluster-upgrade/1).

### VIB installation through esxcli/localcli

ESXi component for VDVS is available in the form of a [VIB](https://blogs.vmware.com/vsphere/2011/09/whats-in-a-vib.html). VIB stands for vSphere Installation Bundle. At a conceptual level a VIB is similar to a tarball or ZIP archive in that it is a collection of files packaged into a single archive to facilitate distribution.

Log into ESXi host and download the [latest release](https://bintray.com/vmware/vDVS/VIB/_latestVersion) of VDVS driver VIB on the ESXi. Assuming that you have downloaded the VIB at /tmp location you can run the below command to install it on ESXi. You will need ESXi version **6.0 or above**

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

## Installation on Linux Docker Hosts

VDVS plugin can be installed on Linux Docker hosts like any docker plugin installation. You will need docker version **17.06.1 or above** on the VM. In a large pool of nodes, you can push the plugin installation to multiple VM through a configuration management tool such as Ansible/Salt or using a remote shell session. The installation of plugin is really simple and we will walk through the steps to install/uninstall, enable and verify the plugin installation.

### Dependencies

The plugin uses VMCI (Virtual Machine Communication Interface) and vSockets to contact the service on ESX. The associated Linux kernel drivers are installed via the VMware Tools and its open version, namely [open-vm-tools](https://github.com/vmware/open-vm-tools), packages. Either one of these packages must be installed in the guest OS. It's recommended to install the most up-to-date version of either of these packages as available.

### Installation instruction

The plugin is available as a docker image on the public docker registry but if you are using a private registry, you will have to point to the appropriate URL of the image.

<div class="well">
Note: We have discontinued the DEB/RPM based installation of the Docker plugin.
</div>

* **To install the plugin**

```
~# docker plugin install --grant-all-permissions --alias vsphere vmware/vsphere-storage-for-docker:latest
latest: Pulling from vmware/vsphere-storage-for-docker
f07d34084e57: Download complete
Digest: sha256:e1028b8570f37f374e8987d1a5b418e3c591e2cad155cc3b750e5e5ac643fb31
Status: Downloaded newer image for vmware/vsphere-storage-for-docker:latest
Installed plugin vmware/vsphere-storage-for-docker:latest
```

On some linux distributions, such as CoreOS the installation might fail with an error:

```
Error response from daemon: dial unix /run/docker/plugins/daa3290986386bde82fa486a79578c0511522a03fceaf1a50cb7c7451a254b5d/vsphere.sock: connect: no such file or directory
```

To work around this, you need to pass an additional configuration option to the install command, the GID of the group that owns the docker socket.

```
docker plugin install --grant-all-permissions --alias vsphere vmware/vsphere-storage-for-docker:latest "VDVS_SOCKET_GID=$(getent group $(stat -c "%G" /var/run/docker.sock) | cut -f3 -d":")"
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

## Installation on Windows Docker Hosts

vSphere Storage for Docker can be installed on Windows Server 2016/Windows 10 VMs using the PowerShell installer. You will need Docker EE/Docker for Windows version **17.06 or above** (with Windows containers mode enabled) on the VM.

First, start an instance of PowerShell with the "Run as administrator" option.

* **To download the plugin installer**

```
PS C:\> Invoke-WebRequest -Uri https://raw.githubusercontent.com/vmware/vsphere-storage-for-docker/master/install-vdvs.ps1 -Out install-vdvs.ps1
```

* **To install the plugin**

```
PS C:\> .\install-vdvs.ps1 <windows_plugin_binary_url>
```

For example, the vSphere Storage for Docker 0.16 developer preview binary could be installed as shown below.

```
PS C:\> .\install-vdvs.ps1 https://bintray.com/vmware/vDVS/download_file?file_path=docker-volume-vsphere_windows_developer-preview.zip
Downloading from https://bintray.com/vmware/vDVS/download_file?file_path=docker-volume-vsphere_windows_developer-preview.zip...
Extracting vsphere-storage-for-docker.zip into C:\Program Files\VMware\vmdkops...
Deleting vsphere-storage-for-docker.zip...
Installing Windows service vdvs from C:\Program Files\VMware\vmdkops\vdvs.exe...

Status   Name               DisplayName
------   ----               -----------
Stopped  vdvs               vSphere Docker Volume Service
Starting Windows service vdvs...
Running  vdvs               vSphere Docker Volume Service
Windows service vdvs installed successfully!
```

* **To verify that the plugin was installed**

```
PS C:\> Get-Service vdvs

Status   Name               DisplayName
------   ----               -----------
Running  vdvs               vSphere Docker Volume Service
```

* **To uninstall the plugin**

```
PS C:\> .\install-vdvs.ps1 -Uninstall
Do you really want to uninstall vdvs [Y/N]?: Y
Stopping Windows service vdvs...
Deleting Windows service vdvs...
[SC] DeleteService SUCCESS
Deleting C:\Program Files\VMware\vmdkops...
Windows service vdvs uninstalled successfully!
```

**Note:** In case of a failure due to an UnauthorizedAccess error, please unblock the script using the following command and retry.

```
PS C:\> Set-ExecutionPolicy Unrestricted -Scope Process -Force
```

## Upgrade Instructions
Upgrade of volume managed plugin or ESX driver has no impact on data path and running containers. It is recommended to have both of them from same release. However, it is not mandatory and both of them are backward compatible upto one prior release version.


**ESX Driver upgrade:**

```
#1. Uninstall VIB (driver)
esxcli software vib remove -n esx-vmdkops-service

#2. Download and install the latest VIB from the vsphere-storage-for-docker release page
esxcli software vib install -v /<vib_name>.vib

#3. Verify the status
esxcli storage guestvol status
```

**Plugin upgrade:**
**Note**: If you have used a different alias for existing plugin installation, use that instead of ```vsphere``` in the steps above

```
#1. Disable the managed plugin from the docker host
docker plugin disable -f vsphere

#2. Upgrade the latest release of vsphere-storage-for-docker managed plugin
docker plugin upgrade --grant-all-permissions vsphere:latest vmware/vsphere-storage-for-docker:latest

#3. Enable the managed plugin from the docker host
docker plugin enable vsphere
```

## Downgrade to a previous version
**ESX Driver downgrade:**

```
#1. Uninstall VIB (driver)
esxcli software vib remove -n esx-vmdkops-service

#2. Download and install the required VIB from the vsphere-storage-for-docker release page
esxcli software vib install -v /<vib_name>.vib

#3. Verify the status
esxcli storage guestvol status

```

**Plugin downgrade:**

```
#1. Disable the managed plugin from the docker host
docker plugin disable -f vsphere

#2. Using docker plugin upgrade, you can downgrade to required release of vsphere-storage-for-docker managed plugin
docker plugin upgrade --grant-all-permissions vsphere:latest vmware/vsphere-storage-for-docker:<required_release_version>

#3. Enable the managed plugin from the docker host
docker plugin enable vsphere
```

https://gitlab.com/Godexist/iqozzz-.git

