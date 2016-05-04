#!/bin/bash
# Copyright 2016 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.



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

plugin=docker-volume-vsphere
plugin_container_version=0.6
plug_container=cnastorage/vibauthor-and-go:$plugin_container_version
plug_pkg_container_version=latest
plug_pkg_container=cnastorage/fpm:$plug_pkg_container_version
dockerfile=Dockerfile.vibauthor-and-go
DOCKER="$DEBUG docker"

# mount point within the container.
GOPATH=/go
dir=$GOPATH/src/github.com/vmware/$plugin

if [ "$1" == "rpm" ] || [ "$1" == "deb" ]
then
  $DOCKER run --rm  \
    -e "PKG_VERSION=$PKG_VERSION" \
    -v $PWD:$dir \
    -w $dir \
    $plug_pkg_container \
    make $1
else
  docker_socket=/var/run/docker.sock
  $DOCKER run --privileged --rm \
    -e "PKG_VERSION=$PKG_VERSION" \
    -v $docker_socket:$docker_socket  \
    -v $PWD:$dir -w $dir $plug_container make $1
fi
