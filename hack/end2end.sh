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

echo "*************************************************************************"
echo "tests starting"
echo "*************************************************************************"

if make testremote VM=$VM1 VM2=$VM2 TEST_VOL_NAME=vol-build$BUILD_NUMBER 
then
  echo "*************************************************************************"
  echo "tests done"
  echo "*************************************************************************"
  make clean-vm VM=$VM1
  make clean-vm VM=$VM2
  make clean-esx ESX=$ESX
else
  echo "*************************************************************************"
  echo "tests failed"
  echo "starting cleanup"
  echo "*************************************************************************"
  make clean-vm VM=$VM1
  make clean-vm VM=$VM2
  make clean-esx ESX=$ESX
  # TODO remove vmdk files if any
  # TODO collect logs and publish
  exit 1
fi
