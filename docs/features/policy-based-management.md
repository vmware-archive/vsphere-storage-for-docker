###...

# Storage Policy Based Management
[vSAN](http://www.vmware.com/products/virtual-san.html) is a shared and hyper-converged flash optimized storage offering from VMware. All ESX nodes in a cluster can access the same vSAN store. vSAN is integrated with the vSphere and runs on the same hardware that vSphere manages.

# vSAN Policy
vSAN provides a framework for software defined storage. You can create policies based on different use cases and access needs and apply them to storage units. For example a simple policy shown in code snippet below describes a policy of zero failure to tolerate.

```
(
 ("proportionalCapacity" i0)
 ("hostFailuresToTolerate" i0)
)
```
## Policy Creation

### Create policy

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy create --name some-policy --content '(("proportionalCapacity" i0)("hostFailuresToTolerate" i0))'
Successfully created policy: some-policy
```

### List Policy

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy ls
Policy Name  Policy Content                                             Active
-----------  ----------------------------------------------------------  ------
some-policy  (("proportionalCapacity" i0)("hostFailuresToTolerate" i0))  Unused
```

### Update a policy

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy update --name some-policy --content '(("proportionalCapacity" i1))'
This operation may take a while. Please be patient.
Successfully updated policy: some-policy
```

### Remove a policy

```
[root@localhost:~] /usr/lib/vmware/vmdkops/bin/vmdkops_admin.py policy rm --name=some-policy
Successfully removed policy: some-policy
```

## Policy Usage

### Attach policy to container
The Docker admin can then create a volume using the policies available.

```
ESX# vmdkops_admin.py policy create —name=myPolicy —content='(("proportionalCapacity" i0)("hostFailuresToTolerate" i0))'
Successfully created policy: myPolicy
```
```
swarm-1# docker volume create —driver=vmdk —name=MyVolume -o size=2gb -o vsan-policy-name=myPolicy -o fstype=xfs
MyVolume
```
