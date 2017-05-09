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


# A simple test to validate refcounts.

# Creates $count running containers using a VMDK volume, checks refcount
# by grepping the log, touches files within and
# checks the files are all there, Then removes the containers and the volume
#
# *** Caveat: at exit, it kills all containers and cleans all volumes on the box !
# It should just accumulate the names of containers and volumes to clean up.
#
# It should eventually be replaced with a proper test in ../refcnt_test.go.
# For now (TP) we still need basic validation
#

DIR=$(dirname ${BASH_SOURCE[0]})
. $DIR/wait_for.sh

log=/var/log/docker-volume-vsphere.log
count=5
vname=refCountTestVol
mount=/mnt/vmdk/$vname
timeout=180

function cleanup_containers {
   to_kill=`docker ps -q`
   to_rm=`docker ps -a -q`

   if [ -n "$to_kill" -o -n "$to_rm" ]
   then
      echo "Cleaning up containers"
   fi
   if [ -n "$to_kill" ] ; then $DOCKER kill $to_kill > /dev/null ; fi
   if [ -n "$to_rm" ] ; then $DOCKER rm $to_rm > /dev/null; fi
}

function cleanup {
   cleanup_containers
   $DOCKER volume rm $vname
}
trap cleanup EXIT

function check_files {
    # Check how many files we see in the still-mounted volume
    files=`$DOCKER run -v $vname:/v busybox sh -c 'ls -1 /v/file*'`
    c=`echo $files | wc -w`
    echo "Found $c files. Expected $count"
    if [ $c -ne $count ] ; then
       echo files: \"$files\"
       return 1
    fi
    return 0
}

function check_recovery_record {
    # log contains refcounting attempts and after success logs summary.
    line=`tail -50 /var/log/docker-volume-vsphere.log | $GREP 'Volume name=' | $GREP 'mounted=true'`
    expected="count=$count mounted=true"

    echo $line | $GREP "$vname" | $GREP -q "$expected" ; if [ $? -ne 0 ] ; then
       echo Found:  \"$line\"
       echo Expected pattern: \"$expected\"
       return 1
    fi
    return 0
}

function test_crash_recovery {
    timeout=$1
    echo "Checking recovery through docker kill"
    # kill docker daemon forcefully
    pkill -9 dockerd
    until pids=$(pidof dockerd)
    do
        echo "Waiting for docker to restart"
        sleep 1
    done

    echo "Waiting for plugin init"
    sleep 5
    sync  # give log the time to flush
    wait_for check_recovery_record $timeout
    if [ "$?" -ne 0 ] ; then
        echo DOCKER RESTART TEST FAILED. Did not find proper recovery record
        exit 1
    fi
}

DOCKER="$DEBUG docker"
GREP="$DEBUG grep"

# Now start the test

echo "Testing refcounts..."

echo "Creating volume $vname and $count containers using it"
$DOCKER volume create --driver=vsphere:latest --name=$vname -o size=1gb
if [ "$?" -ne 0 ] ; then
   echo FAILED TO CREATE $vname
   exit 1
fi

echo "$(docker volume ls)"
for i in `seq 1 $count`
do
  # run containers with restart flag so they restart after docker restart
  $DOCKER run -d --restart=always -v $vname:/v busybox sh -c "touch /v/file$i; sync ; \
      while true; do sleep $timeout; done"
done

echo "Checking the last refcount and mount record"
last_line=`tail -1 /var/log/docker-volume-vsphere.log`
echo $last_line | $GREP -q refcount=$count ; if [ $? -ne 0 ] ; then
   echo FAILED REFCOUNT TEST - pattern  \"refcount=$count\" not found
   echo Last line in the log: \'$last_line\'
   echo Expected pattern \"refcount=$count\"
   exit 1
fi

'
Disabling this check due to race. See issue #1112
echo "Checking volume content"
wait_for check_files $timeout
if [ "$?" -ne 0 ] ; then
   echo FAILED CONTENT TEST - not enough files in /$vname/file\*
   exit 1
fi
'

$GREP -q $mount /proc/mounts ; if [ $? -ne 0 ] ; then
   echo "FAILED MOUNT TEST 1"
   echo \"$mount\" is not found in /proc/mounts
   exit 1
fi

# should fail 'volume rm', so checking it
echo "Checking 'docker volume rm'"
$DOCKER volume rm $vname 2> /dev/null ; if [ $? -eq 0 ] ; then
   echo FAILED DOCKER RM TEST
   echo  \"docker volume rm $vname\" was expected to fail but succeeded
   exit 1
fi

test_crash_recovery $timeout

# kill containers but keep the volume around
cleanup_containers

echo "Checking that the volume is unmounted and can be removed"
$DOCKER volume rm $vname ; if [ $? -ne 0 ] ; then
   echo "FAILED DOCKER RM TEST 2"
   echo \"$DOCKER volume rm $vname\" failed but expected to succeed
   exit 1
fi

$GREP -q $mount /proc/mounts ; if [ $? -eq 0 ] ; then
   echo "FAILED MOUNT TEST "
   echo \"$mount\" found in /proc/mount for an unmounted volume
   exit 1
fi

echo "TEST PASSED."
exit 0

