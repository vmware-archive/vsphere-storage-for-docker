#!/bin/bash

# The preffered choice to build the repo. Uses a docker image
# with all dependencies installed.

# Requirements:
# 1. Docker is installed locally with ability to connect to 
#    the public Docker registry.

usage ()
{
	echo ""
	echo "This is a simple wrapper to run the build inside a docker image with all dependencies installed."
	echo "Build targets can be passed to this script"
	echo ""
	echo "Example:"
	echo ""
	echo "./build.sh clean"
	echo "./build.sh deploy #Check README for details"
	echo ""
}

if [ "$1" == "-h" ]
then
	usage
	exit 1
fi

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

min_ver=`echo $ver | grep -o '\.[0-9]*\.' | sed 's/\.//g'`
if [ $min_ver -lt 8 ]
then 
   echo '***********************************************************'
   echo "Error: need Docker 1.8 or later. Found $ver"
   echo "         Please update docker to a newer version"
   exit 3

fi

plugin=docker-vmdk-plugin
plugin_container_version=0.5
plug_container=kerneltime/vibauthor-and-go:$plugin_container_version
dockerfile=Dockerfile.vibauthor-and-go

GOPATH=/go

# mount point within the container.
dir=$GOPATH/src/github.com/vmware/$plugin
docker_socket=/var/run/docker.sock
set -x
docker run --privileged --rm -v $docker_socket:$docker_socket -v $PWD:$dir -w $dir $plug_container make $1
set +x
