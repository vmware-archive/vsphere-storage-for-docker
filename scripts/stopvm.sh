#!/bin/bash
#
# Stop name process and clean up mount point used in test

name=$1

pid=`pidof $(basename $name)`
if [ "$pid" != "" ] 
then
   kill $pid
fi
$DEBUG rm -rvf /mnt/vmdk/*
