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


# This script sets up the testbed and invokes tests.

usage() {
  echo "$0 <TARGET> <ESX IP> <VM1 IP> <VM2 IP> <Build id> <Installation flag>"
  echo "root user will be used for all operations"
  echo "Advisable to setup ssh keys."
  echo "run this script from the root of the repo"
}

if [ $# -lt 2 ]
then
  usage
  exit 1
fi

FUNCTION_NAME=$1
BUILD_NUMBER=$5
NEED_INSTALLATION=$6

export ESX=$2
export VM1=$3
export VM2=$4

USER=root
. ./misc/scripts/commands.sh
. ./misc/drone-scripts/cleanup.sh
. ./misc/drone-scripts/dump_log.sh

dump_vm_info() {
  set -x
  $SSH $USER@$1 uname -a
  $SSH $USER@$1 docker version
  set +x
}

dump_esx_info() {
  echo
  $SSH $USER@$ESX uname -a
  echo
  $SSH $USER@$ESX vmware -vl
  echo
  $SSH $USER@$ESX df
  echo
  $SSH $USER@$ESX ls -ld /vmfs/volumes/*
  echo
}

dump_logs() {
  log "Info ESX $ESX"
  dump_esx_info $ESX
  log "Info VM1 $VM1"
  dump_vm_info $VM1
  log "Info VM2 $VM2"
  dump_vm_info $VM2
  log "Log VM1 $VM1"
  dump_log_vm $VM1
  log "Log VM2 $VM2"
  dump_log_vm $VM2
  log "log ESX $ESX"
  dump_log_esx $ESX
}


log "truncate vm logs"
truncate_vm_logs $VM1
truncate_vm_logs $VM2

log "truncate esx logs"
truncate_esx_logs $ESX

log "starting deploy and test"

INCLUDE_HOSTD="false"


INSTALL_VIB="clean-auth-db deploy-esx"

if [ "$NEED_INSTALLATION" ]
then
  TARGET+=$INSTALL_VIB
fi

PARAMETER="TEST_VOL_NAME=vol.build$BUILD_NUMBER"

case $FUNCTION_NAME in
pluginSanityCheck)
        TARGET+=" build-plugin"
        ;;
runtests)
        if [ -e /tmp/$ESX ]
        then
          TARGET+=" test-e2e-runalways test-vm"
        else
          touch /tmp/$ESX
          TARGET+=" deploy-vm test-e2e-runalways test-e2e-runonce testasroot test-esx test-vm"
        fi
        ;;
coverage)
        TARGET=" coverage"
        ;;
winplugin)
        TARGET=" build-windows-plugin deploy-windows-plugin test-e2e-runonce-windows"
        ;;
vfileplugin)
        TARGET=" build-vfile-plugin deploy-vfile-plugin test-e2e-runonce-vfile"
        ;;
esac

if make -s $TARGET $PARAMETER;
then
 echo "=> Build Complete" `date`
  #stop_build $VM1 $BUILD_NUMBER
else
 log "Build + Test not successful"
 INCLUDE_HOSTD="true"
 dump_logs
 stop_build $VM1 $BUILD_NUMBER
 exit 1
fi
