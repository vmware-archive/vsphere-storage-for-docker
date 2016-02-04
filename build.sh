#!/bin/bash 
# simple wrapper for docker-vmdk-plugin builds
#
# Assumes docker installed, and source already git-cloned

#set -x

# Check prereq first - we need docker 

d=`docker -v` 
if [ $? -ne 0 ]
then 
   echo '***********************************************************'
   echo "Error: Failed to find Docker - please install it first."
   exit 1
fi

docker ps > /dev/null
if [ $? -ne 0 ]
then 
   echo '***********************************************************'
   echo "Error: Docker is installed but not running or misconfigured"
   echo "       Please make sure you run 'docker ps' before retrying"
   exit 2
fi

echo $d | grep -q 'version 1.9'
if [ $? -ne 0 ]
then 
   echo '***********************************************************'
   echo "Warning: need Docker 1.9 or later. Found $d"
   echo "         Please update to a fresher version if the build fails"
   echo 
   echo "Note: for Docker 1.8, replace ARG with ENV in Dockerfile and rerun"
   echo '***********************************************************'
fi

# now do the work:

id=`id -u`
# use this for plugin location and container name
plugin=docker-vmdk-plugin  
plug_container=$plugin

# mount point within the container
dir=/work           

# use this for vibauthor (and opportunistically for git)
vibauth_container=lamw/vibauthor

# this will build GO part, and will prep all for vibauthor package
echo "Building GO and C code ..." 
docker build -q -t $plug_container .
docker run -u $id --rm -v $PWD:$dir -w $dir $plug_container make

# and package the vib (payloads is expecred to be ready by now)
vib=vmware-esx-vmdkops-1.0.0.vib
srv=vmdkops-esxsrv  # folder name with srv code


echo "Packaging $vib..."
docker run -u $id --rm -v $PWD/$srv:$dir -w $dir $vibauth_container \
        vibauthor --debug --compose \
        --vib=$vib --stage-dir $dir --force

# copy stuff to bin  more convenient manual testing with "rcp" deployment  
payload_bin=$srv/payloads/payload1/usr/lib/vmware/vmdkops/bin
cp $srv/$vib bin
cp $payload_bin/* bin

