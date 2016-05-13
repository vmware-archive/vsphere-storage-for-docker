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


name=$1

if localcli software vib list | grep -q $name ;
then
   # long running, so let's always echo
   echo "localcli software vib remove --vibname $name" 
   localcli software vib remove --vibname $name
   localcli --plugin-dir=/usr/lib/vmware/esxcli/int sched group list --group-path=host/vim/vimuser/cnastorage/ > /dev/null 2>&1
   if [ $? -eq 0 ];
   then
     echo "Failed to clean up resource groups!"
     exit 1
   fi
fi

