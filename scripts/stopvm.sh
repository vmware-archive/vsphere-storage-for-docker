#!/bin/bash
pid=`pidof docker-vmdk-plugin`
if [ "$pid" != "" ] 
then
   $E kill $pid
fi
rm -rvf /mnt/vmdk/*
