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

# A few helper functions for dumping logs

VM_LOGFILE="/var/log/docker-volume-vsphere.log"
ESX_LOGFILE="/var/log/vmware/vmdk_ops.log"
HOSTD_LOGFILE="/var/log/hostd.log"

dump_log_esx() {
  log $ESX_LOGFILE
  $SSH $USER@$1 cat $ESX_LOGFILE
  if [ $INCLUDE_HOSTD == "true" ]
  then
    log $HOSTD_LOGFILE
    $SSH $USER@$1 cat $HOSTD_LOGFILE
  fi
}

dump_log_vm(){
  $SSH $USER@$1 cat $VM_LOGFILE
}

truncate_vm_logs() {
  $SSH $USER@$1 "echo > $VM_LOGFILE"
}

truncate_esx_logs() {
  $SSH $USER@$1 "echo > $ESX_LOGFILE"
  if [ $INCLUDE_HOSTD == "true" ]
  then
    $SSH $USER@$1 "echo > $HOSTD_LOGFILE"
  fi
}
