#!/bin/sh
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


# temporary script to run vmci_srv.py detached.
#
# installs the VIB and starts the service. 
# passed location on test , and full binary name on build machine
#
# Logs info:
# The real log is in "/var/log/vmware/vmdk_ops.log" (see vmci_srv.py )
# redirtecting stdio/err to /tmp/plugin.log in case we missed something in logs
# this will be gone when VIB work is complete
# Note that we are resetting actual log here - it is useful in Drone runs. 

vibfile=$1

pylog=/var/log/vmware/vmdk_ops.log
cat /dev/null > $pylog
# long running, so let's always echo
echo "localcli software vib install --no-sig-check  -v $vibfile"
localcli software vib install --no-sig-check  -v $vibfile
localcli --plugin-dir=/usr/lib/vmware/esxcli/int sched group list| grep vmdkops | grep python> /dev/null
/etc/init.d/vmdk-opsd status| grep pid
