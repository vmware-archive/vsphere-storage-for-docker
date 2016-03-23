#!/bin/sh
#
# Stop vmci_srv.py process and remove the VIB

name=$1

pid=`ps -c | grep -v awk | awk '/vmci_srv.py/ { print $1 }'`
if [ "$pid" != "" ] 
then
   kill $pid
fi

if localcli software vib list | grep -q $name ;
then
   # long running, so let's always echo
   echo "localcli software vib remove --vibname $name" 
   localcli software vib remove --vibname $name
fi
