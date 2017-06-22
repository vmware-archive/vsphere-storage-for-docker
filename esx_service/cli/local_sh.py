#!/usr/bin/env python
#
#  Copyright 2017 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Support for adding/removing information about config db link from /etc/rc.local.d/local.sh
Any config stuff we need and configure in /etc/... will be removed on ESX reboot.
Anything we need to persist between the reboots, needs to be configured here.
"""

import sys
import os.path
import fileinput

# We expect this record at the end of the local.sh.
# We need to insert the content before it.
END_OF_SCRIPT = "exit 0"

# This is what we use to identify the content for DB links..
CONFIG_DB_TAG = "# -- vSphere Docker Volume Service configuration --"

# This is the content template for db links. '{}' will be replaced by datastore name
CONFIG_DB_INFO = CONFIG_DB_TAG + \
"""
#
# Please do not edit this section manually. It is managed by vmdkops_admin.py config command.
# Note: the code relies on local.sh having "exit 0" at the end.
#

datastore={}

slink=/etc/vmware/vmdkops/auth-db
shared_db=/vmfs/volumes/$datastore/dockvols/vmdkops_config.db

if [ -d $(dirname $slink) ] && [ ! -e $slink  ]
then
    ln -s $shared_db $slink
fi

""" + CONFIG_DB_TAG + "\n"

# full path of local.sh script, used to make things persistent between ESXi reboots
LOCAL_SH_PATH = '/etc/rc.local.d/local.sh'


# open file and scan it.
#
# if we reach "exit 0", then add the text section, add the rest of the file and be done
# if we find the tag, then skip to the end of the section
#
def update_content(content, tag, add=True, file=LOCAL_SH_PATH):
    """
    A generic function to update (add or removes) <content> limited with <tag>s, in a <file>
    """
    if not os.path.exists(file):
        return # silently do nothing if the file does not exit
    skip_to_tag = no_more_checks = False
    for line in iter(fileinput.input([file], inplace=True, backup=".bck")):
        if no_more_checks:
            sys.stdout.write(line)
            continue
        if skip_to_tag:
            if line.startswith(tag):
                # found the second tag, complete operation.
                no_more_checks = True
            continue
        if line.startswith(tag):
            # First tag - add the content (if needed) and skip till the next tag
            if add:
                sys.stdout.write(content)
            skip_to_tag = True
            continue
        if line.startswith(END_OF_SCRIPT):
            if add:
                sys.stdout.write(content)
            no_more_checks = True
        sys.stdout.write(line)
    if not no_more_checks:
        # for some reason we did not find 'exit 0' nor tag, so let's dump the
        # requested content just in case.
        if add:
            sys.stdout.write(content)


def update_symlink_info(ds_name="n/a", add=True):
    """Convenience wrapper for updating symlink info only"""
    update_content(CONFIG_DB_INFO.format(ds_name), CONFIG_DB_TAG, add=add)

