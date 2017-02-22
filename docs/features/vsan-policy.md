# What is VSAN
[VSAN](http://www.vmware.com/products/virtual-san.html) is a shared storage (all ESX nodes in a cluster can access the same VSAN store), hyper-converged (VSAN is integrated into vSphere and runs on the same hardware nodes that vSphere manages), flash optimized storage offering from VMware.
# VSAN Policy
VSAN provides software defined storage and for each storage object it can specify the policy which controls attributes such as Number of failures to be tolerated.

# Mapping Docker volumes to VSAN objects with policy

Using the Admin CLI an IT admin can create the policies that can be consumed by Docker volumes. The Docker admin can then create a volume using the policies available.
```
vmdkops-admin policy create --name=myPolicy --content="string"
docker volume create --driver=vsphere --name=MyVol -o vsan-policy-name=myPolicy
```
