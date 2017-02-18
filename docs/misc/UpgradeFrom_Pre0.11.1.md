# Update from Release 0.11

 <work in progress>

# Isssue
Release 0.11 has a regression introduced by Tenant feature.
As a result of this regression 2 Docker VMs running on different ESXs
will not see each other's Docker volumes.

# Workaround

If you are experimenting with the Vsphere Docker volumes and do not have any data you want to retain,
you can

*  remove the volumes (all in `dockvols` folder on your datastore)
*  remove the configuration DB `/etc/vmware/auth-db`
*  upgrade to the newer release (0.11.1). Only ESX part (VIB) needs upgrading




# Upgrade script

We are working on a script to upgrade automatically