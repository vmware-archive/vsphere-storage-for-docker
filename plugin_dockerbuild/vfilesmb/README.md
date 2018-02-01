# vfile samba
This is a simple SMB docker image for [vSphere shared storage for Docker](https://github.com/vmware/vsphere-storage-for-docker)

To start a single server container:
```
docker run -d --net=host -v vol1:/vfilepath --name samba cnastorage/vfile-smb:photon-v1.0
```

To mount:
```
sudo mount -t cifs -o username=vfile,password=vfile,vers=3.0 //[VM IP]/vfileshare /mnt
```

To start a service:
```
docker service create --network ingress --name vfilesmb -p 11139:139 -p 11445:445 -p 11137:137 --mount type=volume,source=vol1,dst=/vfilepath cnastorage/vfile-smb:photon-v1.0
```

To mount:
```
sudo mount -t cifs -o username=vfile,password=vfile,vers=3.0,port=11445 //127.0.0.1/vfileshare /mnt
