#!/bin/bash
#
# deploy-tools.sh
# 
# Has a set of functions to deploy to ESX and guests, start and stop services 
# and clean up
#
# Usage:
#       ./scripts/deploy-tools function-name function-params
#
# e.g.
#      ./scripts/deploy-tools deployvm "ip-addresses" "target-bin" "binaries"
#

# on failure, exit right away 
set -e

# ==== set globals ====


# define what we use for SCP and SSH commands
#
# Note: DEBUG is debug assistance, i.e. if DEBUG is empty/not defined, 
# the actual command is executed.  If 'DEBUG=echo' is typed before 'make', 
# then instead of command (e.g. scp) "echo scp" will be executed so it will 
# print out the commands. Super convenient for debugging.
SCP="$DEBUG scp -q -o StrictHostKeyChecking=no"
SSH="$DEBUG ssh -o StrictHostKeyChecking=no"

# consts

# We remove VIB by internal name, not file name. See description.xml in VIB
internal_vib_name=vmware-esx-vmdkops-service

script_loc=./scripts
tmp_loc=/tmp/docker-volume-plugin
guest_mount_point=/mnt/vmdk

# ====== define functions =======

# deploy_helper
#
# deploys scripts to fixed remote location, copies binaries and starts services
#       Params: startsv-script-name stopsvc-script-name
#
# Relies on globals to pass all info in

function deploy_helper {
   for ip in $ip_list
   do
        target=root@$ip
   
        echo Deploying to $target...
   
        echo "  Copy Helper Scripts..."
        $SSH $target mkdir -p $tmp_loc
        $SCP $script_loc/* $target:$tmp_loc
   
        echo "  Stop services using $stopsvc..."
        $SSH $target  $tmp_loc/$stopsvc $service_name
   
        echo "  Copy Binaries..."
        for file in $files
        do
                $SCP $file $target:$bin_remote_location
        done
   
        echo "  Start Services using $startsvc..."
        $SSH $target $tmp_loc/$startsvc $service_file

   done
}


# deployvm 
#
# Deploys docker-vmdk-plugin and tests to a set of Lunix Guest VMs.
#   Also stops and starts services as needed.
#
# Plugin and test binaries (mentioned in $files) are copied to $bin_remote_location
# for each ip in $ip_list
#
# All scripts from ./scripts are copied to $tmp_loc (location hardcoded)

function deployvm {
   startsvc=startvm.sh
   stopsvc=cleanvm.sh
   
   # hardcoded since it'll be dropped in favor of apt-get anyways:
   service_file=$bin_remote_location/docker-vmdk-plugin
   service_name=$service_file

   deploy_helper

   echo "  Test Docker TCP connection..."
   for ip in $ip_list
   do
      echo "  $SSH root@$ip docker -H tcp://$ip:2375 ps > /dev/null"
      $SSH root@$ip docker -H tcp://$ip:2375 ps > /dev/null
   done
}


# deployesx
#
# Deploys plugin code on ESX(s) using VIB mentioned in $files :
#       Copies service scripts and VIB to ESX and reinstalls the VIB

function deployesx {
   startsvc=startesx.sh
   stopsvc=cleanesx.sh
   # we support only 1 VIB file, so treat $files as a single name
   service_file="$bin_remote_location/$(basename $files)"
   service_name=$internal_vib_name

   deploy_helper
}


# cleanesx
#
# Does vib and tmp files cleanup on ESX

function cleanesx {
    stopsvc=cleanesx.sh
 
    for ip in $ip_list
    do
        echo "Cleaning up on ESX $ip..."
        target=root@$ip
        $SSH $target mkdir -p $tmp_loc
        $SCP $script_loc/$stopsvc $target:$tmp_loc
        $SSH $target $tmp_loc/$stopsvc $internal_vib_name
        $SSH $target "rm -rf $tmp_loc"
    done
}


# cleanvm
#
# Does vib and tmp files cleanup on Linux guests

function cleanvm {
   remote_binaries=`for f in $files ; do echo "$bin_remote_location/$(basename $f)"; done`
   stopsvc=cleanvm.sh
   # We kill process by name:
   name=docker-vmdk-plugin

   for ip in $ip_list
   do
        echo "Cleaning up on VM $ip..."
        target=root@$ip
        $SSH $target mkdir -p $tmp_loc
        $SCP $script_loc/$stopsvc $target:$tmp_loc

        echo "   Asking docker to remove volumes ($volumes)..."
        # make sure docker engine is not hanging due to old/dead plugins
        $SSH $target service docker restart
        # and now clean up
        for vol in $volumes
        do
           $SSH $target "if docker volume ls | grep -q $vol; then \
                        docker volume rm $vol; fi "
        done
        
        echo "   Stopping services..."
        $SSH $target $tmp_loc/$stopsvc $name
        $SSH $target "rm -rf $tmp_loc"
   
        echo "   Removing binaries and restarting docker..."
        $SSH $target rm -f $remote_binaries
        $SSH $target rm -rvf $guest_mount_point/$test_vol
        $SSH $target rm -rvf /tmp/docker-volumes/
        $SSH $target service docker restart
   done
}


#
# define usage message

function usage {
   echo $1
   echo <<EOF
deploy-tools.sh provideds a set of helpers for deploying docker-vmdk-plugin
binaries to ESX and to guest VMs. 

Usage:  deploy-tools.sh command params...

Comands and params are as follows: 
deployesx "esx-ips"  vib-file-name
deployvm  "vm-ips"  "binaries-to-deploy" bin-remote-location 
cleanesx  "esx-ips" 
cleanvm   "vm_ips"  "binaries-to-clean"  bin-remote-location "test-volumes-to-clean"
EOF

    exit 1
}

# ============= "main" ===============
#
# Check params, and call the requested function 


# first param is always function name to invoke
function_name=$1 ; shift

# globals used in misc. functions instead of params (to avoid extra parse)
ip_list="$1"
files="$2"
bin_remote_location="$3"
volumes="$4"

# check that all params are present:
if [ -z "$function_name" -o  -z "$ip_list" ]
then 
   usage "Missing parameters: need at least \"func-name ipaddr\""
fi


case $function_name in
deployesx)
        if [ -z "$files" ] ; then usage "Missing params: vib-name" ; fi
        bin_remote_location=$tmp_loc  # copy VIB here, and install from here
        deployesx
        ;;
cleanesx)
        cleanesx
        ;;
deployvm)
        if [ -z "$files" -o -z "$bin_remote_location" ] ; then usage "Missing params: files or rem_location" ; fi
        deployvm
        ;;
cleanvm)
        if [ -z "$files" -o -z "$bin_remote_location" -o -z "$volumes" ] ; then usage "Missing params: files, bin_remote_location or volume"; fi
        cleanvm
        ;;
*)
        usage "Unknown function:  \"$function_name\""
        ;;
esac

