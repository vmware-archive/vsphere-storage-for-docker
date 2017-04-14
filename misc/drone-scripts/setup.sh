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

echo "Resetting testbed"
govc snapshot.revert -vm $ESX_6_5 init
govc snapshot.revert -vm $ESX_6_0 init

echo "Waiting for revert to complete";

DIR=$(dirname ${BASH_SOURCE[0]})
. $DIR/../scripts/wait_for.sh

# Threshold to time out
retryCount=30

echo ESX 6.5
wait_for "govc vm.ip $ESX_6_5" $retryCount

echo ESX 6.0
wait_for "govc vm.ip $ESX_6_0" $retryCount

echo "Reset complete"

# setting environment variables pointing to ESX6.5.

#This part is needed and keep it latest whenever new VMs are added to
#ESX6.5 or removed. CI is running tests against ESX6.5 (docker host resided
#on vmfs) very first hence the retry mechanism added for the docker host
#exist on vmfs datastore.

export GOVC_URL=$GOVC_URL_6_5
export GOVC_USERNAME=$GOVC_USERNAME_ESX
export GOVC_PASSWORD=$GOVC_PASSWORD_ESX

echo "Wait for VM to get ready"

wait_for "$GOVC_GET_IP photon.vmfs" $retryCount
wait_for "$GOVC_GET_IP Ubuntu.16.10" $retryCount

echo "Resume testing"

exit 0
