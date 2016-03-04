#!/bin/sh

cleanup() {
  make clean-vm VM=$VM1
  make clean-vm VM=$VM2
  make clean-esx ESX=$ESX
}

SSH="ssh -o StrictHostKeyChecking=no"
VM=$1
BUILD_NUMBER=$2

stop_build() {
  cleanup
  $SSH $VM1 /tmp/lock.sh unlock $BUILD_NUMBER
}
