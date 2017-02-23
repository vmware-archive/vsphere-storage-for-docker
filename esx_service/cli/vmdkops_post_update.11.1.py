#!/usr/bin/env python
# Copyright 2017 VMware, Inc. All Rights Reserved.
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

# -------------------------------------------------------------------------------------------
# Updates ESX with vSphere Docker Volume Service 0.11 (and earlier)
#  to 0.11.1 and further
# -------------------------------------------------------------------------------------------

import os
import os.path
import sqlite3
import sys
import shutil

import vmdk_ops

# vmdkops python utils are in PY_LOC, so add to path.
sys.path.insert(0, vmdk_ops.PY_LOC)
import vmdk_utils
import auth
import auth_data

# Hard coded (in auth package) UUD for default tenant.
STATIC_UUID = auth.DEFAULT_TENANT_UUID

STATIC_NAME = auth.DEFAULT_TENANT

# CLI return codes
OK = 0
ERROR = 1

# do we need to stop and restart the vmdkops service
STOP_SERVICE = True  # 'False' is for debug only - makes it faster

def patch_a_store(ds_path, old_uuid):
    """Renames and moves stuff as needed in a single DS/dockvols"""
    print("Working on Datastore '{0}'".format(ds_path))

    # move stuff from old_uuid to new_uuid ()
    old_dir = os.path.join(ds_path, old_uuid)
    new_dir = os.path.join(ds_path, STATIC_UUID)
    symlink_name = os.path.join(ds_path, STATIC_NAME)
    if not os.path.isdir(old_dir):
        print("  Skipping {0} - not found".format(old_dir))
        return

    if os.path.exists(new_dir):
        # target exists , move files and remove oldir
        print("  Moving from {0}, to {1}".format(old_dir, new_dir))
        for f in os.listdir(old_dir):
            src = os.path.join(old_dir, f)
            dst = os.path.join(new_dir, f)
            if os.path.isfile(dst):
                print("    File {0} already exists, skipping the move".format(dst))
                continue
            shutil.move(src, dst)
        if not os.listdir(old_dir):
            print("  Deleting empty {0}".format(old_dir))
            os.rmdir(old_dir)
        else:
            print("  *** Warning: {0} is not empty after migration. Please check the content.")
    else:
        print("  Renaming {0} to {1}".format(old_dir, new_dir))
        os.rename(old_dir, new_dir)

    print("  Adjusting {0} symlink to pont to {1}".format(symlink_name, STATIC_UUID))
    try:
        os.remove(symlink_name)
    except:
        pass
    os.symlink(STATIC_UUID, symlink_name)


def main():
    """
    This code updates ESX with vSphere Docker Volume Service 0.11 (and earlier)
    to 0.11.1 and further, by moving _DEFAULT tenant ID to well known and static UUID,
    and then correcting directories layout and auth_db tables to comply with new UUID.

    Specifically, it does the following:
    - Checks if AUTH_DB exists.
      If it does not, exit with a message - it means nothing to patch on this ESX
    - Gets uuid (aka "old_uuid') for _DEFAULT tenant from DB.
      If it already STATIC_UUID , exit with a message - nothing to patch
    - Stops the service
    - backs up the DB
    - scans through all <datastore>/volumes/dockvols and
        - mkdir STATIC_UUID, if it does not exist
        - move all from old_uuid to STATIC_UUID
        - symlinks "_DEFAULT" to STATIC_UUID
    In single DB transcation
        - replaces old_uuid with STATIC UUID in tenant_id field for all tables:
          (privileges, vms,  tenants, volumes)
    starts the service , and if all good removes backup DB

    NOTE: this does not delete any data, so the Docker volumes will stay around
          no matter if the code succeeds or fails
    """

    dbfile = auth_data.AUTH_DB_PATH

    # STEP: check DB presense and fetch old_uuid
    if not os.path.isfile(dbfile):
        print("Config DB", dbfile, "is not found, nothing to update - exiting.")
        sys.exit(0)

    cursor = sqlite3.connect(dbfile).cursor()
    cursor.execute("select * from tenants where name='{0}'".format(STATIC_NAME))
    try:
        tenant_id, tenant_name, tenant_desr, tenant_def_ds = cursor.fetchone()
    except TypeError:
        print("Can't find '{0}' tenant, exiting".format(STATIC_NAME))
        sys.exit(ERROR)

    print("Found default tenant: {0} {1} {2} {3}".format(tenant_id,
                                                         tenant_name, tenant_desr, tenant_def_ds))

    old_uuid = tenant_id
    if old_uuid == STATIC_UUID:
        print("*** DB seems to have been already migrated,  exiting ***")
        sys.exit(OK)


    # STEP: Stop the service and back up the DB
    backup = dbfile + ".bck"
    if os.path.isfile(backup):
        print("Backup file '{0}' already exists - skipping DB backup".format(backup))
    else:
        print("Backing up Config DB to '{0}'".format(backup))
        shutil.copy(dbfile, backup)

    if STOP_SERVICE:
        print("Stopping vmdk-opsd service")
        os.system("/etc/init.d/vmdk-opsd stop")

    # STEP : patch a datastore - convert dir names to new UUID if needed and move files
    print("Starting conversion of _DEFAULT tenant directory names. old_uid is {0}".format(old_uuid))
    stores = vmdk_utils.get_datastores()
    if not stores:
        print("Docker volume storage is not initialized - skipping directories patching")
    else:
        for datastore in stores:
            ds_path = datastore[2]
            patch_a_store(ds_path, old_uuid)

    # STEP: patch database
    print("Working on DB patch...")

    # sql for update the DB
    # note that:
    #       {0} is old_uuid (default tenant uuid pre-upgrade)
    #       {1} is new_uuid (default tenant uuid post-upgrade)
    #       {2} is tmp name - we need it to comply with DB constraints
    #       {3} is default tenant description (from DB)
    #       {4} is default DB for default tenant (from DB)
    #       {5} is the name ("_DEFAULT") for default tenant
    # TBD - use named params in formatting
    sql_query_template = \
    """
    -- insert temp record to make foreign key happy
    INSERT INTO tenants VALUES ( '{1}', '{2}', '{3}', '{4}' ) ;

    -- update the tables
    UPDATE vms SET tenant_id = '{1}' WHERE tenant_id = '{0}';
    UPDATE volumes SET tenant_id = '{1}' WHERE tenant_id = '{0}';
    UPDATE privileges SET tenant_id = '{1}' WHERE tenant_id = '{0}';

    -- recover _DEFAULT tenant record
    DELETE FROM tenants WHERE id = '{0}';
    UPDATE tenants SET name = '{5}' WHERE name = '{2}';
    UPDATE versions SET major_ver=1, minor_ver=1;
    """

    tmp_tenant_name = "__tmp_name_upgrade_0_11"
    sql_query = sql_query_template.format(old_uuid, STATIC_UUID, tmp_tenant_name,
                                          tenant_desr, tenant_def_ds,
                                          STATIC_NAME)
    cursor.executescript(sql_query)

    # STEP: restart the service
    if STOP_SERVICE:
        print("Starting vmdk-opsd  service")
        os.system("/etc/init.d/vmdk-opsd start")

    # TBD: remove backup ?
    print ("*** ALL DONE ***")


if __name__ == "__main__":
    main()
