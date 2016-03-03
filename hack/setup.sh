#!/bin/sh

# This script sets up the testbed.

usage() {
  echo "./setup.sh <ESX IP> <VM1 IP> <VM2 IP>"
  echo "root user will be used for all operations"
  echo "Advisable to setup ssh keys"
}

VM1=root@$2
VM2=root@$3
ESX=root@$1

#TODO generalize to an array of VMs

cleanup() {
  make clean-vm VM=$VM1
  make clean-vm VM=$VM2
  make clean-esx ESX=$ESX
}

echo "*************************************************************************"
echo "cleanup any stale state"
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
  cleanup
  exit 1
fi
