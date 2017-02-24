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

The issue is fixed in Release 0.11.2

## Upgrade to newer build when you DO NOT have Docker volumes to retain

If you are experimenting with the vSphere Docker Volumes and do not have any data you want to retain,
you can do the following to upgrade an existing installation:

* remove the Docker volumes (all files and directories in `dockvols` folder on your datastore)
* remove the configuration DB  `/etc/vmware/vmdkops/auth-db`
* upgrade ESX to the newer release (0.11.2) per installation instructions on github.com. Only ESX part (VIB) needs upgrading to 0.11.2, you may skip upgrading
vSphere Docker Volume Service RPMs on Docker VMs.
* restart Docker services (e.g. `systemctl restart docker`) to clear up Docker volume cache
(optional)


## Upgrade to newer build when you DO have Docker volumes to retain

We provide a script to migrate data and configuration from Release 0.11 (or 0.10) to 0.11.2.
The script is located [here](https://raw.githubusercontent.com/vmware/docker-volume-vsphere/master/esx_service/cli/vmdkops_post_update.11.1.py).

Upgrade steps are as follows: 
* Install VIB with Release 0.11.2 (or later) on each ESX.
* Download the upgrade script. This step is for Release 0.11.2 only. For 0.12 and later, skip directly to running the script.
 * Download the script from https://raw.githubusercontent.com/vmware/docker-volume-vsphere/master/esx_service/cli/vmdkops_post_update.11.1.py and place it into /usr/lib/vmware/vmdkops/bin/ on each ESX
  * make it executabe on each ESX : `cd /usr/lib/vmware/vmdkops/bin/ ; chmod a+x vmdkops_post_update.11.1.py;`
* on each ESX, run `/usr/lib/vmware/vmdkops/bin/vmdkops_post_update.11.1.py`

To understand which steps the script is taking, please look at the [source](https://github.com/vmware/docker-volume-vsphere/blob/master/esx_service/cli/vmdkops_post_update.11.1.py)

# Contact

As usual, if you have questions please [contact us](https://github.com/vmware/docker-volume-vsphere#contact-us)
