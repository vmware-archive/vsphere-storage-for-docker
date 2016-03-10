#!/bin/sh
log=/tmp/plugin.log
echo ==== `date` ===== >> $log
/usr/local/bin/docker-vmdk-plugin < /dev/null >> $log 2>&1 &
