#!/bin/bash
# Copyright 2017 VMware, Inc. All Rights Reserved.
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

# This script is used to set up a swarm cluster which is needed by vFile
# More information:https://github.com/vmware/vsphere-storage-for-docker/tree/master/docs/external/vfile-plugin.md#installation

source commands.sh
filename=$1

i=0
idx=0
while IFS= read -r line || [ -n "$line" ]
do
    if [ $i == 0 ]
    then
        # NODE_COUNT is the total number of nodes that in the swarm cluster
        if [[ $line =~ ^NODE_COUNT=-?[0-9]+$ ]]
        then
            NODE_COUNT="${line#*=}"
            echo "NODE_COUNT=$NODE_COUNT"
        else
            echo "Invalid value for NODE_COUNT:$line"
            exit 1
        fi
    fi
    if [ $i == 1 ]
    then
        # MGR_COUNT is the total number of manager nodes in the swarm cluster
        if [[ $line =~ ^MGR_COUNT=-?[0-9]+$ ]]
        then
            MGR_COUNT="${line#*=}"
            echo "MGR_COUNT=$MGR_COUNT"
        else
            echo "Invalid value for MGR_COUNT:$line"
            exit 1
        fi
    fi

    if [ "$i" -gt "1" ]
    then
        # Read IP address in to array
        # In the configuration file, the first $MGR_COUNT line of IP address
        # will be the IP address of swarm manager node
        IP_ADDRESS[idx]=$line
        idx=$((idx+1))
    fi
    i=$((i+1))
done <$1

IP_COUNT=$idx

echo "IP_COUNT $IP_COUNT"

 if [ "$MGR_COUNT" -gt "$NODE_COUNT" ]
 then
     echo "Total number of nodes cannot be smaller than the total number of manager nodes"
     exit 1
 fi

if [ $((MGR_COUNT%2)) -eq 0 ]
then
    echo "Total number of manager nodes in the swarm cluster cannot be a even number"
    exit 1
fi

if [ "$MGR_COUNT" -gt "7" ]
then
    echo "Total number of manager in the swarm cluster is too big"
    exit 1
fi

if [ $NODE_COUNT != $IP_COUNT ]
then
    echo "Total number of nodes does not match the number of IP addresses"
    exit 1
fi

echo "Swarm Cluster Setup Start"

echo "======> Initializing first swarm manager ..."
$SSH root@${IP_ADDRESS[0]}  "docker swarm init"

# Fetch Tokens
ManagerToken=`$SSH root@${IP_ADDRESS[0]} docker swarm join-token manager | grep token`
WorkerToken=`$SSH root@${IP_ADDRESS[0]} docker swarm join-token worker | grep token`

echo "Manager Token: ${ManagerToken}"
echo "Workder Token: ${WorkerToken}"

# Add remaining manager to swarm
echo "======> Add other manager nodes"
for i in `seq 1 $((MGR_COUNT-1))`
do
    echo "node with IP ${IP_ADDRESS[$i]} joins swarm as a Manager"
    $SSH root@${IP_ADDRESS[$i]} ${ManagerToken}
done

# Add worker to swarm
echo "======> Add worker nodes"
for i in `seq $((MGR_COUNT)) $((NODE_COUNT-1))`
do
     echo "node with IP ${IP_ADDRESS[$i]} joins swarm as a Worker"
     $SSH root@${IP_ADDRESS[$i]} ${WorkerToken}
done

# list nodes in swarm cluster
$SSH root@${IP_ADDRESS[0]} "docker node ls"

echo "Swarm Cluster Setup Complete"
