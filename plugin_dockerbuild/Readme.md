### Support for building vsphere-storage-for-docker(DVS) as Docker Managed Plugin

Docker managed plugins are currently documented [here](https://docs.docker.com/engine/extend/)

Running `make all` in this folder will assume that
the DVS is pre-built and dockerhub is authenticared with `docker login`, then it will  package it
as a Managed Plugin, and push to dockerhub.

**Note that this is not integrated with Build or CI, which is an upcoming change**

Assuming the `Makefile`  defines names as 'cnastorage/vsphere-storage-for-docker:0.12', the plugin  could
be installed on Docker (1.13+) as follows:

* no question asked, and pretend the plugin name is 'vsphere' (can be used in `volume create` and `plugin rm`)
```
docker plugin install --grant-all-permissions --alias vsphere \
  cnastorage/vsphere-storage-for-docker:0.12
```
* vanilla interactive install as disabled plugin (will require `docker plugin enable `  to be operational
```
docker plugin install --disable cnastorage/vsphere-storage-for-docker:0.12
```

The `docker volume` command behaves as before but the plugin name in `docker create` is either what was passed as `--alias` or the full plugin name , i.e. `cnastorage/vsphere-storage-for-docker:0.12`

#### Logs

The plugin logs are still at `/var/log/vsphere-storage-for-docker.log`
However, you can change the log level by passing `VDVS_LOG_LEVEL` key to `docker plugin install`, e.g.
```
 docker plugin install --grant-all-permissions \
   cnastorage/vsphere-storage-for-docker:0.12  VDVS_LOG_LEVEL=debug
 ```

#### Socket group ID

The group ID to use when creating the socket file (used to field requests) can be specified via an env. variable "VDVS_SOCKET_GID", either when installing the plugin or later via docker plugin <plugin-id> set "VDVS_SOCKET_GID=<group ID>".
