#!/bin/sh
# Do vib and tmp files cleanup on ESX
# ESX IPs passed in $1

stopsvc=stopesx.sh

SCP="$E scp -o StrictHostKeyChecking=no"
SSH="$E ssh -o StrictHostKeyChecking=no"

script_loc=./scripts
tmp_loc=/tmp/docker-volume-plugin


for ip in $1
do
   echo "Cleaning up on ESX $ip..."
   target=root@$ip
   $SSH $target mkdir -p $tmp_loc
   $SCP $script_loc/$stopsvc $target:$tmp_loc
   $SSH $target $tmp_loc/$stopsvc
   $SSH $target "rm -rf $tmp_loc"
done
