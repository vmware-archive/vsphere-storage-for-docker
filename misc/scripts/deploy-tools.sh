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
VFILE_PLUGNAME=vfile
VIB_NAME=esx-vmdkops-service
TMP_LOC=/tmp/$PLUGIN_NAME
VMDK_OPS_UNITTEST=/tmp/vmdk_ops_unit*
BUILD_LOC=$TMP_LOC/build
PLUGIN_LOC=$TMP_LOC/plugin_dockerbuild
COMMON_VARS="../Commonvars.mk"
VMDK_OPS_CONFIG=/etc/vmware/vmdkops/log_config.json
PLUGIN_SRC_DIR=../
PLUGIN_SRC_ZIP=vdvs-src.zip
PLUGIN_BIN_ZIP=vdvs-bin.zip
WIN_TEMP_DIR=C:\\Users\\root\\AppData\\Local\\Temp
WIN_PLUGIN_SRC_DIR=C:\\Users\\root\\go\\src\\github.com\\vmware\\docker-volume-vsphere
WIN_PLUGIN_BIN_ZIP_LOC=$WIN_PLUGIN_SRC_DIR\\build\\windows\\docker-volume-vsphere.zip
WIN_BUILD_DIR=build/windows/
HOSTD=/etc/init.d/hostd

# VM Functions

function deployvmtest {
    for ip in $IP_LIST
    do
        TARGET=root@$ip
        log "Deploying test code to $TARGET"
        $SSH $TARGET $MKDIR_P $TMP_LOC
        $SCP $SOURCE/*.test $TARGET:$TMP_LOC
        $SCP $SCRIPTS/wait_for.sh $TARGET:$TMP_LOC
    done
}

function deployvm {
    for ip in $IP_LIST
    do
        TARGET=root@$ip
        installManagedPlugin
        deployVMPost
    done
}

function buildplugin {
    for ip in $IP_LIST
    do
        TARGET=root@$ip
        log "Cleaning up older files from $TARGET if exists..."
        $SSH $TARGET "rm -fr $TMP_LOC; $MKDIR_P $TMP_LOC $BUILD_LOC"
        log "Copying required files to $TARGET ..."
        $SCP $PLUGIN_BIN $TARGET:$BUILD_LOC
        $SCP $MANAGED_PLUGIN_SRC $COMMON_VARS $TARGET:$TMP_LOC
        if [ -z  "$($SSH $TARGET 'which make')" ]; then
            setupVMType
            case $FILE_EXT in
            deb)
                log "installing make on ubuntu machine"
                $SSH $TARGET 'apt install make'
                ;;
            rpm)
                log "installing make on photon machine"
                $SSH $TARGET 'tdnf install make -y'
                ;;
            esac

        fi
        $SSH $TARGET "cd $PLUGIN_LOC ; DOCKER_HUB_REPO=$DOCKER_HUB_REPO VERSION_TAG=$VERSION_TAG EXTRA_TAG=$EXTRA_TAG make ${PREFIX}info ${PREFIX}clean ${PREFIX}plugin"
        if [ -z ${PREFIX} ]; then
        managedPluginSanityCheck
        fi
        $SSH $TARGET "cd $PLUGIN_LOC ; DOCKER_HUB_REPO=$DOCKER_HUB_REPO VERSION_TAG=$VERSION_TAG EXTRA_TAG=$EXTRA_TAG make ${PREFIX}push ${PREFIX}clean"
    done
}

function buildwindowsplugin {
    TARGET=root@${IP_LIST[0]}

    log "Compressing source into $PLUGIN_SRC_ZIP..."
    cd $PLUGIN_SRC_DIR
    rm -f $PLUGIN_SRC_ZIP
    git archive --format zip --output $PLUGIN_SRC_ZIP HEAD

    log "Cleaning up older files from $TARGET..."
    $SSH $TARGET "powershell Remove-Item -Recurse -Force $WIN_PLUGIN_SRC_DIR"

    log "Transferring $PLUGIN_SRC_ZIP to $TARGET..."
    scp $PLUGIN_SRC_ZIP $TARGET:$WIN_TEMP_DIR

    log "Extracting $PLUGIN_SRC_ZIP on $TARGET..."
    $SSH $TARGET "powershell Expand-Archive $WIN_TEMP_DIR\\$PLUGIN_SRC_ZIP -DestinationPath $WIN_PLUGIN_SRC_DIR"

    log "Building windows plugin on $TARGET..."
    $SSH $TARGET "cd $WIN_PLUGIN_SRC_DIR && build.bat"

    log "Copying $WIN_PLUGIN_BIN_ZIP_LOC from $TARGET to local..."
    mkdir -p $WIN_BUILD_DIR
    scp $TARGET:$WIN_PLUGIN_BIN_ZIP_LOC $WIN_BUILD_DIR/$PLUGIN_BIN_ZIP

    log "Cleaning up..."
    rm -f $PLUGIN_SRC_ZIP
}

function deploywindowsplugin {
    cd $PLUGIN_SRC_DIR

    for ip in $IP_LIST
    do
        TARGET=root@$ip

        log "Transferring dependencies to $TARGET..."
        scp install-vdvs.ps1 $TARGET:$WIN_TEMP_DIR
        scp $WIN_BUILD_DIR/$PLUGIN_BIN_ZIP $TARGET:$WIN_TEMP_DIR

        log "Installing plugin as a windows service on $TARGET..."
        $SSH $TARGET "powershell -ExecutionPolicy ByPass -File $WIN_TEMP_DIR\\install-vdvs.ps1 file:///$WIN_TEMP_DIR\\$PLUGIN_BIN_ZIP -Force"
    done
}

function managedPluginSanityCheck {
    $SCP $SCRIPTS/plugin_sanity_test.sh $TARGET:$BUILD_LOC
    $SSH $TARGET "sh $BUILD_LOC/plugin_sanity_test.sh \"$PLUGNAME\""
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

function installManagedPlugin {
    if [ $PLUGIN_NAME == $VFILE_PLUGNAME ]
    then
        log "installManagedPlugin: Installing vfile plugin [$MANAGED_PLUGIN_NAME]"
        $SSH $TARGET "docker plugin install --grant-all-permissions --alias $PLUGIN_ALIAS $MANAGED_PLUGIN_NAME VFILE_TIMEOUT_IN_SECOND=300"
    else
        log "installManagedPlugin: Installing vDVS plugin [$MANAGED_PLUGIN_NAME]"
        $SSH $TARGET "docker plugin install --grant-all-permissions --alias $PLUGIN_ALIAS $MANAGED_PLUGIN_NAME"
    fi
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

    # Restart hostd so that esxcli extension file is read
    $SSH $TARGET $HOSTD restart
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

function deployESXInstallForUpgrade {
    $SSH $TARGET $VIB_INSTALL -v $VIB_URL
    if [ $? -ne 0 ]
    then
        log "deployESXInstall: Installation hit an error on $TARGET"
        exit 2
    fi
}

function deployesxforupgrade {
        TARGET=root@$ESX
        log "Deploying to ESX $TARGET"
        deployESXInstallForUpgrade
        deployESXPost
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
        $SSH $TARGET "$RM_RF $TMP_LOC $VMDK_OPS_UNITTEST $VMDK_OPS_CONFIG"
    done
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

function cleanvfile {
    set +e
    for IP in $IP_LIST
    do
        TARGET=root@$IP
        cleanupVFile
    done
}

function cleanupVMPre {
    case $FILE_EXT in
    deb)
        $SSH $TARGET "$DEB_QUERY $PLUGIN_NAME > /dev/null 2>&1"
        if [ $? -eq 0 ]
        then
            let PLUGIN_INSTALLED=0
        else
            let PLUGIN_INSTALLED=1
        fi
        ;;
    rpm)
        $SSH $TARGET "$RPM_QUERY $PLUGIN_NAME > /dev/null 2>&1"
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
        $SSH $TARGET "$IS_SYSTEMD > /dev/null 2>&1"
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
    $SSH $TARGET "docker plugin rm $MANAGED_PLUGIN_NAME -f > /dev/null 2>&1"
}

function cleanupVFile {
    $SSH $TARGET "docker plugin rm $MANAGED_PLUGIN_NAME -f > /dev/null 2>&1"
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
deployvm  "vm-ips"  "managed_plugin_name"
deployvmtest "vm-ips" "folder containing test binaries" "folder containing scripts"
cleanesx  "esx-ips"
cleanvm   "vm_ips" "managed_plugin_name"
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
deployesxforupgrade)
        VIB_URL="$2"
        if [ -z "$VIB_URL" ]
        then
            usage "Missing params: URL to vib file" ;
        fi
        deployesxforupgrade
        ;;
cleanesx)
        cleanesx
        ;;
deployvm)
        MANAGED_PLUGIN_NAME="$2"
        PLUGIN_ALIAS="$3"
        PLUGIN_NAME="$4"
        if [ -z "$MANAGED_PLUGIN_NAME" -o -z "$PLUGIN_ALIAS" ]
        then
            usage "Missing params: managed_plugin_name|plugin_alias"
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
        MANAGED_PLUGIN_NAME="$2"
        if [ -z "$MANAGED_PLUGIN_NAME" ]
        then
            usage "Missing params: managed_plugin_name"
        fi
        cleanvm
        ;;
cleanvfile)
        MANAGED_PLUGIN_NAME="$2"
        if [ -z "$MANAGED_PLUGIN_NAME" ]
        then
            usage "Missing params: managed_plugin_name"
        fi
        cleanvfile
        ;;
buildplugin)
        PLUGIN_BIN="$2"
        MANAGED_PLUGIN_SRC="$3"
        SCRIPTS="$4"
        DOCKER_HUB_REPO="$5"
        VERSION_TAG="$6"
        EXTRA_TAG="$7"
        PLUGNAME="$8"
        PREFIX="$9"
        if [ -z "$PLUGIN_BIN" -o -z "$MANAGED_PLUGIN_SRC" -o -z "$SCRIPTS" ]
        then
            usage "Missing params: plugin/binary/script folder"
        fi
        buildplugin
        ;;
buildwindowsplugin)
        buildwindowsplugin
        ;;
deploywindowsplugin)
        deploywindowsplugin
        ;;
*)
        usage "Unknown function:  \"$FUNCTION_NAME\""
        ;;
esac
