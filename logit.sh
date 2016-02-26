#!/bin/bash
# Log and run commands
echo "====================== run" $1 "======================" >> run.log
date -u >> run.log
$1 | tee -a run.log
date -u >> run.log
echo "====================== end" $1 "======================" >> run.log
