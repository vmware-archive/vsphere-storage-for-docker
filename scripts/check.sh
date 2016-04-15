#!/bin/bash

#
# simple wrapper for check build pre-requisites
#

printNQuit() {
   echo $1
   exit 1
}

# dockerbuild needs docker :-)

if [ "$1" == "dockerbuild" ]
then
 command -v docker > /dev/null
 if [ $? -ne 0 ]
 then
   # TODO: To run docker build an older version of docker than
   # 1.9 might be ok..?
   printNQuit "Docker is needed to run dockerbuild"
 fi
 if [ "$(docker version -f '{{.Server.Os}}')" != "linux" ]
 then
   printNQuit "This build requires Docker Server running on Linux."
 fi
 exit 0
fi

# pgk build just needs FPM (it is taken care of by Dockerfiles/Dockerfile.fpm)

if [ "$1" == "pkg" ]
then
 command -v fpm > /dev/null
 if [ $? -ne 0 ]
 then
   printNQuit "FPM needs to be installed"
 fi
 exit 0
fi

# And all other builds need GO 1.5+ version and proper config
#------------------------------------------------------------

if [ `uname` != "Linux" ]
then
  echo "This build is supported only on Linux."
  exit 1
fi
# Check

GO_REQUIRED=1.5

command -v go > /dev/null
if [ $? -ne 0 ]
then
   printNQuit "go lang version 1.5 or higher is needed"
fi

GO_INSTALLED=$(go version| awk '{print $3}'| sed -r 's/([a-z])//g')

if (( $(echo "$GO_INSTALLED $GO_REQUIRED"| awk '{print $1 < $2}') ))
then
   printNQuit "Error: go 1.5 or higher is needed for vendoring support"
fi

if [[ -z "$GOPATH" ]]
then
   printNQuit "GOPATH environment variable needs to be set"
fi

EXPECTED_PATH="$GOPATH/src/github.com/vmware/docker-vmdk-plugin"

if [[ "$EXPECTED_PATH" != "$PWD" ]]
then
   printNQuit "There is a local import of a package. The src should be under GOPATH=$GOPATH \
               Expected:\" $EXPECTED_PATH \" CURRENTPATH:\" $PWD:"
fi

# check vibauthor availability
#-----------------------------

command -v vibauthor  > /dev/null
if [ $? -ne 0 ]
then
   printNQuit "Error: vibauthor needs to be installed to build repo."
fi
