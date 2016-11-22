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


#
# deploy-tools.sh
#
# Has a set of functions to deploy to ESX and guests, start and stop services
# and clean up
#
# Usage:
#       ./misc/scripts/deploy-tools function-name function-params
#
# e.g.
#      ./misc/scripts/deploy-tools deployvm "ip-addresses" "package folder"
#

. ../misc/scripts/commands.sh

PLUGIN_NAME=docker-volume-vsphere
VIB_NAME=esx-vmdkops-service
TMP_LOC=/tmp/$PLUGIN_NAME
VMDK_OPS_UNITTEST=/tmp/vmdk_ops_unit*

# VM Functions

function deployvmtest {
    for ip in $IP_LIST
    do
        TARGET=root@$ip
        log "Deploying test code to $TARGET"
        $SSH $TARGET $MKDIR_P $TMP_LOC
        $SCP $SOURCE/*.test $TARGET:$TMP_LOC
        $SCP $SCRIPTS/refcnt_test.sh $TARGET:$TMP_LOC
        $SCP $SCRIPTS/wait_for.sh $TARGET:$TMP_LOC
    done
}

function deployvm {
    for ip in $IP_LIST
    do
        TARGET=root@$ip
        setupVMType
        log "Deploying $TARGET : $FILE_EXT"
        deployVMPre
        deployVMInstall
        deployVMPost
    done
}

function setupVMType {
    $SSH $TARGET "$IS_PHOTON > /dev/null" || $SSH $TARGET "$IS_RHEL > /dev/null"
    if [ "$?" == "0" ]
    then
        FILE_EXT="rpm"
        return 0;
    fi

    $SSH $TARGET "$IS_DEB > /dev/null"
    if [ "$?" == "0" ]
    then
        FILE_EXT="deb"
        return 0
    else
        log "setupVMType: Unsupported VM Type $TARGET"
        exit 1
    fi
}

function deployVMPre {
    $SSH $TARGET $MKDIR_P $TMP_LOC
    $SCP $SOURCE/*.$FILE_EXT $TARGET:$TMP_LOC
}

function deployVMInstall {
    set -e
    case $FILE_EXT in
    deb)
        $SSH $TARGET "$DEB_INSTALL $TMP_LOC/*.deb"
        ;;
    rpm)
        $SSH $TARGET "$RPM_INSTALL $TMP_LOC/*.rpm"
        ;;
    esac
    set +e
}

function deployVMPost {
    $SSH $TARGET "$PIDOF $PLUGIN_NAME"
    if [ $? -ne 0 ]
    then
        log "deployVMPost: $PLUGIN_NAME is not running on $TARGET"
        exit 1
    fi
}

# ESX Functions

# deployesx
#
# Deploys plugin code on ESX(s) using VIB mentioned in $SOURCE :

function deployesx {
    for ip in $IP_LIST
    do
        TARGET=root@$ip
        log "Deploying to ESX $TARGET"
        $SSH $TARGET  rm -f /etc/vmware/vmdkops/log_config.json 
        deployESXPre
        deployESXInstall
        deployESXPost
    done
}

function deployESXPre {
    $SSH $TARGET $MKDIR_P $TMP_LOC
    $SCP $SOURCE $TARGET:$TMP_LOC
}

function deployESXInstall {
    $SSH $TARGET $VIB_INSTALL --no-sig-check -v $TMP_LOC/$(basename $SOURCE)
    if [ $? -ne 0 ]
    then
        log "deployESXInstall: Installation hit an error on $TARGET"
        exit 2
    fi
}

function deployESXPost {
    $SSH $TARGET $VMDK_OPSD status
    if [ $? -ne 0 ]
    then
        log "deployESXPost: Service is not running on $TARGET"
        exit 3
    fi
    $SSH $TARGET $SCHED_GRP list| $GREP vmdkops | $GREP python> /dev/null
    if [ $? -ne 0 ]
    then
        log "deployESXPost: Service is not configured on $TARGET"
        exit 4
    fi
}

# cleanesx
#
# Does vib and tmp files cleanup on ESX

function cleanesx {
    for ip in $IP_LIST
    do
        log "Cleaning up on ESX $ip"
        TARGET=root@$ip
        $SSH $TARGET "$VIB_LIST | $GREP $VIB_NAME 2&>1 > /dev/null"
        if [ $? -eq 0 ];
        then
            $SSH $TARGET $VIB_REMOVE --vibname $VIB_NAME
        fi
        $SSH $TARGET $SCHED_GRP list \
            --group-path=host/vim/vimuser/cnastorage/ > /dev/null 2>&1
        if [ $? -eq 0 ];
        then
            log "cleanesx: Failed to clean up resource groups!"
            exit 1
        fi
        $SSH $TARGET "$RM_RF $TMP_LOC"
        $SSH $TARGET "$RM_RF $VMDK_OPS_UNITTEST"
    done
    rm -f /etc/vmware/vmdkops/log_config.json 
}

# cleanvm
#
# Does vib and tmp files cleanup on Linux guests

function cleanvm {
    set +e
    for IP in $IP_LIST
    do
        TARGET=root@$IP
        setupVMType
        log "cleanvm: Cleaning up on $TARGET : $FILE_EXT"
        cleanupVMPre
        cleanupVM
        cleanupVMPost
    done
}

function cleanupVMPre {
    case $FILE_EXT in
    deb)
        $SSH $TARGET "$DEB_QUERY $PLUGIN_NAME > /dev/null"
        if [ $? -eq 0 ]
        then
            let PLUGIN_INSTALLED=0
        else
            let PLUGIN_INSTALLED=1
        fi
        ;;
    rpm)
        $SSH $TARGET "$RPM_QUERY $PLUGIN_NAME > /dev/null"
        if [ $? -eq 0 ]
        then
            let PLUGIN_INSTALLED=0
        else
            let PLUGIN_INSTALLED=1
        fi
        ;;
    esac

    if [ $PLUGIN_INSTALLED -eq 0 ]
    then
        $SSH $TARGET "$IS_SYSTEMD > /dev/null"
        if [ $? -eq 0 ]
        then
            $SSH $TARGET systemctl stop docker
            $SSH $TARGET systemctl stop $PLUGIN_NAME
            $SSH $TARGET $PIDOF $PLUGIN_NAME
            $SSH $TARGET systemctl start docker
        else
            $SSH $TARGET service docker stop
            $SSH $TARGET service $PLUGIN_NAME stop
            $SSH $TARGET service docker start
        fi
    fi
}

function cleanupVM {
    case $FILE_EXT in
    deb)
        if [ $PLUGIN_INSTALLED -eq 0 ]
        then
            $SSH $TARGET $DEB_PURGE $PLUGIN_NAME
        fi
        ;;
    rpm)
        if [ $PLUGIN_INSTALLED -eq 0 ]
        then
            $SSH $TARGET $RPM_ERASE $PLUGIN_NAME
        fi
        ;;
    esac
}

function cleanupVMPost {
    $SSH $TARGET "$PIDOF $PLUGIN_NAME"
    if [ "$?" == "0" ]
    then
        log "cleanupVMPost: Service still running on $TARGET"
        exit 4
    fi

    $SSH $TARGET "$RM_RF $TMP_LOC"
}

#
# define usage message

function usage {
   echo $1
   echo <<EOF
deploy-tools.sh provideds a set of helpers for deploying $PLUGIN_NAME
binaries to ESX and to guest VMs.

Usage:  deploy-tools.sh command params...

Comands and params are as follows:
deployesx "esx-ips"  "vib file"
deployvm  "vm-ips"  "folder containig deb or rpm"
deployvmtest "vm-ips" "folder containing test binaries" "folder containing scripts"
cleanesx  "esx-ips"
cleanvm   "vm_ips"  "test-volumes-to-clean"
EOF
    exit 1
}

# ============= "main" ===============
#
# Check params, and call the requested function


# first param is always function name to invoke
FUNCTION_NAME=$1 ; shift

# globals used in misc. functions instead of params (to avoid extra parse)
IP_LIST=`echo $1 | xargs -n1 | sort -u | xargs` # dedup the IP list

# check that all params are present:
if [ -z "$FUNCTION_NAME" -o  -z "$IP_LIST" ]
then
    usage "Missing parameters: need at least \"func-name ipaddr\""
fi

case $FUNCTION_NAME in
deployesx)
        SOURCE="$2"
        if [ -z "$SOURCE" ]
        then
            usage "Missing params: folder hosting vib-name" ;
        fi
        deployesx
        ;;
cleanesx)
        cleanesx
        ;;
deployvm)
        SOURCE="$2"
        if [ -z "$SOURCE" ]
  then
        usage "Missing params: folder"
  fi
        deployvm
        ;;
deployvmtest)
        SOURCE="$2"
        SCRIPTS="$3"
        if [ -z "$SOURCE" -o -z "$SCRIPTS" ]
        then
            usage "Missing params: binary or scripts folder"
        fi
        deployvmtest
        ;;
cleanvm)
        VOLUMES="$2"
        if [ -z "$VOLUMES" ]
        then
            usage "Missing params: volume"
        fi
        cleanvm
        ;;
*)
        usage "Unknown function:  \"$FUNCTION_NAME\""
        ;;
esac
