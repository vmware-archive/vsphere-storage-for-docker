Change Log

<table style="width:92%;">
<colgroup>
<col width="15%" />
<col width="13%" />
<col width="12%" />
<col width="52%" />
</colgroup>
<thead>
<tr class="header">
<th><p>Version</p></th>
<th><p>Date</p></th>
<th><p>Who</p></th>
<th><p>Change</p></th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><p>1.0</p></td>
<td><p>8/17/16</p></td>
<td><p>Msterin</p></td>
<td><p>Walked over the doc with AndrewStone and added work items at the end</p></td>
</tr>
<tr class="even">
<td><p>1.1</p></td>
<td><p>8/18/16</p></td>
<td><p>AStone</p></td>
<td><p>Conversion to Markdown and DB Usage updates</p></td>
</tr>
<tr class="odd">
<td><p>1.2</p></td>
<td><p>8/29/16</p></td>
<td><p>AStone</p></td>
<td><p>Updates to remove roles altogether and simplify auth data model based on meetings from 8/26/16</p></td>
</tr>
</tbody>
</table>

Introduction
============

Currently the vSphere Docker Volume Service allows any container on any VM with access to any datastore to create, mount, and delete volumes at will. There are no limits to the size of the volumes that can be created or quotas for datastore usage for container volumes. Also, all Docker engines can see and use any volume on any datastore. This lack of access control takes the management of storage out of the hands of the IT administrator and puts it in the hands of the developer. This is a worrying proposition to many IT admins, and since they are the ones that must allow deployment of the service and its associated host agent, we must allow them to regain control with proper enforcement mechanisms that limit damage due to malfeasance or mistake.

In order to further ease the management burden on vSphere administrators, we want to provide them with a way to manage permissions across hosts, datastores, and VMs from a single central location. We also want to maintain existing workflows and embrace existing solutions to both ease development and enhance the comfort of the user. The standard way to centrally manage resources and permissions in the vSphere universe is via vCenter. Our goal then is to provide a solution that enables configuration of access controls via vCenter, while maintaining the flexibility to allow granular enough permissions to satisfy container use cases on vSphere datastores.

Background - existing (vSphere and Docker) permissions
======================================================

vSphere permissions
-------------------

VSphere authorization is based around users, groups, roles, objects and permissions. As described in the [vSphere Security Guide](https://pubs.vmware.com/vsphere-60/topic/com.vmware.ICbase/PDF/vsphere-esxi-vcenter-server-60-security-guide.pdf) chapter 4:

> The permission model for vCenter Server systems relies on assigning permissions to objects in the object hierarchy of that vCenter Server. Each permission gives one user or group a set of privileges, that is, a role, for a selected object. For example, you can select an ESXi host and assign a role to a group of users to give those users the corresponding privileges on that host.

These privileges are predefined, but new ones can be defined **via vCenter extensions (solutions).** All predefined privileges are listed in Chapter 10 of the vSphere Security Guide. While there are permissions on datastores defined, they are not granular or flexible enough to satisfy our needs for access control involving container workflows. In order to satisfy our requirements we must do one of the following:

1.  Add a Datastore.Volume object level to allow more granular permissions, such as limiting the size of new volumes or making them visible only to certain users, by editing the Authz/VC code
2.  Build a vCenter extension that does the same as 1
3.  Forego vCenter permissions altogether and implement an alternative authorization solution

Docker users and permissions
----------------------------

The Docker engine is usable from any user account with privileges to run Docker commands. Docker *requires root privileges* to run Docker commands, which can be enabled by adding users to the ‘docker’ group. Therefore there are no restrictions on Docker usage between users of Docker.

Furthermore, the Docker volume plugin API does not pass any user information from Docker to volume drivers. The sources of Docker volume requests from the guest are indistinguishable from one another. Additionally, the Docker client may be on a separate physical machine from the Docker server, and therefore the Docker server machine may not even know which user is running Docker commands. This means that guest user based access control for vSphere Docker Volume Service permissions is infeasible without significant changes to Docker core code, specifically Plugin API and remote access control code.

Required permissions and caveats
================================

From internal discussions and discussion with customers, it has become apparent that a certain base level of access control is required for production usage of a Docker volume plugin. Volume size, datastore placement, visibility, and operations such as create, mount, and delete must be limited to certain users or groups of users. However, due to the fact that Docker does not provide for user disambiguation we must come up with a way to authorize and grant privileges in some other manner.

We have chosen to authorize access and grant privileges based on the guest VM running the command. Every request that comes from the plugin and ends up on the host agent is associated with a VM running on that host. This is [unforgeable](http://pubs.vmware.com/vsphere-60/topic/com.vmware.ICbase/PDF/ws9_esx60_vmci_sockets.pdf) and allows for access control using VM as the identity. We can associate privileges with each VM or group of VMs and provide responses from the host agent to the volume driver based on those privileges. This allows the full range of required permissions to be satisfied.

Note that since vSphere authorization is based around users, and Docker volume driver authorization is based around VM identities, we are already outside the realm of what’s possible using standard methods. While we could add privileges at finer levels and add a whole new hierarchy of permissions for tenants and visibility of volumes, there is still no built-in vSphere method to enforce those permissions using VM identities. In order to minimize coupling to vSphere release cycles, and reduce implementation complexity, privileges granted to VMs for Docker volume management will be implemented in a different manner to be described in the implementation section.

Permissions Model
=================

As mentioned above, permissions for Docker volume service usage are granted to VMs or groups of VMs. A group of VMs is known as a `Tenant`. Tenants provide for full isolation of volumes such that volumes created by one tenant are neither visible nor usable from other tenants, even if those volumes reside on datastores shared by tenants. An exception to this rule is if the `global` `visibility` privilege is granted to a tenant for a datastore. This allows the tenant to see all volumes in all tenants on the datastore.

Privileges consist of operations against a datastore and limits on those operations. Privileges are assigned per datastore, and each tenant may have different privileges for different datastores.

To illustrate the above, here is a simple **Use Case**:

-   There are 2 product teams which need 2 docker environments for each team - Dev and Test.
-   They request it from Vsphere Admin.
    -   The Admin creates 4 tenants (Product1Dec, Product1Test, Product2Dev, Product2Test) and sets storage quotas
    -   The admin then create a bunch (say 16 :-) VMs and assigns them to specific Tenants.
-   The admin gives teams dedicated endpoints (VMs IPs)
-   Teams now manage user accounts themselves, but the VMs have only access to storage allocated to them
    -   Also, Docker volumes are now setting in separate namespaces, so "volume1" for Product1Test is a different volume from "volume1" for Product2Test

### Privileges and limits

The following privileges and limits can be granted to tenants for specific datastores.

-   \[ this one is postponed\] *Privilege*: Global Visibility – Tenant can see all volumes created by all tenants
-   *Privilege*: Create volumes
    -   *Limit*: Maximum Size of a volume, in MB
    -   *Limit*: Maximum storage provisioned on a datastore to all volumes for the tenant, in MB.
-   *Privilege*: Remove a volume
-   *Privilege*: Mount a volume

#### Example

The following example is illustrated via CLI commands from a hypothetical program called `auth_config`. `auth_config` is not tied to any specific implementation but is expected to display the workflow of managing authorization configuration.

1.  **Create a tenant named `tenant1` consisting of 3 VMs**
    `auth_config` `tenant` `create` `tenant1` `--vms` `vm1,vm2,vm3`

2.  **Assigning create, mount and delete privileges to a datastore for a tenant**
    `auth_config` `tenant` `set` `privileges` `--tenant` `tenant1` `--datastore` `datastore1` `--privileges` `create,mount,delete`

3.  **Commit the configuration**

`auth_config` `commit`

</ol>
### Defaults

By default, VMs are assigned to a default tenant and are granted unlimited privileges to all datastores visible from the host on which they reside. The privileges on the default tenant cannot be modified or restricted. VMs can see all volumes ever created on these datastores by other VMs that are or were part of the default tenant, and can mount and delete these volumes. However, volumes created by VMs on a tenant other than the default tenant are not visible to VMs in the default tenant. Note that a VM can only be a member of a single tenant at a time.

The goal of open defaults on the default tenant is to allow evaluation usage of the Docker volume driver without the user having to worry about configuration. However, in a production scenario, *all VMs* should be assigned to non-default tenants. Even though volumes in other tenants are not visible to VMs in the default tenant, and thus inaccessible to them, VMs in the default tenant can fill production datastores and make creation of volumes from other tenants impossible.

Implementation
==============

Since authorization is based on VM identity and visibility of volumes is tied to tenants, a new strategy outside the standard vSphere authorization model is required. While the privileges and identities used for authorization differ from the vSphere model, it is still desirable to have them managed from within vCenter, as this is the standard place for centralized configuration in all current vSphere deployments. Both of the preceding statements lead to the following implementation design:

-   A vCenter plugin will be created to allow centralized management of tenants, privileges, datastores and VMs for the Docker volume plugin.
-   A new host based authorization mechanism will enable access control at a tenant level.
-   New VMODL interface definitions for defining tenants, roles and privileges for the Docker volume plugin will be created.
-   All configuration for host based authorization will be via VMOMI using the new VMODL definitions.
-   VMOMI SOAP endpoints on ESXi hosts for the authorization definitions will update per-datastore SQLite databases on configuration changes.

VCenter plugin
==============

In order to provide a user interface enabling centralized configuration without changing the core vCenter code, a vCenter plugin will be implemented. As our team is not flush with UI experts, the layout described here will be a rough overview consisting of the necessary behaviors and operations. The specific layout and views will likely be different in the actual implementation.

### Overall layout

The overall suggested layout of the plugin is that of a single page web-app. This page will exist as a tab in vCenter. All operations required for managing roles and permissions of the Docker volume plugin within a datacenter will be provided in this tab.

### VMs as Docker hosts

It should be possible to show all VMs in the datacenter in the plugin. There should be some mechanism (checkboxes, drag and drop etc..) to enable that VM as a Docker host. Once a VM is enabled as a Docker host it is capable of being assigned to a tenant.

### Tenants, VMs, Datastores

All Docker host VMs are assigned to tenants. A Docker host VM is a member of exactly one tenant. Tenants should be created in a dialog box or other UI element. This dialog should allow applying privileges to datastores for the given tenant. Privileges can be changed at any time after creation.

All updates to a running configuration should be changed via a big fat red COMMIT button.

### Web Server

Since we are extending VMODL, all configuration requests from the web client will go through VPXD. This obviates the need for a specific webserver running on the VCenter host.

### Authorization Databases

In order to perform authorization, the `vmdk_ops` agent on each ESX host must be able to verify access rights for Docker volumes for a requesting VM. Each datastore maintains a SQLite database that manages all access control information to facilitate this authorization. These databases get populated from VMODL update handlers running on ESX hosts.

`vmdk_ops` maintains a persistant connection to the SQLite database on each Datastore it has access to. On each Docker operation against a datastore, the `vmdk_ops` agent will check the requesting VM’s permissions for this operation on the datastore in question. It does this by first checking which tenant the VM belongs to, and then checking the tenants’ rights on that datastore.

### Database Schema

Databases are per-datastore. There are 3 tables provided in the authorization database: tenants, vms, and privileges. Their schemas are provided below using SQLite syntax. To enhance query speed, indexes and views can be created as necessary.

### Tenants Schema

Each tenant has an immutable id, a name, and a description.

``` sql
CREATE TABLE tenants(
  id TEXT PRIMARY KEY NOT NULL,
  name TEXT,
  description TEXT,
);
```

### VMs Schema

Each VM maps to a single tenant.

``` sql
CREATE TABLE vms(
  vm_id INTEGER PRIMARY KEY NOT NULL,
  vm_name TEXT,
  tenant_id TEXT NOT NULL,
  FOREIGN KEY(tenant_id) REFERENCES tenants(id)
);
```

### Privileges Schema

Privileges are unique to a (tenant, datastore) pair.

``` sql
CREATE TABLE privileges (
  tenant_id TEXT NOT NULL,
  datastore TEXT NOT NULL,
  create INTEGER,
  delete INTEGER,
  mount INTEGER,
  max_volume_size INTEGER,
  max_usage INTEGER,
  PRIMARY KEY (tenant_id, datastore),
  FOREIGN KEY(tenant_id) REFERENCES tenants(id)
);
```

Volume Storage
==============

Each tenant will have it’s own directory under /vmfs/volumes/<DATASTORE>/dockvols. All volumes created by that tenant will live in the tenant’s directory. Tenants without global visibility permissions are only allowed to list volumes in that directory. Tenants with global visibility permissions can list all volumes in all tenant directories. Due to the fact that volumes in different tenants can have the same names, the tenant name must be prepended to the volume name when access is requested outside the tenant. For example, if tenant `A` wants to create a volume named `NewVolume` in tenant `B` on `datastore1`, the Docker volume command would be `docker` `volume` `create` `B/NewVolume@datastore1`. However, if tenant `A` only wanted to create that volume on it’s own tenant, it could just do `docker` `volume` `create` `NewVolume@datastore1`. If `datastore1` was the default datastore for the VM then it could be excluded from the command as well, like: `docker` `volume` `create` `NewVolume`.

Volumes listed from tenants with global visibility would appear like this, where `A` and `B` are tenant names:

    A/Vol1@datastore1
    B/Vol1@datastore1
    B/Vol1@datastore2

Volumes listed from tenants without global visibility would not show the tenant name:

    Vol1@datastore1
    Vol2@datastore1
    Vol1@datastore2

NOTE: global visibility support is postponed (9/26/2016)

