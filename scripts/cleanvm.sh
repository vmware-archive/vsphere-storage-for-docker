#!/bin/bash
# Do vib and tmp files cleanup on Linux guests

binaries=$1
bin_remote=$2
test_vol=$3
ips=$4

#set -x 

remote_bins=`for f in $binaries ; do echo "$bin_remote/$(basename $f)"; done`
stopsvc=stopvm.sh

SCP="$E scp -o StrictHostKeyChecking=no"
SSH="$E ssh -o StrictHostKeyChecking=no"

script_loc=./scripts
tmp_loc=/tmp/docker-volume-plugin

for ip in $ips
do
   echo "Cleaning up on VM $ip..."
   target=root@$ip
   $SSH $target mkdir -p $tmp_loc
   $SCP $script_loc/$stopsvc $target:$tmp_loc
   $SSH $target $tmp_loc/$stopsvc
   $SSH $target "rm -rf $tmp_loc"
   
   echo "   Removing volumes and restarting docker..."
   vibfile="$1/$(basename $2)"
   $SSH $target rm -f $remote_bins
   $SSH $target rm -rvf /mnt/vmdk/$test_vol
   $SSH $target "if docker volume ls | grep -q $test_vol; then \
                docker volume rm $test_vol; fi "
   $SSH $target rm -rvf /tmp/docker-volumes/
   $SSH $target service docker restart
done
