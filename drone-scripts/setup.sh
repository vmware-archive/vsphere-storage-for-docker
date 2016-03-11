#!/bin/sh

# This script sets up the testbed.

usage() {
  echo "./setup.sh <ESX IP> <VM1 IP> <VM2 IP> <Optional Build id>"
  echo "root user will be used for all operations"
  echo "Advisable to setup ssh keys."
  echo "run this script from the root of the repo"
}

if [ $# -lt 3 ] 
then
  usage
  exit 1
fi

#TODO generalize to an array of VMs
ESX=root@$1
VM1=root@$2
VM2=root@$3
BUILD_NUMBER=$4

SCP="scp -o StrictHostKeyChecking=no"
SSH="ssh -o StrictHostKeyChecking=no"

. ./drone-scripts/cleanup.sh

$SCP ./drone-scripts/lock.sh $VM1:/tmp/

# Unlock performed in stop_build in cleanup.sh
until $SSH $VM1 /tmp/lock.sh lock $BUILD_NUMBER
 do
  sleep 30
  echo "Retying acquire lock"
done 

echo "Acquired lock for build $BUILD_NUMBER"

echo "*************************************************************************"
echo "cleanup stale state"
echo "*************************************************************************"
cleanup

echo "*************************************************************************"
echo "starting deploy"
echo "*************************************************************************"

if make deploy-esx ESX=$ESX && make deploy-vm VM=$VM1 && make deploy-vm VM=$VM2;
then
  echo "*************************************************************************"
  echo "deploy done"
  echo "*************************************************************************"
else
  echo "*************************************************************************"
  echo "deploy failed cleaning up"
  echo "*************************************************************************"
  stop_build $VM1 $BUILD_NUMBER
  exit 1
fi
