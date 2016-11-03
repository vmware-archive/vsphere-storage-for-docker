# Tenancy

Multi-tenancy is an architecture in which a single instance of a software application serves multiple customers or "tenants." Tenants can be used to provide isolation between independent groups in shared environments, where multiple groups are using some common infrastructure i.e. compute, storage, network, etc. With tenancy, you can achieve isolation of resources of one tenant from other tenants. 

For the vSphere Docker Volume Service, Multi-tenancy is implemented by assigning a Datastore and VMs to a tenant.  A tenant can be granted access to create, delete or mount volumes on a specific datastore. VMs assigned to a tenant can then execute Docker volume APIs on an assigned datastores. Within a datastore multiple tenants can store their Docker volumes. A tenant cannot access volumes created by a different tenant i.e. tenants have their own independent namespace, even if tenants share datastores. VMs cannot be shared between tenants.

Some attributes that define Tenant;

- vSphere Administrator can define group of one or more Docker Host (VM) as
Tenant.
- Docker Host (VM) can be member of one and only one Tenant.
- Docker Host (VM) if not assigned to any Tenant are member of _Default tenant.
- vSphere Administrator can grant tenancy:ant privileges & set resource consumption
limits at granularity of datastore.

## Admin CLI

Tenants can be created and managed via the [Admin CLI](/user-guide/admin-cli/#tenant)

## References

- [Design Spec for tenancy](https://github.com/vmware/docker-volume-vsphere/blob/master/docs/misc/docker-volume-auth-proposal.v1_2.md)
- [Introduction to vSphere Storage](https://pubs.vmware.com/vsphere-60/index.jsp#com.vmware.vsphere.storage.doc/GUID-F602EB17-8D24-400A-9B05-196CEA66464F.html)
