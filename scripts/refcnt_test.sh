#!/bin/bash
#
# A simple test to validate refcounts.
#
# Creates $count running containers using a VMDK volume, checks refcount
# by grepping the log (assumes DEBUG log level), touches files within and
# checks the files are all there, Then removes the containers and the volume
#
# *** Caveat: at exit, it kills all containers and cleans all volumes on the box !
# It should just accumulate the names of containers and volumes to clean up.
#
# It should eventually be replaced with a proper test in ../refcnt_test.go.
# For now (TP) we still need basic validation
#
# It also should do crash validation :
# - restart plugin and check all in place
# - kill -9 docker ,then  restart docker and check all is clean
#

log=/var/log/docker-vmdk-plugin.log
count=5
vname=vol1
mount=/mnt/vmdk/$vname

function cleanup {
   to_kill=`docker ps -q`
   to_rm=`docker ps -a -q`
   if [ -n "$to_kill" -o -n "$to_rm" ]
   then
      echo "Cleaning up"
   fi
   if [ -n "$to_kill" ] ; then $DOCKER kill $to_kill > /dev/null ; fi
   if [ -n "$to_rm" ] ; then $DOCKER rm $to_rm > /dev/null; fi
}

trap cleanup EXIT

DOCKER="$DEBUG docker"
GREP="$DEBUG grep"

# Now start the test

echo "Testing refcounts..."

echo "Creating a volume and $count containers using it"
$DOCKER volume create --driver=vmdk --name=$vname
for i in `seq 1 $count`
do
  $DOCKER run -d -v $vname:/v busybox sh -c "touch /v/file$i; sleep 60"
done

echo "Checking volume content"
files=`$DOCKER run -v $vname:/v busybox sh -c 'ls -1 /v/file*'`
c=`echo $files | wc -w`
echo "Found $c files. Expected $count"
if [ $c -ne $count ] ; then
   echo FAILED CONTENT TEST - cannot file enough files in /$vname/file\*
   echo files: $files
   exit 1
fi

echo "Checking the last refcount and mount record"
last_line=`tail -1 /var/log/docker-vmdk-plugin.log`
echo $last_line | $GREP -q refcount=$count
status=$?
if $? -eq 0 ; then
   echo FAILED REFCOUNT TEST, status $status
   echo Last line in the log: \'$last_line\'
   echo missing pattern "refcount=$count"
   return 2
fi

$GREP -q $mount /proc/mounts || (echo "FAILED MOUNT TEST 1"; exit 3)

# should fail 'volume rm', so checking it
echo "Checking 'docker volume rm'"
$DOCKER volume rm $vname 2> /dev/null && (echo "FAILED DOCKER RM TEST 1"; exit 4)

# <TBD>
# if we restart the plugin, all the counts should be as is
# kill -9 `pidof docker-vmdk-plugin'
# /usr/local/bin/docker-vmdk-plugin &
# grep last 5 lines of the log for 'volume=$vname refcnt=$count"
# </tbd>

cleanup

echo "Checking that the volume is unmounted and can be removed"
$DOCKER volume rm $vname || (echo "FAILED DOCKER RM TEST 2"; exit 5)
$GREP -q $mount /proc/mounts && (echo "FAILED MOUNT TEST 2"; exit 6)

echo "TEST PASSED."
exit 0
