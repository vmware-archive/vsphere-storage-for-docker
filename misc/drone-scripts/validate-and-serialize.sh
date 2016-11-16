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

unit_test_array=($TEST_URL_ARRAY)
numServers=${#unit_test_array[@]}
DRONE_BUILD_NUMBER=${DRONE_BUILD_NUMBER:=0}
prevBuildStatus=`drone build info vmware/docker-volume-vsphere $(( $DRONE_BUILD_NUMBER-$numServers ))`
outArray=($prevBuildStatus)
total_builds=`drone build list vmware/docker-volume-vsphere| wc -l`

govc datastore.mkdir docker-volume-vsphere/$DRONE_BUILD_NUMBER
if [ "$?" != "0" ]
then
    echo
    echo
    echo "Restart not supported"
    echo "Record already exists for build, push a new commit to trigger build"
    echo
    echo
    exit 1
fi

# Delete older build records
govc datastore.rm docker-volume-vsphere/$(( $DRONE_BUILD_NUMBER-$total_builds )) 2>&1 > /dev/null

while [[ ${outArray[2]} == *"running"* ]]; do
    echo "Waiting 5 minutes for previous build $(( $DRONE_BUILD_NUMBER-$numServers )) to complete";
    sleep 300;
    prevBuildStatus=`drone build info vmware/docker-volume-vsphere $(( $DRONE_BUILD_NUMBER-$numServers ))`
    outArray=($prevBuildStatus)
done

# Check if any other build is in the ongoing folder, if present, check if entry is stale.
# If entry is stale clean it up.
# If entry is an on going build wait for it.
prevBuild=`govc datastore.ls docker-volume-vsphere/ongoing|tail -n 1`
while [[ "$prevBuild" != "" ]];
do
    prevBuildStatus=`drone build info vmware/docker-volume-vsphere $prevBuild`
    outArray=($prevBuildStatus)
    if [[ ${outArray[2]} != *"running"* ]]
    then
        govc datastore.rm docker-volume-vsphere/ongoing/$prevBuild
    else
        echo "Waiting 5 minutes for previous build $prevBuild to complete";
        sleep 300;
    fi
    prevBuild=`govc datastore.ls docker-volume-vsphere/ongoing|tail -n 1`
done

govc datastore.mkdir docker-volume-vsphere/ongoing/$DRONE_BUILD_NUMBER

exit 0
