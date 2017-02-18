# Update from Release 0.11

# Issue
Release 0.11 has a regression introduced by Tenant feature.

This regression impacts visibility of Docker volumes created by a Docker VM running on another ESX node
and located on a shared Datastore.

## Example :
* Docker VM1 runs on ESX1 and uses Datastore1.
* Docker VM2 runs on ESX2 and uses Datastore2.
* Both Datastore1 and Datastore2 are accessible from ESX1 and ESX2.

Prior to Release 0.11, both VM1 and VM2 saw Docker volumes created by another VM.
In 0.11 they do not see each other's Docker volumes.

# Upgrade

The issue is fixed in Release 0.11.1

## Upgrade to newer build when you DO NOT have Docker volumes to retain

If you are experimenting with the vSphere Docker Volumes and do not have any data you want to retain,
you can do the following to upgrade an existing installation:

* remove the Docker volumes (all files and directories in `dockvols` folder on your datastore)
* remove the configuration DB  `/etc/vmware/vmdkops/auth-db`
* upgrade ESX to the newer release (0.11.1) per installation instructions on github.com. Only ESX part (VIB) needs upgrading to 0.11.1, you may skip upgrading
vSphere Docker Volume Service RPMs on Docker VMs.
* restart Docker services (e.g. `systemctl restart docker`) to clear up Docker volume cache
(optional)


## Upgrade to newer build when you DO have Docker volumes to retain

We are working on a script to upgrade automatically, and plan to release it in the week of 2/20.


# Contact

As usual, if you have questions please [contact us](https://github.com/vmware/docker-volume-vsphere#contact-us)
