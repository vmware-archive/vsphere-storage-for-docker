# vDVS Authorization Config DB state machine

vSphere Docker Volume Service (`vDVS`) Authorization Config DB (aka `Config DB`) supports `modes` which is easier to see as states in a state machine. This document is a quick write up of the states and allowed state transitions. It is intended to help understanding and verifying behavior of vmdkops_admin.py (`config` command set for managing Config DB) and auth_data.AuthDataManager methods related to DBMode type.

## States

The following states are tracked:

* `Unknown` - initial state of the AuthDataManager, meaning no attempt to find out the actual state has been made yet. This state is replaced by actual state on the instance construction time
* `NotConfigured` - there is no link for DB in /etc/vmware/vmdkops. No auth config operation is allowed, and the allow_all_access() method will always return True. All requests from vDVS client are auto-approved.
* `SingleNode` - there is an actual Config DB in /etc/vmware/vmdkops. All authorization system is limited to a single ESX node
* `MultiNode` - There is a link in /etc/vmware/vmdkops to a Config DB on a shared datastore. Every ESXi node which has similar link will share the authorization configuration.
* `BrokenLink` - Error state. Either the symlink point to non-existing file, or the DB file is corrupt. All vDVS client requests will be denied in this mode, and vmdk_ops.py error log will indicate the BrokenLink state. The only way to recover is to remove the local configuration and re-init the Config DB again

## State transitions

Since there are very few states and the transitions are rather simple, we will outline them here in plan text and skip diagrams.

Currently, there are 2 components setting state:
`auth_data.AuthenticationDataManager constructor` call (passive setting state for internal usage) and `vmdkops_admin.py admin CLI commands` (active changing of the state). Going forward, we plan to add another component - vmdk_ops.py service `Config DB periodic discovery`, to assure that internal state reflects changes to Config DB made from other ESXi nodes. Discovery is discussed later in this document

### From Unknown

On auth_data.AuthenticationDataManager.connect(), we check the actual DB mode change internal state as follows:

* Unknown-> NotConfigured   use "no access control" mode for vmdk-opsd operations
* Unknown-> SingleNode      open the DB and keep going (backward compat, local DB)
* Unknown-> MultiNode        open the DB and keep going (shared DB via symlink)
* Unknown-> BrokenLink      ERROR - throw exception


### From NotConfigured

Manual transition from NotConfigured is done by the admin CLI.
Automatic transition from NotConfigured is done by Discovery (see below)

* NotConfigured -> NotConfigured    Keep going.
* NotConfigured -> SingleNode       Forced by by creating a new DB in /etc/vmware/vmdkops (AuthenticationDataManager.new_db(). When discovered, we just connect to the DB, log an INFO message and keep going.
* NotConfigured -> MultiNode         Forced by creating the DB on a shared datastore and creating symlink in /etc/vmware/vmdkops. When discovered, we just connect to the DB, log an INFO message and keep going. (same as in NotConfigured->SingleNode)
* NotConfigured -> BrokenLink       ERROR - throw exception

### BrokenLink handling
BrokenLink is an error condition. We do not know what was the admin intent so we fail all Docker volume requests until the situation is resolved by the admin using `config rm --local` command

* BrokenLink -> NotConfigured - by admin `config rm` command
* BrokenLink -> BrokenLink -  noop
* BrokenLink -> SingleNode or MultiNode - no direct transition, the Config DB has to go through NotConfigured first.
  * However, a vDVS service may first find out the state is BrokenLink, then admin does 2 commands and then vDVS service may directly transfer it's internal state to MultiNode or SingleNode


### From SingleNode

Transition from SingleNode means the admin live-reconfigured the Config DB.
On the node where the operation happened, the admin CLI will refresh the vDVS service which will find out the new state from the DB.

On other nodes, the discovery will find out about the change to `MultiNode` only, and will issue a warning to the admin  - since we do not know what was the intent.

* SingleNode    -> Single Node      noop
* SingleNode    -> NotConfigured    happened on `config -rm --local` only. Restart the service.
* SingeNode     -> MultiNode        can happen on discovery. WARNING - Config changed. No action  we do not know what was the intent.
* SingeNode     -> BrokenLink       ERROR - Throw exception

### From MultiNode

To transition from MultiNode, either the admin has to issue `config --rm` or there should be a FS or DB issue (i.e. the symlink was accidentally removed)

* MultiNode      -> MultiNode       noop
* MultiNode      -> NotConfigured   restart the service (on discovery or on CLI)
* MultiNode      -> SingleNode      same
* MultiNode      -> BrokenLink      ERROR - Throw exception

## Discovery

Discovery goal is to sync `vmdk_ops.py` Config DB state to reflect changes to the state done on a different ESXi node.

**The key use case** is initial configuration, where all nodes start with NotConfigured. On one of the nodes, admin (manually or via automation) configures a shared Config DB on a datastore visible to all nodes. Then discovery kicks in on all nodes, finds out there is a fresh Config DB on a shared store, and configures the node to use this DB. This way Admin only need to do configuration on one node and the rest pick it up . See Issue https://github.com/vmware/docker-volume-vsphere/issues/1086 for details.

The only state transition supported by Discovery is NotConfigured -> MultiNode.
*  Note - once defined, the DB becomes a single point of failure, and the only way to remove it is to run local CLI `config rm --local` on each ESX.
  * To clarify `config rm --local` removes al `local knowledge` about Config DB - be it the symlink or the actual DB.
* The Config DB will contain a list of ESX nodes allowed to participate in the discovery (FQDN, FQDN regexp, IPv4 IP or regexp).
  * Default fot the list is '*' (all allowed).
* The service will be periodically checking for new DB on shared datastores, and if 1 (and only one) shows up there, will check for current ESX being allowed, and then change local config to use shared DB .
  * it will also register ESX node in "nodes using the DB" table for audit and troubleshooting
* when a shared DB is discovered, the service will perform `config init --datastore=DS` operation which will in turn refresh the service to use the new configuration. This will switch the service Config DB mode to MultiNode.


*** END OF DOCUMENT **

