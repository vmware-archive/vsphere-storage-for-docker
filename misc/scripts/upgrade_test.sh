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

    source ../misc/scripts/commands.sh
    MANAGED_PLUGIN_NAME="vsphere:latest"
    E2E_Tests="github.com/vmware/docker-volume-vsphere/tests/e2e"
    GO="go"
    DEPLOY_TOOLS_SH=../misc/scripts/deploy-tools.sh

    get_vib_url() {
        echo "Get version $1"
        MATCH_ENTRY=$( grep $1 ../misc/scripts/upgrade_test_vib.txt)
        echo $MATCH_URL_ENTRY
        IFS=' ' read -a match_array <<< "${MATCH_ENTRY}"
        VIB_URL=${match_array[1]}
    }

    VIB_URL=""
    get_vib_url $UPGRADE_FROM_VER
    BASE_VIB_URL=$VIB_URL
    echo "BASE_VIB_URL=$BASE_VIB_URL"

    get_vib_url $UPGRADE_TO_VER
    UPGRADE_VIB_URL=$VIB_URL
    echo "UPGRADE_VIB_URL=$UPGRADE_VIB_URL"

    echo "Upgrade test: from ver $UPGRADE_FROM_VER to ver $UPGRADE_TO_VER"

	echo "Upgrade test step 1: deploy on $ESX with $BASE_VIB_URL"
	$DEPLOY_TOOLS_SH deployesxforupgrade $ESX $BASE_VIB_URL

	echo "Upgrade test step 2.1: remove plugin $MANAGED_PLUGIN_NAME on $VM1"
	$DEPLOY_TOOLS_SH cleanvm $VM1 $MANAGED_PLUGIN_NAME

	echo "Upgrade test step 2.2: deploy plugin vmware/docker-volume-vsphere:$UPGRADE_FROM_VER on $VM1"
	../misc/scripts/deploy-tools.sh deployvm $VM1 vmware/docker-volume-vsphere:$UPGRADE_FROM_VER
	$SSH $VM1 "systemctl restart docker || service docker restart"

	echo "Upgrade test step 3: run pre-upgrade test"
	$GO test -v -tags runpreupgrade $E2E_Tests

	echo "Upgrade test step 4: deploy on $ESX with $UPGRADE_VIB_URL"
	$DEPLOY_TOOLS_SH deployesxforupgrade  $ESX $UPGRADE_VIB_URL

	echo "Upgrade test step 5.1: remove plugin $MANAGED_PLUGIN_NAME on $VM1"
	$DEPLOY_TOOLS_SH cleanvm $VM1 $MANAGED_PLUGIN_NAME

	echo "Upgrade test step 5.2: deploy plugin vmware/docker-volume-vsphere:$UPGRADE_TO_VER on $VM1"
	../misc/scripts/deploy-tools.sh deployvm $VM1 vmware/docker-volume-vsphere:$UPGRADE_TO_VER
	$SSH $VM1 "systemctl restart docker || service docker restart"

	echo "Upgrade test step 6: run post-upgrade test"
	$GO test -v -tags runpostupgrade $E2E_Tests