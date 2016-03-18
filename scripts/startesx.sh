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


vibfile="$1/$(basename $2)"

# long running, so let's echo
echo "localcli software vib install --no-sig-check  -v $vibfile" 
localcli software vib install --no-sig-check  -v $vibfile

log=/tmp/plugin.log
pylog=/var/log/vmware/docker-vmdk-plugin.log 
echo === `date` Actual logs are in $pylog  === > $log
cat /dev/null > $pylog
nohup python -B /usr/lib/vmware/vmdkops/bin/vmci_srv.py >> $log 2>&1 &
