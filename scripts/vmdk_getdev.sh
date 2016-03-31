#!/bin/bash

scsiLabel="SCSI$2"
unit=$1
echo "got unit bus as $@" > /tmp/xx

cd /sys/bus/pci/devices
for i in `ls`
do
   if [ -e $i/label ]
   then 
      dev=`cat $i/label`

      if [ -n $dev ] && [ $dev = $scsiLabel ]
      then
         device="/dev/disk/by-path/pci-$i-scsi-0:0:$unit:0"
         echo $device
         exit 0
      fi
   fi
done

