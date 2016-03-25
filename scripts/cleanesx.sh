#!/bin/sh
#

name=$1

if localcli software vib list | grep -q $name ;
then
   # long running, so let's always echo
   echo "localcli software vib remove --vibname $name" 
   localcli software vib remove --vibname $name
   localcli --plugin-dir=/usr/lib/vmware/esxcli/int sched group list --group-path=host/vim/vimuser/cnastorage/ > /dev/null 2>&1
   if [ $? -eq 0 ];
   then
     echo "Failed to clean up resource groups!"
     exit 1
   fi
fi

