#!/bin/sh

# This scripts runs the end to end tests.

usage() {
  echo "./end2end.sh <ESX IP> <VM1 IP> <VM2 IP> <Optional Build id>"
  echo "root user will be used for all operations"
  echo "Advisable to setup ssh keys"
  echo "Run this script from the root of the repo"
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

. ./drone-scripts/cleanup.sh

echo "*************************************************************************"
echo "tests starting"
echo "*************************************************************************"

dump_log() {
  echo "*************************************************************************"
  echo "dumping log: ESX " $ESX
  echo "*************************************************************************"
  $SSH $ESX cat /var/log/vmware/docker-vmdk-plugin.log
  $SSH $ESX cat /tmp/plugin.log
  echo "*************************************************************************"
  echo "dumping log: VM " $VM1
  echo "*************************************************************************"
  $SSH $VM1 cat /tmp/plugin.log
  echo "*************************************************************************"
  echo "dumping log: VM " $VM2
  echo "*************************************************************************"
  $SSH $VM2 cat /tmp/plugin.log
}

if make testremote VM=$VM1 VM2=$VM2 TEST_VOL_NAME=vol-build$BUILD_NUMBER
then
  echo "*************************************************************************"
  echo "tests done"
  echo "*************************************************************************"
  dump_log
  stop_build $VM1 $BUILD_NUMBER
else
  echo "*************************************************************************"
  echo "tests failed"
  echo "*************************************************************************"
  dump_log
  echo "*************************************************************************"
  echo "cleaning up"
  echo "*************************************************************************"

  stop_build $VM1 $BUILD_NUMBER
  # TODO remove vmdk files if any
  # TODO collect logs and publish
  exit 1
fi
