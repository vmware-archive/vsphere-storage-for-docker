#!/bin/bash
log=/tmp/plugin.log
echo ==== `date` ===== >> $log
nohup python /usr/lib/vmware/vmdkops/bin/vmci_srv.py >> $log 2>&1 &
