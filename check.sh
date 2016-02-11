#!/bin/bash 
#
# simple wrapper for check build pre-requisites 
#

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
