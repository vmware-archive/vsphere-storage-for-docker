#!/bin/bash
#
# Stop name process and clean up mount point used in test

name=$1

pid=`pidof $(basename $name)`
if [ "$pid" != "" ] 
then
   kill $pid
fi
for d in /mnt/vmdk/*
do
   $DEBUG umount $d 2>/dev/null
   $DEBUG rm -rvf $d
done
