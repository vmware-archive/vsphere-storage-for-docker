# vmdkops - ESX-based (python) service to provide basic disk ops to guest

A daemon listens to vSocket from guest  VMs, and executes VMDK create / attach /
delete/ detach commands.

It is controlled by `/etc/init.d/vmdk-opsd`

`make` builds VIB for installation on ESX.

Install with with something like this:
```
  > esxcli software vib install --no-sig-check -v `pwd`/<vib_name>
```

On client side, ../client_plugin/vmdkops is a Go client module

