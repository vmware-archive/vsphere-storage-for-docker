# vmdkops - ESX-based (python) service to provide basic disk ops to guest

A daemon listens to vSocket from guest  VMs, and executes VMDK create / attach /
delete/ detach commands

'make' builds VIB for installation on ESX. 
Install with with something like this:
  > esxcli software vib install --no-sig-check -v `pwd`/vmware-esx-vmdkops-1.0.0-0.0.1.vib

On client side, ../src/vmware/vmdkops is a Go client module

NOTE: this is Work In Progress: 
- VIB is not functional yet, to deploy see ../Makefile:deploy 
- Python code is not daemon yet (and no watchdog) so for trying out 
  run in on ESX as 'python /usr/lib/vmware/vmdkops/bin/vmci_srv.py"




