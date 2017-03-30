* IMPORTANT: This is an experimental feature supported only on standalone ESX as of release 0.12. We are seeking feedback/requirements to support mult-tenancy across vSphere Cluster/Datacenter. You can drop us a note at containers@vmware.com.

# Tenancy

Multi-tenancy is an architecture in which a single instance of a software application serves multiple customers or "tenants." Tenants can be used to provide isolation between independent groups in shared environments, where multiple groups are using some common infrastructure i.e. compute, storage, network, etc. With tenancy, you can achieve isolation of resources of one tenant from other tenants.

For the vSphere Docker Volume Service, Multi-tenancy is implemented by assigning a Datastore and VMs to a vmgroup.  A vmgroup can be granted access to create, delete or mount volumes on a specific datastore. VMs assigned to a vmgroup can then execute Docker volume APIs on an assigned datastores. Within a datastore multiple vmgroups can store their Docker volumes. A vmgroup cannot access volumes created by a different vmgroup i.e. vmgroups have their own independent namespace, even if vmgroups share datastores. VMs cannot be shared between vmgroups.

Key attributes of tenancy:

- vSphere Administrator can define group of one or more Docker Host (VM) as
vmgroup
- Docker Host (VM) can be a member of one and only one vmgroup.
- vSphere Administrator can grant vmgroup privileges & set resource consumption
- Vmgroups can share the same underlying storage but preserve volume namespace isolation.
limits at granularity of datastore.

## Admin CLI

Vmgroups can be created and managed via the [Admin CLI](/user-guide/admin-cli/#Vmgroup)

## Default vmgroup
When a VM which does not belong to any vmgroup issues a request to vmdk_ops, this VM will be assumed to be in _DEFAULT vmgroup, and will get privileges
associated with this vmgroup. \_DEFAULT vmgroup will be automatically created by system post install, so by default vmdk_ops will support request from
any VM , thus maintaining backward compatibility and simplicity of installation.An admin can remove this vmgroup or modify privileges, thus locking
down vmdk_ops to serve only explicitly configured VMs.

## Default privileges
When a VM tries to manipulate volumes on a datastore which is not configured,  the system will use _DEFAULT privilege, which is created automatically
by system post install. This _DEFAULT privilege allows access to ANY datastore by default. An admin can edit or remove this record, thus locking down
the functionality to allow access only to explicitly configured datastores.

## Default datastore
When a VM addresses the volume using short notation (volume_name, without @datastore), all VMs in this vmgroup will use default datastore to resolve short volume reference (volume_name will actually mean volume_name@default_datastore).
If "default_datastore" is not set for a vmgroup, then datastore where the VM resides will be used as "default_datastore".

## References

- [Design Spec for tenancy](https://github.com/vmware/docker-volume-vsphere/blob/master/docs/misc/docker-volume-auth-proposal.v1_2.md)
