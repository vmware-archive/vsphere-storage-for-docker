#!/bin/sh
# temporary script to run vmci_srv.py detached.
#
# installs the VIB and starts the service. 
# passed location on test , and full binary name on build machine
#
# Logs info:
# The real log is in "/var/log/vmware/docker-vmdk-plugin.log" (see vmci_srv.py )
# redirtecting stdio/err to /tmp/plugin.log in case we missed something in logs
# this will be gone when VIB work is complete
# Note that we are resetting actual log here - it is useful in Drone runs. 

vibfile=$1

pylog=/var/log/vmware/docker-vmdk-plugin.log 
cat /dev/null > $pylog
# long running, so let's always echo
echo "localcli software vib install --no-sig-check  -v $vibfile" 
localcli software vib install --no-sig-check  -v $vibfile
localcli --plugin-dir=/usr/lib/vmware/esxcli/int sched group list| grep vmdkops | grep python> /dev/null
/etc/init.d/vmdk-opsd status| grep pid
