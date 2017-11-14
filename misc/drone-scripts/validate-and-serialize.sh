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

# datastore name refers to storage backed by base ESX (physical ESX) where
# ongoing/<build_no> contains the directory named after the currently running
# CI/ build
DS='-ds=datastore1'

# lock names, to use in lock/unlock
# build_lock:
#       Overall lock makes sure only one thread at a time runs
# check_lock:
#       Represents a lock to perform cleanup leftover from previous run builds.
build_lock="build_lock"
check_lock="check_lock"

# refers current CI build#
DRONE_BUILD_NUMBER=${DRONE_BUILD_NUMBER:=0}

# wait time in seconds
interval=90

# Decide whether cleanup is needed or not.
# returns true if the ongoing/build info is stale and the build is not running
# anymore, or if the user wants to restart the build so the ongoing/build and
# current build have the same ID
function is_cleanup_needed {
    # Retrieve on-going build information
    ongoingBuildInfo=`drone build info vmware/docker-volume-vsphere $1`
    buildInfoObj=($ongoingBuildInfo)

    # verifies status value OR request is resulted from *Restart* event
    if [[ ${buildInfoObj[2]} != *"running"* || $1 == $DRONE_BUILD_NUMBER ]]
    then
        return 0
    else
        return 1
    fi
}

# Util to release lock
function release_lock {
    if [ -z "$1" ] ;
    then
        echo "Missing argument in release_lock" ;
        echo "USAGE: release_lock <lock_name>"
        return 2;
    fi
    govc datastore.rm $DS docker-volume-vsphere/$1
    return $?
}

# Util to create directory which in turns consumed as lock
function acquire_lock {
    if [ -z "$1" ] ;
    then
        echo "Missing argument in acquire_lock" ;
        echo "USAGE: acquire_lock <lock_name>"
        return 2;
    fi
    govc datastore.mkdir $DS docker-volume-vsphere/$1 2>/dev/null > /dev/null
    return $?
}

# First it tries acquire lock "build-start-lock" which makes sure only
# one thead (test run request gets this opportunity ) and other are
# being blocked and keeping polling at definite interval.
# Whichever thread gets success acquiring the "build-start-lock" lock
# moves forward with kicking off the test run.
while ! acquire_lock $build_lock
do
    # take build check lock
    if acquire_lock $check_lock
    then
        # Grab ongoing build information from drone.
        runningBuild=`govc datastore.ls $DS docker-volume-vsphere/ongoing`
        runningBuildArr=($runningBuild)
        # Let's check cleanup is needed or not
        # cleans up if return value is 0 otherwise not; checking build is finished or not
        if is_cleanup_needed ${runningBuildArr[0]}
        then
            echo "Cleaning stale data..."
            govc datastore.rm $DS docker-volume-vsphere/ongoing/${runningBuildArr[0]}
            release_lock $check_lock
            break
        else
            # Let's release build_check_lock
            release_lock $check_lock
        fi
    fi
    # wait for some time to poll again
    echo "Waiting $interval seconds for build ${runningBuildArr[0]} to complete";
    sleep $interval;
done

govc datastore.mkdir $DS docker-volume-vsphere/ongoing/$DRONE_BUILD_NUMBER
echo "$DRONE_BUILD_NUMBER is added to ongoing folder"
exit 0