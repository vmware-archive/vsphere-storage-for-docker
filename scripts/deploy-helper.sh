
# deploy helper - 'sourced' from deploy-*.sh
# need to set binaries, bin_remote (optional), ip_list, startsvc and stopsvc
# deploys ./scripts to fixed remote locaiton, then copies files and starts services

# on failure, exit right away 
set -e

# fixed locations:
script_loc=./scripts
tmp_loc=/tmp/docker-volume-plugin

# check the needed info is provided
if [ -z "$binaries" ]  ; then echo binaries need to be defined ; exit 1; fi
if [ -z "$bin_remote" ]  ; then bin_remote=$tmp_loc; fi
if [ -z "$ip_list" ]  ; then echo ip_list need to be defined ; exit 1; fi

# commands to use 
# Note: E is debug assistance, e.g.  "E=echo make deploy-vm"
SCP="$E scp -q -o StrictHostKeyChecking=no"
SSH="$E ssh -o StrictHostKeyChecking=no"

for ip in $ip_list
do
   target=root@$ip
   
   echo Deploying to $target...
   
   echo "  Copy Helper Scripts..."
   $SSH $target mkdir -p $tmp_loc
   $SCP $script_loc/* $target:$tmp_loc
   
   echo "  Stop services ..."
   $SSH $target  $tmp_loc/$stopsvc $binaries
   
   echo "  Copy Binaries..."
   for file in $binaries
   do
      $SCP $file $target:$bin_remote
   done
   
   echo "  Start Services..."
   $SSH $target $tmp_loc/$startsvc $bin_remote "$binaries"

done
