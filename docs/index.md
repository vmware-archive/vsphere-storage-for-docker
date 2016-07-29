# Introduction to Docker Volume Driver for vSphere

Docker volume driver is designed to solve persistency needs for 
stateful containers running on top of VMware's stack.

Some of the high level features for the plugin are: 

1. Designed to run over shared storage in a cluster (Single node
 setup for testing is supported)
2. Easy to deploy and manage. There is zero configuration and 
zero credential management. 
3. Support for VSAN policy.
4. Integration with vCenter (under development)
5. Backup of Docker volumes (under development)

This plugin is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/plugins_volume/). This plugin does not need credential management or configuration management. 

# Feedback

On going work and feature requests are tracked using [GitHub Issues](https://github.com/vmware/docker-volume-vsphere/issues). Please feel free to file [issues](https://github.com/vmware/docker-volume-vsphere/issues) or email [cna-storage@vmware.com](mailto:cna-storage@vmware.com)

# Documentation Version
The documentation here is for the latest release. The current master documentation can be found in markdown format in the [documentation folder](https://github.com/vmware/docker-volume-vsphere/tree/master/docs). For older releases, browse to [releases](https://github.com/vmware/docker-volume-vsphere/releases) select the release, click on the tag for the release and browse the docs folder.
