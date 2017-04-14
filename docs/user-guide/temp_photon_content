# Configuration
## Photon Driver
The docker photon volume driver creates and manages the volumes on a photon platform. Photon platform API is used to create and manage volumes. You need two additional configurations for photon driver in the configuration file.

```
{
    "Driver": "<driver name - vsphere/vmdk/photon>"
    "MaxLogAgeDays": 28,
    "MaxLogSizeMb": 100,
    "LogPath": "/var/log/docker-volume-vsphere.log",
    "LogLevel": "info",
    "Target" : "http://<photon_controller_ip>:<target port>",
    "Project" : "<21-digit photon project ID>",
    "Host" : "<32-digit photon VM ID "
}
```
<table class="table table-striped table-hover ">
  <thead>
    <tr>
      <th>Parameter Name</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>target</td>
      <td>The endpoint of the Photon Controller with hostname and port in format: http://<photon_host>:<photon_port></td>
    </tr>
    <tr>
      <td>project</td>
      <td>The project ID in Photon to which this host belongs.</td>
    </tr>
    <tr>
      <td>host</td>
      <td>The ID of docker host in Photon.</td>
    </tr>
</tbody>
</table>

# CLI

### Photon

#### Flavour 
You can specify the flavour of persistent disk available in the Photon controller while creating a volume. The flavour in Photon implies certain limits on the volume being created.
More information about flavours is available [here](https://github.com/vmware/photon-controller/wiki/Flavors).

```
docker volume create --driver=photon --name=CloneVolume -o flavor=<Photon persistent disk flavor name>
```


For more information about Photon plaform, please refer [Wiki page](https://github.com/vmware/photon-controller/wiki).