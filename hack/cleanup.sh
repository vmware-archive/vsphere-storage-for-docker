#!/bin/sh

cleanup() {
  make clean-vm VM=$VM1 TEST_VOL_NAME=vol-build$BUILD_NUMBER
  make clean-vm VM=$VM2 TEST_VOL_NAME=vol-build$BUILD_NUMBER
  make clean-esx ESX=$ESX
}

stop_build() {
  cleanup
  $SSH $VM1 /tmp/lock.sh unlock $BUILD_NUMBER
}
