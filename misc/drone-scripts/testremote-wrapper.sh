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


# This scripts runs the end to end tests.

usage() {
  echo "./end2end.sh <ESX IP> <VM1 IP> <VM2 IP> <Optional Build id>"
  echo "root user will be used for all operations"
  echo "Advisable to setup ssh keys"
  echo "Run this script from the root of the repo"
}

export ESX=$1
export VM1=$2
export VM2=$3
export BUILD_NUMBER=$4

if [ $# -lt 3 ]
then
  usage
  exit 1
fi

. ./misc/drone-scripts/cleanup.sh
. ./misc/drone-scripts/dump_log.sh

echo "*************************************************************************"
echo "tests starting"
echo "*************************************************************************"

if make testasroot testremote TEST_VOL_NAME=vol-build$BUILD_NUMBER
then
  echo "*************************************************************************"
  echo "tests done"
  echo ""
  dump_log $VM1 $VM2 $ESX
  stop_build $VM1 $BUILD_NUMBER
else
  echo "*************************************************************************"
  echo "tests failed"
  echo ""
  dump_log $VM1 $VM2 $ESX
  echo "*************************************************************************"
  echo "cleaning up"
  echo "*************************************************************************"

  stop_build $VM1 $BUILD_NUMBER
  exit 1
fi
