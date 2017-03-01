** Docker volumes on ESX **

Docker volumes on vSphere are powered by [VMDKs](https://en.wikipedia.org/wiki/VMDK). VMDKs can reside on datastores created on top of varying backends (NFS, SAN, VSAN, vVol). vSphere Docker Volume Service supports high availability for Docker volumes and allows for any VM requesting for the volume to gain access to it. The ability to attach the VMDK(Docker volume) to any VM when paired with a cluster manager such as Swarm allows a persistent container to be highly available.

The VMDKs on ESX are stored on the datastore in a folder that is independent of a VM.

<script type="text/javascript" src="https://asciinema.org/a/80424.js" id="asciicast-80424" async></script>
