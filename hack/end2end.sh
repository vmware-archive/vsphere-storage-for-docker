#!/bin/sh

# This scripts runs the end to end tests.

usage() {
  echo "./end2end.sh <ESX IP> <VM1 IP> <VM2 IP> <Optional Build id>"
  echo "root user will be used for all operations"
  echo "Advisable to setup ssh keys"
}

ESX=root@$1
VM1=root@$2
VM2=root@$3
BUILD_NUMBER=$4
SSH="ssh -o StrictHostKeyChecking=no"

if [ $# -lt 3 ] 
then
  usage
  exit 1
fi

. ./hack/cleanup.sh

echo "*************************************************************************"
echo "tests starting"
echo "*************************************************************************"

if make testremote VM=$VM1 VM2=$VM2 TEST_VOL_NAME=vol-build$BUILD_NUMBER 
then
  echo "*************************************************************************"
  echo "tests done"
  echo "*************************************************************************"
  stop_build $VM1 $BUILD_NUMBER
else
  echo "*************************************************************************"
  echo "tests failed"
  echo "starting cleanup"
  echo "*************************************************************************"
  stop_build $VM1 $BUILD_NUMBER
  # TODO remove vmdk files if any
  # TODO collect logs and publish
  exit 1
fi
