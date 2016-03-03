#!/bin/bash

# Log and run commands
echo "====================== run" $1 "======================" >> run.log
date -u >> run.log
$1 >> run.log
