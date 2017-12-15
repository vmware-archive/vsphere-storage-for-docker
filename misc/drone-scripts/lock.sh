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


OPERATION=$1
BUILD_NUMBER=build-$2

LOCK_DIR='/tmp/vsphere-storage-for-docker.lock'

if [ "$OPERATION" == "lock" ]; then
  if mkdir "$LOCK_DIR";
  then
    echo "$BUILD_NUMBER" > $LOCK_DIR/build
    exit 0
  else
    echo "Cannot lock $LOCK_DIR held by " `cat $LOCK_DIR/build`
    exit 1
  fi
fi

LOCK_BUILD_NUMBER=`cat $LOCK_DIR/build`

if [ "$OPERATION" == "unlock" ]; then

  if [ "$LOCK_BUILD_NUMBER" != "$BUILD_NUMBER" ]; then
    echo "lock held by $LOCK_BUILD_NUMBER != $BUILD_NUMBER"
    exit 1
  fi

  rm $LOCK_DIR/build
  rmdir $LOCK_DIR
  echo "Lock for $BUILD_NUMBER released"
  exit 0
fi

exit 1
