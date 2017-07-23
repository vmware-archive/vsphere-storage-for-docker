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

set -x

GOVC=govc
CURL=curl

SSH_PASS=/usr/bin/sshpass
SSH="$SSH_PASS -p $TEST_PASSWORD ssh -q -kTax -o StrictHostKeyChecking=no"
SSH_PUB_KEY=$HOME/.ssh/id_rsa.pub
SSH_AUTH_KEYS=/root/.ssh/authorized_keys

ROOT=root

DEFAULT_TMPDIR="/tmp"
TEST_VM_OVA="test-vm.ova"
TEST_VM1="$TEST_USER-vdvstest-master"
TEST_VM2="$TEST_USER-vdvstest-worker1"
TEST_VM3="$TEST_USER-vdvstest-worker2"
TEST_VM_TEMPLATE="$TEST_USER-vdvstest-template"

LINKED_CLONE="vm.create -ds=$TEST_VMDATASTORE -disk $TEST_VM_TEMPLATE/$TEST_VM_TEMPLATE.vmdk -disk-datastore=$TEST_VMDATASTORE -link"
VMS=($TEST_VM1 $TEST_VM2 $TEST_VM3)
TEST_VMS=()
TEST_VM_IPS=()

MANIFEST=$PWD/.testvms

DOCKER=/usr/local/bin/docker
DOCKER_INFO="$DOCKER info"
DOCKERD_PORT=2377
DOCKER_INSTALL_COMMON="curl -fsSL https://get.docker.com/ | sh"
DOCKER_INSTALL_TDNF="tdnf -y install docker"

fetch_ova()
{
	tmp=$DEFAULT_TMPDIR

	ova_fpath=$tmp/$TEST_VM_OVA
	echo "Fetching $TEST_OVAURL ......" 
	$CURL --anyauth -#L $TEST_OVAURL -o $ova_fpath
}

# Delete all test VMs
delete_vms()
{
	for n in ${TEST_VMS[@]}; do
		echo "Deleteing VM $n"
		$GOVC vm.destroy $n
	done
	$GOVC vm.destroy $TEST_VM_TEMPLATE
}

# Deploy the test worker VMs to the ESX host and power on 
# the VMs.
deploy_vms()
{
	echo "Checking if VM $TEST_VM_TEMPLATE already exists...."
	info=`$GOVC vm.info $TEST_VM_TEMPLATE|wc -l`
	if [ $info -eq 0 ]; then
		echo "Importing OVA $1 as VM $vmname on datastore $TEST_VMDATASTORE"
		if [ `echo $1|grep ova` ] || [ $? -eq 0 ]; then
			$GOVC import.ova -ds=$TEST_VMDATASTORE -name=$TEST_VM_TEMPLATE $1
		else
			$GOVC import.ovf -ds=$TEST_VMDATASTORE -name=$TEST_VM_TEMPLATE $1
		fi
		if [ $? -ne 0 ]; then
			delete_vms
			echo "Failed to deploy VM $vmname for $1"
			exit 1
		fi
	fi
	# Create the linked clones from the template and
	# add the VMs to the list of test VMs
	for vmname in ${VMS[@]}; do
		$GOVC $LINKED_CLONE $vmname
		if [ $? -ne 0 ]; then
			echo "Failed to create vm $vmname"
			delete_vms
			exit 1
		fi
		TEST_VMS+=($vmname)
	done
}

# Get the IPs for each VM deployed earlier
get_vm_ips()
{
	for vmname in ${TEST_VMS[@]}; do
		echo "Fetching IP for VM $vmname"
		vm_ip=`$GOVC vm.ip -a $vmname|cut -f1 -d,`
		echo "Got IP $vm_ip for VM $vmname"
		# add to the VM IPs list
		TEST_VM_IPS+=($vm_ip)
	done
}

# Copy the ssh public key to the new VMs
copy_ssh_keys()
{
	for vm_ip in ${TEST_VM_IPS[@]}; do
		echo "Copying keys to $vm_ip"
		cat $SSH_PUB_KEY | $SSH $ROOT@$vm_ip "cat >> $SSH_AUTH_KEYS"
	done
}

# Install docker if not present or upgrade docker to
# version specified in TES_DOCKER_VERSION
install_docker()
{
	vm_ip=$1
	$SSH $ROOT@$vm_ip $DOCKER_INFO #> /dev/null 2>&1
	if [ $? -ne 0 ]; then
		echo "Docker not found on $vm_ip, installing..."
		# try installing docker
		$SSH $ROOT@$vm_ip $DOCKER_INSTALL_COMMON
		if [ $? -ne 0 ]; then
			# try a tdnf install
			$SSH $ROOT@$vm_ip $DOCKER_INSTALL_TDNF
			if [ $? -ne 0 ]; then
				echo "Failed to install docker on $vm_ip, deleting vms, exiting."
				delete_vms
				exit 1
			fi
		fi
	else
		# check docker version matches requested version
		docker_version=`$DOCKER version --format '{{.Server.Version}}'`
		echo "Found docker version $docker_version installed, skipping install.."
	fi

}

# Verify docker is installed and running
verify_docker_available()
{
	for vm_ip in ${TEST_VM_IPS[@]}; do
		install_docker $vm_ip
		verify_docker_isrunning $vm_ip
	done
}

# Verify if docker is available on the VMs
verify_docker_isrunning()
{
	vm_ip=$1
	retry=10
	info=1
	while [ $info -ne 0 ] && [ $retry -gt 0 ]; do
		$SSH $ROOT@$vm_ip $DOCKER_INFO > /dev/null 2>&1
		info=$?
		if [ $info -ne 0 ]; then
			sleep 2
		fi
		retry=`expr $retry - 1`
	done
	if [ $info -ne 0 ]; then
		echo "Failed to detect docker instance on $vm_ip, deleting vms, exiting."
		delete_vms
		exit 1
	fi
}

# Create a docker swarm with one master and two worker nodes
create_docker_swarm()
{
	echo "Creating docker swarm on ${TEST_VMS[@]}"
	master_ip=${TEST_VM_IPS[0]}
	$SSH $ROOT@$master_ip $DOCKER node ls > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		$SSH $ROOT@master_ip $DOCKER swarm init --advertise-addr $master_ip > /dev/null 2>&1
	fi

	worker_token=`$DOCKER swarm join-token -q worker`

	# join the worker nodes to the swarm
	echo "Joining ${TEST_VM_IPS[1]} to the swarm."
	$SSH $ROOT@${TEST_VM_IPS[1]} $DOCKER swarm join --token $worker_token $master_ip:$DOCKERD_PORT

	echo "Joining ${TEST_VM_IPS[2]} to the swarm."
	$SSH $ROOT@${TEST_VM_IPS[2]} $DOCKER swarm join --token $worker_token $master_ip:$DOCKERD_PORT
}

# Create a manifest of the VMS created
create_manifest()
{
	cat > $MANIFEST <<TEST
	export VM1=${TEST_VM_IPS[0]}
	export VM2=${TEST_VM_IPS[1]}
	export VM3=${TEST_VM_IPS[2]}
	export VM1_NAME=${TEST_VMS[0]}
	export VM2_NAME=${TEST_VMS[1]}
	export VM3_NAME=${TEST_VMS[2]}
TEST
}

# Print a usage message
usage()
{
	echo "Missing argument."
	echo "testbed.sh <0|1>"
	echo "0 - Delete testbed"
	echo "1 - Create testbed"
}

if [ $# -eq 0 ]; then
	usage
	exit 1
fi

if [ $1 -eq 0 ]; then
	for n in ${VMS[@]}; do
		vmname=$TEST_USER-$n
		TEST_VMS+=($vmname)
	done
	delete_vms
	rm $MANIFEST
	echo "Removed testbed ${TEST_VMS[@]}"
	exit 0
fi

# If an OVA_PATH has been provided then use that to deploy the VMs
# else if an OVA_URL is provided then download the OVA and use that instead
if [ "X"$TEST_OVAURL"X" != "XX" ]; then
	fetch_ova $TEST_OVAURL
	deploy_vms ova_fpath
elif [ -f $TEST_OVAPATH ]; then
	deploy_vms $TEST_OVAPATH
else
	echo "OVA_PATH/OVA_URL not set, no OVA to deploy."
	exit 1
fi

get_vm_ips

copy_ssh_keys

verify_docker_available

create_docker_swarm

create_manifest

#End
