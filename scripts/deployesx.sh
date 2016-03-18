#!/bin/bash
#
# Deploys plugin code on ESX(s) using VIB  
#  bring service scripts and VIB to ESX and reinstalls the VIB
#
# Usage: 
# ./deployesx.sh VIB-name "list-of-ip_addresses"
#

if [ $# -lt 2 ] 
then
   echo Usage: ./deployesx.sh VIB-name \"list-of-ip_addresses\"
   exit 1
fi

# get params
binaries=$1
ip_list=$2

# scripts used in this code:
startsvc=startesx.sh
stopsvc=stopesx.sh

. ./scripts/deploy-helper.sh
