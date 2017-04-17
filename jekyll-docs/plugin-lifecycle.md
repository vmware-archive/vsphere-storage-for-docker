# Docker Volume Plugin Lifecycle.

This document is designed to be a comprehensive document capturing the plugin lifecycle and its implications to volume mangement.
This is meant to help customers and other developers understand the rationale for the various decisions made by the plugin.

# Plugin Lifecycle and impact on volume refcount.

The docker volume plugin is not actively managed by docker. The lifcycle of the plugin exists independent of the lifecycle of docker. There is a requirement within the plugin to track and remember the number of times docker has mounted a volume if the plugin wishes to free resources for a volume when no containers are consuming a volume. One such resource managed by the plugin is the number of volumes attached to a VM (more details below).

A plugin's lifecycle includes.

1. Install
2. Uninstall
3. Stop
4. Start
5. Recover from a VM crash

## Plugin lifecycle and expected behavior

[Docker recommends that the docker engine should be started after the plugin] (https://docs.docker.com/engine/extend/plugin_api/)

Note: Upgrade of the plugin is done as an uninstall of the older version and an install of the newer version.

1. Install: If it is the first install Docker can be running while the plugin is installed. During installs that are part of an upgrade, the admin must insure that docker is stopped and is started after the plugin starts up.
2. Uninstall: During uninstall docker should be stopped. Before uninstall, make sure no containers still use the VMDK volumes, stop docker engine, and then uninstall the plugin. Docker will generate timeouts on docker start if there are still VMDK volumes after the plugin is uninstalled.
3. Stop: Docker should be stopped before stopping the plugin. [Docker's documentation] (https://docs.docker.com/engine/extend/plugin_api/)
4. Start: The plugin must start before docker. [Docker's documentation](https://docs.docker.com/engine/extend/plugin_api/)
5. Crash: The init system will start the plugin before docker. The plugin will gracefully clean up volumes that are no longer referenced from any container. 

## Ref Counting.

The following are related to the internal details for the plugin. This is not customer facing.

Ref Counting for the volume plugin is needed to know when it is safe to umount and detach a volume.

Volume plugins need to maintain a refcount for each volume, increasing the ref count on every mount and decrementing it on unmount. When the count is 0 the plugin is free to unmount the volume and free up resources (Detach the volume from the VM).

### Possible states
```
0. Init         : Volume exists but is not attached to any VM.
1. Attached     : Volume is attached to the VM but the FS is not mounted.
2. Mounted      : Volume FS is mounted but no container is using it.
3. InUse        : Volume FS is mounted and one or more containers is using it.
```

The plugin initiates state changes for the states of the volume, and only allows correct transitions.

```
State Transitions       # RefCount Change
-----------------------------------------
Init 	 -> Attached    # RefCount 0 -> 0
Attached -> Init        # RefCount 0 -> 0
Attached -> Mounted     # RefCount 0 -> 0
Mounted  -> Attached    # RefCount 1 -> 0
Mounted  -> InUse       # RefCount 0 -> 1
InUse    -> Mounted     # RefCount 1 -> 0
InUse 	 -> InUse       # RefCount++
```

### Possible failures and impact on volume states

* Docker engine crashes: Docker has not sent any unmount request to the plugin.
```
States: Volumes are either in Init or in Mounted state. 
Recovery: Mounted state needs recovery.
```
. Plugin crashes (Plugin accidentally restarted is considered a crash, install and upgrade require docker to be stopped, as mentioned above).
 ```
States: Volumes can be in any state and plugin needs to rebuild volume state. 
Recovery: The end goal is to insure volumes are in Init, or InUse.
```
* VM Crashes and starts up: Docker has not consumed any volume and plugin has no refcount.
```
States: Volumes can be in Init or Attached state. 
Recovery: In the absence of docker engine consuming volumes all volumes need to return to Init state.
```

Note: The manual/automated stopping and starting of docker covers the installation and upgrade case.

# Current issues with Docker

## Bugs 
There are bugs in the existing implementation for sending unmount requests. https://github.com/docker/docker/issues/22564
