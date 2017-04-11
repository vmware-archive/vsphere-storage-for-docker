### Introduction

Currently the Docker volume driver for VSphere allows any container on any VM with access to a given datastore
to create, mount, and delete volumes at will. There are no limits to the size of the volumes that
can be created or quotas for datastore usage for container volumes. This lack of access control
takes the management of storage out of the hands of the IT administrator and puts it in the hands of
the developer. This is a worrying proposition to many IT admins, and since they are the ones that
must allow deployment of the driver and it's associated host agent, we must allow them to regain
control with proper enforcement mechanisms that limit damage due to malfeasance or mistake.

### Permissions Model

Since docker containers run inside VMs, and can have root permissions, we cannot rely on the VM user
for access control. Permissions must be granted to a VM, since VMs are created and managed
by the IT administrator. This brings up a few issues:

  1. How are VMs identified in order to grant permissions?
  2. What is the granularity of a permission?
  3. Where are permissions stored?
  4. What are the actual permissions?

In order to answer these questions, we must consider 2 scenarios. The first scenarios is permissions
for a single VM with a local datastore, and the second is multiple VMs with a shared datastore. The
model must work for both scenarios in order to be useful.

1. VMs can share names so must be identified by unique IDs. These IDs should be unique across a
   cluster of hosts. If desired, we can associate many IDs with a Role that can be used to grant
   permissions. In this case permissions granted to a role apply to all VMs that have that role
   applied.
2. Permissions are managed per datastore. This means that VMs can have different privileges on
   different datastores. For instance, on one datastore a VM may be allowed to create volumes of a
   given size, while on another datastore the VM cannot create volumes at all. Note that in order to
   provide out of the box usability and experimentation, all datastores allow all operations with no
   restrictions. Permissions must be added per VM per Datastore by an administrator as necessary.
3. Since there is no useful centralized storage outside of VC for storing permissions, and we don't
   want to rely on VC, VM permissions for a given datastore shall reside inside the datastore. There
   must be a way to manage concurrency control and eliminate data races, so using a plain old file
   is probably not the solution. A preliminary idea is to store this data in a database such as
   Postgres or SQlite which already provides these capabilities. The downside of this is that now,
   we need to manage (possibly many) DB instances as part of our solution.
4. The list below shows available privileges per VM and any further restrictions allowed for each one.

  * Create a volume on a given datastore
    * Maximum size of volume
    * Maximum numbers of volumes
  * Remove a volume on a given datastore
    * Remove any volume
    * Remove only volumes created by this VM
  * Mount a volume on a given datastore
    * Mount any volume
    * Mount only volumes created by this VM

Additionally we can constrain each datastore with:
  * A total size for all volumes created
  * A total number of volumes created

### UI

In order to quickly implement a usable permissions system, we will re-use existing tools. The
current way that administrators manage volumes is via the admin CLI on each ESX host. This CLI will
be enhanced with permissions management and will be able to grant or revoke permissions for any VM
and datastore visible to it.

In a later stage, in order to simplify centralized management, an API providing the same
functionality as the CLI will be implemented for each host agent. This api can then be used to
create a VC plugin that allows one click provisioning of Swarm and Kubernetes clusters with proper
permissions configured.
