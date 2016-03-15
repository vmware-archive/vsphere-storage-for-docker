#!/bin/sh
stdlog=/tmp/plugin.log
log=/var/log/docker-vmdk-plugin.log
echo ==== `date` ===== > $stdlog
echo ==== `date` ===== > $log
/usr/local/bin/docker-vmdk-plugin < /dev/null >> $log 2>&1 &
