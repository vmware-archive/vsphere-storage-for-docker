#!/bin/bash 
#
# simple wrapper for check build pre-requisites 
#

# Check prereq first - we need docker 

vibauthor --version > /dev/null
if [ $? -ne 0 ]
then 
   echo '***********************************************************'
   echo "Error: vibauthor needs to be installed to build repo."
   exit 1
fi
