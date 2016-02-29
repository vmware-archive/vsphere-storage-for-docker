#!/bin/bash 

#
# simple wrapper for check build pre-requisites 
#

# Check prereq first - we need docker 

use_docker() {
   echo $1
   echo "The preffered build is via ./build.sh using docker"
   exit 1
}

GO_REQUIRED=1.5

command -v go > /dev/null
if [ $? -ne 0 ]
then 
   use_docker "go lang version 1.5 or higher is needed"
fi

GO_INSTALLED=$(go version| awk '{print $3}'| sed -r 's/([a-z])//g')

if (( $(echo "$GO_INSTALLED $GO_REQUIRED"| awk '{print $1 < $2}') ))
then
   use_docker "Error: go 1.5 or higher is needed for vendoring support"
 fi

if [[ -z "$GOPATH" ]] 
then
   use_docker "GOPATH environment variable needs to be set"
fi

EXPECTED_PATH="$GOPATH/src/github.com/vmware/docker-vmdk-plugin"

if [[ "$EXPECTED_PATH" != "$PWD" ]] 
then
   use_docker "There is a local import of a package. The src should be under GOPATH=$GOPATH \
               Expected:\" $EXPECTED_PATH \" CURRENTPATH:\" $PWD:"
fi 

command -v vibauthor  > /dev/null
if [ $? -ne 0 ]
then 
   use_docker "Error: vibauthor needs to be installed to build repo."
fi
