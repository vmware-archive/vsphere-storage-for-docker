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

tmp_loc=/tmp/docker-volume-vsphere

# VM Functions

function deployvm {
    for ip in $IP_LIST
    do
        TARGET=root@$ip
        setupVMType
        echo "=> $TARGET : $FILE_EXT"
        deployVMPre
        deployVMInstall
        deployVMPost
    done
}

function setupVMType {
    $SSH $TARGET "$GREP -i photon /etc/os-release" > /dev/null
    if [ "$?" == "0" ]
    then
        FILE_EXT="rpm"
        return 0;
    fi

    $SSH $TARGET "$GREP -i ubuntu /etc/os-release" > /dev/null
    if [ "$?" == "0" ]
    then
        FILE_EXT="deb"
        return 0
    else
        echo "Unsupported VM Type $TARGET"
        exit 1
    fi
}

function deployVMPre {
    $SSH $TARGET $MKDIR_P $tmp_loc
    $SCP $SOURCE/*.$FILE_EXT $TARGET:$tmp_loc
}

function deployVMInstall {
    echo "=> VM Installing"
    set -e
    case $FILE_EXT in
    deb)
        $SSH $TARGET "$DEB_INSTALL $tmp_loc/*.deb" > /dev/null 2>&1
        $SSH $TARGET "$DEB_QUERY -s docker-volume-vsphere"
        ;;
    rpm)
        $SSH $TARGET "$RPM_INSTALL $tmp_loc/*.rpm > /dev/null"
        $SSH $TARGET "$RPM_QUERY docker-volume-vsphere"
        ;;
    esac
    set +e
}

function deployVMPost {
   echo "=> VM Post"
   $SSH $TARGET "$PS | $GREP docker-volume-vsphere| $GREP -v $GREP"
   if [ $? -ne 0 ] 
   then
      echo "docker-volume-vsphere is not running on $TARGET"
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
        deployESXPre
        deployESXInstall
        deployESXPost
    done
}

function deployESXPre {
    $SSH $TARGET $MKDIR_P $tmp_loc
    $SCP $SOURCE $TARGET:$tmp_loc
}

function deployESXInstall {
    $SSH $TARGET $VIB_INSTALL --no-sig-check -v $tmp_loc/*.vib
    if [ $? -ne 0 ] 
    then
        echo "Installation hit an error on $TARGET"
        exit 2
    fi
}

function deployESXPost {
    $SSH $TARGET $VMDK_OPSD status 
    if [ $? -ne 0 ] 
    then
        echo "Service is not running on $TARGET"
        exit 3
    fi
    $SSH $TARGET $SCHED_GRP list| $GREP vmdkops | $GREP python> /dev/null
    if [ $? -ne 0 ] 
    then
        echo "Service is not configured on $TARGET"
        exit 4
    fi
}

# cleanesx
#
# Does vib and tmp files cleanup on ESX

function cleanesx {
    for ip in $IP_LIST
    do
        echo "Cleaning up on ESX $ip..."
        TARGET=root@$ip
	$SSH $TARGET $VIB_REMOVE --vibname esx-vmdkops-service 
	$SSH $TARGET $SCHED_GRP list \
            --group-path=host/vim/vimuser/cnastorage/ > /dev/null 2>&1
	if [ $? -eq 0 ];
	then
	    echo "Failed to clean up resource groups!"
	    exit 1
	fi
        $SSH $TARGET "$RM_RF $tmp_loc"
    done
}

# cleanvm
#
# Does vib and tmp files cleanup on Linux guests

function cleanvm {
    NAME=docker-volume-vsphere
    for IP in $IP_LIST
    do
        echo "Cleaning up on VM $IP..."
        TARGET=root@$IP
        cleanupVolumes
        setupVMType
        cleanupVMPre
        cleanupVM
        cleanupVMPost
    done
}

function cleanupVolumes {
    echo "=> Asking docker to remove volumes ($VOLUMES)"
    for vol in $VOLUMES
    do
        $SSH $TARGET "if docker volume ls | $GREP -q $vol; then \
        docker volume rm $vol; fi "
    done
}

function cleanupVMPre {
    $SSH $TARGET systemctl stop $NAME
}

function cleanupVM {
    case $FILE_EXT in
    deb)
        $SSH $TARGET dpkg -P $NAME
        ;;
    rpm)
        $SSH $TARGET rpm -e $NAME
        ;;
    esac
}

function cleanupVMPost {
    $SSH $TARGET "ps au | $GREP $NAME | $GREP -v $GREP"
    if [ "$?" == "0" ]
    then 
        echo "Service still running on $TARGET"
        exit 4
    fi

    $SSH $TARGET "$RM_RF $tmp_loc"
}

#
# define usage message

function usage {
   echo $1
   echo <<EOF
deploy-tools.sh provideds a set of helpers for deploying docker-volume-vsphere
binaries to ESX and to guest VMs. 

Usage:  deploy-tools.sh command params...

Comands and params are as follows: 
deployesx "esx-ips"  "vib file"
deployvm  "vm-ips"  "folder containig deb or rpm"
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
        if [ -z "$SOURCE" ] ; then usage "Missing params: folder" ; fi
        deployvm
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
