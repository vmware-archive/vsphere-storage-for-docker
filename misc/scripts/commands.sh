#!/bin/bash
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

# This file contains the command definitions used in other scripts.

# Note: DEBUG is debug assistance, i.e. if DEBUG is empty/not defined, 
# the actual command is execd.  If 'DEBUG=echo' is typed before 'make', 
# then instead of command (e.g. scp) "echo scp" will be execd so it will 
# print out the commands. Super convenient for debugging.

DEB_INSTALL="dpkg -i"
DEB_PURGE="dpkg -P"
DEB_QUERY="dpkg-query"
IS_DEB="grep -i ubuntu /etc/os-release"
IS_RPM="grep -i photon /etc/os-release"
IS_SYSTEMD="stat /proc/1/exe | grep File | grep systemd"
MKDIR_P="mkdir -p"
GREP=grep
PIDOF=pidof
PS="ps aux"
RM_RF="rm -rf"
RPM_ERASE="rpm -e"
RPM_INSTALL="rpm -ivh"
RPM_QUERY="rpm -q"
SCHED_GRP="localcli --plugin-dir=/usr/lib/vmware/esxcli/int sched group"
SCP="$DEBUG scp -r -q -o StrictHostKeyChecking=no"
SSH="$DEBUG ssh -kTax -o StrictHostKeyChecking=no"
VIB_INSTALL="localcli software vib install"
VIB_REMOVE="localcli software vib remove"
VMDK_OPSD="/etc/init.d/vmdk-opsd"

function log {
    echo 
    echo "=> $1"
    echo
}
