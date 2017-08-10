# Interop with vSphere features

The vDVS plugin works well with all vSphere features and deviations if any are as documented below:

**SvMotion/XvMotion**

SvMotion/XvMotion is not supported for VMs that use vDVS volumes.

- SvMotion/XvMotion causes vDVS volumes to be migrated out of the datastore location and hence causing the plugin to lose access to the volume. [#1618](https://github.com/vmware/docker-volume-vsphere/issues/1618)

**Storage DRS**

Storage DRS is not supported for VMs that use vDVS volumes. 

- Volume creation succeeds on a datastore which is in maintenance mode. Datastore maintenance mode is a VC feature and ESX is not aware of if any datastore is put in maintenance mode. [#1651](https://github.com/vmware/docker-volume-vsphere/issues/1651)
- Specifying "Datastore Cluster" name during volume creation is not supported. Datastore clusters (as a part of Storage DRS) is a VC feature and not available on an individual ESX. [#556](https://github.com/vmware/docker-volume-vsphere/issues/556)
- Storage DRS doesn't apply for VMs that have independent disks attached to it.

**Snapshot**

Creating snapshots of a VM which has containers running is not supported.

- If a container is running and a snapshot is taken then if the container exits, the volume is not detached. VC shows an error that the volume is part of a snapshot and hence can't be removed from the VM config. For this reason, snapshotting a VM after starting one or more containers isn't supported as it leaves the volume attached to the VM. [#1649](https://github.com/vmware/docker-volume-vsphere/issues/1649)


