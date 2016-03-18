#!/bin/bash
#
# Deploys docker-vmdk-plugin and tests to a set of Lunix Guest VMs.
#   Also stops and starts services as needed.
#
# Plugin and test binaries are copied to bin_remote
# All scripts from ./scripts are copied to $tmp_loc (location hardcoded)
#
# Usage: 
# ./deployvm.sh "list-of-binaries" bin-remote "list-of-ip_addresses"
#  list-of-binaries is a space separated list of local binaries to copy to vm
#  bin-remote is where to place them on remote linux vm
# list-of-ip_addresses is a space separated list of  of vms to deploy to 
# 

if [ $# -lt 3 ] 
then
   echo Usage: ./deployvm.sh \"list-of-binaries\" bin-remote \"list-of-ip_addresses\"
   exit 1
fi

export binaries=$1
export bin_remote=$2
export ip_list=$3

# scripts used in this code:
export startsvc=startvm.sh
export stopsvc=stopvm.sh

. ./scripts/deploy-helper.sh


SSH="$E ssh -o StrictHostKeyChecking=no"
echo "  Test Docker TCP connection..."
for ip in $ip_list
do
    echo "  $SSH root@$ip docker -H tcp://$ip:2375 ps > /dev/null"
    $SSH root@$ip docker -H tcp://$ip:2375 ps > /dev/null
done
