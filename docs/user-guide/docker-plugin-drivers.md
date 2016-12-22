[TOC]
The Docker volume plugin supports the below platforms and the corresponding drivers for those. The plugin supports all volume provisioning and managenment operations, defined by the Docker Volume plugin interface, on both platforms.

1. Vmdk
2. Photon

<script type="text/javascript" src="https://asciinema.org/a/80417.js" id="asciicast-80417" async></script>

## Docker Vmdk volume driver 
The Docker Vmdk volume driver supports provisioning and managing docker volumes on a standalone or cluster of ESX servers via a service (ESX service) that's installed and runs on each server. Docker volumes are created and managed via publicly available VIM (Virtual Infrastructure Management) APIs on the ESX host.

## Docker Photon volume driver 
The Docker Photon Volume driver supports provisioning and managing docker volumes on a Photon platform consisting of a cluster of ESX hosts managed via a Photon controller instance. Docker volumes are created and managed entirely via the open Photon platform API via the Photon controller.

The Docker Volume plugin can support either or both types of volumes, as required, on a given Docker host.
