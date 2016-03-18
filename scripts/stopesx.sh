#!/bin/sh
# VIB note: we remove by internal name, not file name - see description.xml in VIB
vibname=vmware-esx-vmdkops-service

pid=`ps -c | grep -v awk | awk '/vmci_srv.py/ { print $1 }'`
if [ "$pid" != "" ] 
then
   kill $pid
fi

if localcli software vib list | grep -q $vibname ;
then
   # long running, so let's echo
   echo "localcli software vib remove --vibname $vibname" 
   localcli software vib remove --vibname $vibname
fi
