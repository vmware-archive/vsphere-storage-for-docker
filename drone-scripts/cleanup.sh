#!/bin/sh

stop_build() {
  make clean-esx clean-vm TEST_VOL_NAME=vol-build$BUILD_NUMBER
  $SSH $VM1 /tmp/lock.sh unlock $BUILD_NUMBER
}
