#!/bin/bash
OPERATION=$1
BUILD_NUMBER=build-$2

LOCK_DIR='/tmp/docker-vmdk-plugin.lock'

if [ "$OPERATION" == "lock" ]; then
  if mkdir "$LOCK_DIR";
  then
    echo "$BUILD_NUMBER has the lock"
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
