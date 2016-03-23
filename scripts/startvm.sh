#!/bin/bash
plugin_name=$1

stdlog=/tmp/plugin.log
log=/var/log/docker-vmdk-plugin.log
echo ==== `date` ===== > $stdlog
echo ==== `date` ===== > $log
echo plugin_name=$plugin_name
eval "$plugin_name  < /dev/null >> $log 2>&1 &"

