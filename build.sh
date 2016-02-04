#!/bin/bash 
# simple wrapper for docker-vmdk-plugin builds
#
# Assumes docker installed, and source already git-cloned

#set -x

# Check prereq first - we need docker 

ver=`docker -v` 
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

maj_ver=`echo $ver | grep -o '\.[0-9]*\.' | sed 's/\.//g'`
if [ $maj_ver -lt 8 ]
then 
   echo '***********************************************************'
   echo "Error: need Docker 1.9 or later. Found $ver"
   echo "         Please update docker to a newer version"
   exit 3

fi

if [ $maj_ver -eq 8 ]
then
        echo '***********************************************************'
        echo "Warning: if the build fails on Docker 1.8, "
        echo "         replace ARG with ENV in Dockerfile and rerun"
        echo "         Command: sed -i 's/ARG WHO/ENV WHO/' Dockerfile" 
        echo 
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

