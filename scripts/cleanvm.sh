#!/bin/bash

# Kill process if it runs

name=$1

pid=`pidof $(basename $name)`
if [ "$pid" != "" ] 
then
   kill $pid
fi

