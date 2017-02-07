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

echo ESX 6.5
until govc vm.ip $ESX_6_5
do
    echo "Waiting for revert to complete for ESX6.5";
    sleep 1;
done

echo ESX 6.0
until govc vm.ip $ESX_6_0
do
    echo "Waiting for revert to complete for ESX6.0";
    sleep 1;
done

echo "Reset complete"
sleep 5;
echo "Resume testing"

exit 0
