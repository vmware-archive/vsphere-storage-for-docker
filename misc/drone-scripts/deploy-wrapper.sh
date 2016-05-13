#!/bin/sh
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

BUILD_NUMBER=$4

export ESX=$1
export VM1=$2
export VM2=$3

SCP="scp -o StrictHostKeyChecking=no"
SSH="ssh -o StrictHostKeyChecking=no"
USER=root
. ./misc/drone-scripts/cleanup.sh

$SCP ./misc/drone-scripts/lock.sh $VM1:/tmp/

# Unlock performed in stop_build in cleanup.sh
until $SSH $USER@$VM1 /tmp/lock.sh lock $BUILD_NUMBER
 do
  sleep 30
  echo "Retrying acquire lock"
done 

dump_vm_info() {
  set -x
  $SSH $USER@$1 uname -a
  $SSH $USER@$1 docker version
  set +x
}

dump_esx_info() {
  set -x
  $SSH $USER@$ESX uname -a
  $SSH $USER@$ESX vmware -vl
  set +x
}

echo "Acquired lock for build $BUILD_NUMBER"

echo "*************************************************************************"
echo "cleanup stale state"
echo "*************************************************************************"
cleanup

echo "*************************************************************************"
echo "starting deploy"
echo "*************************************************************************"

if make deploy-esx deploy-vm;
then
  dump_esx_info $ESX
  dump_vm_info $VM1
  dump_vm_info $VM2
  echo "*************************************************************************"
  echo "deploy done"
  echo "*************************************************************************"
else
  echo "*************************************************************************"
  echo "deploy failed cleaning up"
  echo "*************************************************************************"
 echo " Dumping logs..."
 . ./misc/drone-scripts/dump_log.sh
 dump_log $VM1 $VM2 $ESX
  stop_build $VM1 $BUILD_NUMBER
  exit 1
fi
