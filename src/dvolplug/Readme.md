# dvolplug - Docker VMDK volume mgr / plugin

This is a docker-side code to provide Docker Volume API for VMWARE VMDKs.

It communicates to the host (esx) service over vSocket, and requests
 create/delete/attach/detach VMDK operations

The over-vSocket API is JSON RPC.

The plugin depends on vmdkops Go module and vmdkops-esxsrv pyhton service.
(more doc TBD here)
