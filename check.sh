#!/bin/bash 
#
# simple wrapper for check build pre-requisites 
#

# Check prereq first - we need docker 

if [ "$1" = "nodocker" ] 
then 
        if [ `uname` != "Linux" ] 
        then
              #  don't want to mess with cross-compile
              echo '***********************************************************'
              echo "Error: Linux box is needed for no-docker build"
              exit 3
        fi
        exit 0
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
