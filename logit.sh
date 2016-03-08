#!/bin/bash
set -o pipefail
# Log and run commands
echo "====================== run" $1 "======================" >> run.log
date -u >> run.log
$1 | tee -a run.log
exit ${PIPESTATUS[0]}
