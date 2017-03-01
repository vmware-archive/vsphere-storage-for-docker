# What is vSAN
[vSAN](http://www.vmware.com/products/virtual-san.html) is a shared storage (all ESX nodes in a cluster can access the same vSAN store), hyper-converged (vSAN is integrated into vSphere and runs on the same hardware nodes that vSphere manages), flash optimized storage offering from VMware.
# vSAN Policy
vSAN provides software defined storage and for each storage object it can specify the policy which controls attributes such as Number of failures to be tolerated.

# Mapping Docker volumes to vSAN objects with policy

Using the Admin CLI an IT admin can create the policies that can be consumed by Docker volumes. The Docker admin can then create a volume using the policies available.
```
ESX# vmdkops_admin.py policy create —name=myPolicy —content='(("proportionalCapacity" i0)("hostFailuresToTolerate" i0))'
Successfully created policy: myPolicy
```
```
swarm-1# docker volume create —driver=vmdk —name=MyVolume -o size=2gb -o vsan-policy-name=myPolicy -o fstype=xfs
MyVolume
```
