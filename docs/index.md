# Introduction to Docker Volume Driver for vSphere

vSphere Docker Volume service enables you to run stateful containerized applications on top of VMware’s stack i.e. on vSphere. 

It is designed to:

- **Provide proven persistent shared storage:** You can now use any VMware supported enterprise class storage backed by vSAN, VMFS, NFS, etc. 
- **Enable Multitenancy, Security and Access Control:** vSphere Admins can effortlessly set access permissions for shared storage across hosts, datastores and VMs from a single location
- **Provide Operational Simplicity:** Zero Configuration, zero credential management. It as easy to deploy and manage

Enable self-serve operational model: Use docker APIs to manage volume lifecycle while maintaining admin control over consumption 

This plugin is integrated with [Docker Volume Plugin framework](https://docs.docker.com/engine/extend/plugins_volume/). This plugin does not need credential management or configuration management. 
 
<script type="text/javascript" src="https://asciinema.org/a/80417.js" id="asciicast-80417" async></script>

# Feedback

On going work and feature requests are tracked using [GitHub Issues](https://github.com/vmware/docker-volume-vsphere/issues). Please feel free to file [issues](https://github.com/vmware/docker-volume-vsphere/issues) or email [cna-storage@vmware.com](mailto:cna-storage@vmware.com)

# Documentation Version
The documentation here is for the latest release. The current master documentation can be found in markdown format in the [documentation folder](https://github.com/vmware/docker-volume-vsphere/tree/master/docs). For older releases, browse to [releases](https://github.com/vmware/docker-volume-vsphere/releases) select the release, click on the tag for the release and browse the docs folder.
