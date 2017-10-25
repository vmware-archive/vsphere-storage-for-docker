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

# To sanity test on the same VM using managed plugin
# More information https://github.com/vmware/docker-volume-vsphere/issues/1020#issue-213470358

echo "plugin_sanity_test: [INFO] Running plugin_sanity_test on the clean test setup..."

# get installed plugin name
PLUGNAME=$1
pluginName=`docker plugin ls | sed -n '/'$PLUGNAME'/p' | awk '{ print $2 }'`
echo "plugin_sanity_test: [INFO] Installed plugin name is:$pluginName"

# make sure plugin name is not empty
if [ -z $pluginName ]; then
    echo "plugin_sanity_test: [ERROR] error has occurred while fetching created plugin name..
         please make sure plugin is built correctly or not"
         exit 1
fi

# setting random value for plugin name
volumeName="volume_`date +%Y%m%d%H%M%S`";

# Enable plugin and verifies it is enabled or not
echo "plugin_sanity_test: [INFO] Enable plugin and verifies it is enabled or not..."
docker plugin enable $pluginName
enabled=`docker plugin inspect $pluginName -f {{.Enabled}}`
if [ $? -ne 0  -o  $enabled != "true" ]; then
        echo "[ERROR] plugin_sanity_test: docker plugin inspect failed..[" $enabled "]"
        exit 1
fi

# try to create volume and expects volume creation
echo "plugin_sanity_test: [INFO] try to create volume again and expects volume creation this time as plugin is enabled..."
docker volume create -d $pluginName --name $volumeName
if [ $? -ne 0 ]; then
        echo "[ERROR] plugin_sanity_test: Error occurred while creating volume after plugin enablement"
        exit 1
fi

# setting random value for container name
containerName="container_`date +%Y%m%d%H%M%S`";
fileName="file_`date +%Y%m%d%H%M%S`";

# run busybox and write something to this volume
echo "plugin_sanity_test: [INFO] run busybox and write something to this volume"
docker run --rm -v $volumeName:/vol1 --name=$containerName busybox touch /vol1/$fileName
if [ $? -ne 0 ]; then
        echo "[ERROR] plugin_sanity_test: Error has occurred while writing data to volume"
        exit 1
fi
# run busybox and check the written stuff
echo "plugin_sanity_test: [INFO] run busybox and check the written stuff"
fileExistCount=`docker run --rm -v $volumeName:/vol1 --name=abcC$containerNament busybox sh -c 'ls -1 /vol1/'$fileName' | wc -w'`
echo "plugin_sanity_test: [INFO] fileExistCount="$fileExistCount

if [ $fileExistCount -ne 1 ]; then
    echo "[ERROR] plugin_sanity_test: writing data to volume verification failed"
    exit 1
fi

# check the volume indeed belongs to the plugin (Not local)
echo "plugin_sanity_test: [INFO] check the volume indeed belongs to the plugin (Not local)..."
docker_driver=`docker volume inspect $volumeName -f {{.Driver}}`
if [ $? -ne 0  -o  $docker_driver != $pluginName ]; then
        echo "[ERROR] plugin_sanity_test: docker plugin inspect failed..[" $docker_driver "]"
        exit 1
fi

# rm the volume
echo "plugin_sanity_test: [INFO] rm the volume..."
docker volume rm $volumeName
if [ $? -ne 0 ]; then
        echo "[ERROR] plugin_sanity_test: Error occurred while removing volume"
        exit 1
fi

# disable the plugin and remove it (or rm -rf or just make clean in ./plugin)
echo "plugin_sanity_test: [INFO] disable the plugin..."
docker plugin disable $pluginName
enabled=`docker plugin inspect $pluginName -f {{.Enabled}}`
if [ $? -ne 0  -o  $enabled != "false" ]; then
        echo "[ERROR] plugin_sanity_test: docker plugin inspect failed..[" $enabled "]"
#        exit 1
fi

#remove it (or rm -rf or just make clean in ./plugin) will be done through deploy-tools.sh
