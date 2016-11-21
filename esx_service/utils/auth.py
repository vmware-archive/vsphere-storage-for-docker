# Copyright 2016 VMware, Inc. All Rights Reserved.
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

""" Module to provide APIs for authorization checking for VMDK ops.

"""
import logging
import auth_data
import sqlite3
import convert
import auth_data_const
import volume_kv as kv
import threadutils

# All supported vmdk commands
CMD_CREATE = 'create'
CMD_REMOVE = 'remove'
CMD_ATTACH = 'attach'
CMD_DETACH = 'detach'

SIZE = 'size'

# thread local storage in this module namespace
thread_local = threadutils.get_local_storage()

def get_auth_mgr():
    """ Get a connection to auth DB. """
    global thread_local
    if not hasattr(thread_local, '_auth_mgr'):
        thread_local._auth_mgr = auth_data.AuthorizationDataManager()
        thread_local._auth_mgr.connect()
    return thread_local._auth_mgr

def get_tenant(vm_uuid):
    """ Get tenant which owns this VM by querying the auth DB. """
    _auth_mgr = get_auth_mgr()
    try:
        cur = _auth_mgr.conn.execute(
                    "SELECT tenant_id FROM vms WHERE vm_id = ?",
                    (vm_uuid, )
        )
        result = cur.fetchone()
    except sqlite3.Error as e:
        logging.error("Error %s when querying from vms table for vm_id %s", e, vm_uuid)
        return str(e), None, None

    if result:
        logging.debug("get tenant vm_uuid=%s tenant_id=%s", vm_uuid, result[0])

    tenant_uuid = None
    tenant_name = None
    if result:
        tenant_uuid = result[0]
        try:
            cur = _auth_mgr.conn.execute(
                "SELECT name FROM tenants WHERE id = ?",
                (tenant_uuid, )
                )
            result = cur.fetchone()
        except sqlite3.Error as e:
            logging.error("Error %s when querying from tenants table for tenant_id %s",
                          e, tenant_uuid)
            return str(e), None, None
        if result:
            tenant_name = result[0]
            logging.debug("Found tenant_uuid %s, tenant_name %s", tenant_uuid, tenant_name)

    return None, tenant_uuid, tenant_name

def get_privileges(tenant_uuid, datastore):
    """ Return privileges for given (tenant_uuid, datastore) pair by
        querying the auth DB.
    """
    _auth_mgr = get_auth_mgr()
    privileges = []
    logging.debug("get_privileges tenant_uuid=%s datastore=%s", tenant_uuid, datastore)
    try:
        cur = _auth_mgr.conn.execute(
            "SELECT * FROM privileges WHERE tenant_id = ? and datastore = ?",
            (tenant_uuid, datastore)
            )
        privileges = cur.fetchone()
    except sqlite3.Error as e:
        logging.error("Error %s when querying privileges table for tenant_id %s and datastore %s",
                      e, tenant_uuid, datastore)
        return str(e), None
    return None, privileges

def has_privilege(privileges, type):
    """ Check the privileges has the specific type of privilege set. """
    if not privileges:
        return False
    logging.debug("%s=%d", type, privileges[type])
    return privileges[type]

def get_vol_size(opts):
    """ get volume size. """
    if not opts or SIZE not in opts:
        logging.warning("Volume size not specified")
        return kv.DEFAULT_DISK_SIZE
    return opts[SIZE].upper()


def check_max_volume_size(opts, privileges):
    """ Check whether the size of the volume to be created exceeds
        the max volume size specified in the privileges.

    """
    if privileges:
        vol_size_in_MB = convert.convert_to_MB(get_vol_size(opts))
        max_vol_size_in_MB = privileges[auth_data_const.COL_MAX_VOLUME_SIZE]
        logging.debug("vol_size_in_MB=%d max_vol_size_in_MB=%d",
                      vol_size_in_MB, max_vol_size_in_MB)
        # if max_vol_size_in_MB which read from DB is 0, which means
        # no max_vol_size limit, function should return True
        if max_vol_size_in_MB == 0:
            return True
        return vol_size_in_MB <= max_vol_size_in_MB
    else:
        # no privileges
        return True

def get_total_storage_used(tenant_uuid, datastore):
    """ Return total storage used by (tenant_uuid, datastore)
        by querying auth DB.

    """
    _auth_mgr = get_auth_mgr()
    total_storage_used = 0
    try:
        cur = _auth_mgr.conn.execute(
            "SELECT SUM(volume_size) FROM volumes WHERE tenant_id = ? and datastore = ?",
            (tenant_uuid, datastore)
            )
    except sqlite3.Error as e:
        logging.error("Error %s when querying storage table for tenant_id %s and datastore %s",
                      e, tenant_uuid, datastore)
        return str(e), total_storage_used
    result = cur.fetchone()
    if result:
        if result[0]:
            total_storage_used = result[0]
            logging.debug("total storage used for (tenant %s datastore %s) is %s MB", tenant_uuid,
                          datastore, total_storage_used)

    return None, total_storage_used

def check_usage_quota(opts, tenant_uuid, datastore, privileges):
    """ Check if the volume can be created without violating the quota. """
    if privileges:
        vol_size_in_MB = convert.convert_to_MB(get_vol_size(opts))
        error_info, total_storage_used = get_total_storage_used(tenant_uuid, datastore)
        if error_info:
            # cannot get the total_storage_used, to be safe, return False
            return False
        usage_quota = privileges[auth_data_const.COL_USAGE_QUOTA]
        # if usage_quota which read from DB is 0, which means
        # no usage_quota, function should return True
        if usage_quota == 0:
            return True
        return vol_size_in_MB + total_storage_used <= usage_quota
    else:
        # no privileges
        return True

def check_privileges_for_command(cmd, opts, tenant_uuid, datastore, privileges):
    """
        Check whether the (tenant_uuid, datastore) has the privileges to run
        the given command.

    """
    result = None
    cmd_need_mount_privilege = [CMD_ATTACH, CMD_DETACH]
    if cmd in cmd_need_mount_privilege:
        if not has_privilege(privileges, auth_data_const.COL_MOUNT_VOLUME):
            result = "No mount privilege"

    if cmd == CMD_CREATE:
        if not has_privilege(privileges, auth_data_const.COL_CREATE_VOLUME):
            result = "No create privilege"
        if not check_max_volume_size(opts, privileges):
            result = "volume size exceeds the max volume size limit"
        if not check_usage_quota(opts, tenant_uuid, datastore, privileges):
            result = "The total volume size exceeds the usage quota"

    if cmd == CMD_REMOVE:
        if not has_privilege(privileges, auth_data_const.COL_DELETE_VOLUME):
            result = "No delete privilege"

    return result

def tables_exist():
    """ Check tables needed for authorization exist or not. """
    _auth_mgr = get_auth_mgr()

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'tenants';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table tenants exists or not", e)
        return str(e), False

    if not result:
        error_info = "table tenants does not exist"
        logging.error(error_info)
        return error_info, False

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'vms';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table vms exists or not", e)
        return str(e), False

    if not result:
        error_info = "table vms does not exist"
        logging.error(error_info)
        return error_info, False

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'privileges';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table privileges exists or not", e)
        return str(e), False

    if not result:
        error_info = "table privileges does not exist"
        logging.error(error_info)
        return error_info, False

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'volumes';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table volumes exists or not", e)
        return str(e), False

    if not result:
        error_info = "table volumes does not exist"
        logging.error(error_info)
        return error_info, False

    return None, True

def authorize(vm_uuid, datastore, cmd, opts):
    """ Check whether the command can be run on this VM.

        Return value: result, tenant_uuid, tenant_name

        - result: return None if the command can be run on this VM, otherwise, return
        corresponding error message
        - tenant_uuid: If the VM belongs to a tenant, return tenant_uuid, otherwise, return
        None
        - tenant_name: If the VM belongs to a tenant, return tenant_name, otherwise, return
        None

    """
    logging.debug("Authorize: vm_uuid=%s", vm_uuid)
    logging.debug("Authorize: datastore=%s", datastore)
    logging.debug("Authorize: cmd=%s", cmd)
    logging.debug("Authorize: opt=%s", opts)

    try:
        get_auth_mgr()
    except auth_data.DbConnectionError as e:
        error_info = "Failed to connect auth DB({0})".format(e)
        return error_info, None, None

    # If table "tenants", "vms", "privileges" or "volumes" does not exist
    # don't need auth check
    if not tables_exist():
        logging.error("Required tables in auth db do not exist")
        error_info = "Required tables in aut db do not exist"
        return error_info, None, None

    error_info, tenant_uuid, tenant_name = get_tenant(vm_uuid)
    if error_info:
       return error_info, None, None

    if not tenant_uuid:
        # This VM does not associate any tenant, don't need auth check
        logging.debug("VM %s does not belong to any tenant", vm_uuid)
        return None, None, None
    else:
        error_info, privileges = get_privileges(tenant_uuid, datastore)
        if error_info:
            return error_info, None, None
        result = check_privileges_for_command(cmd, opts, tenant_uuid, datastore, privileges)

        if not result:
            logging.info("cmd %s with opts %s on tenant_uuid %s datastore %s is allowed to execute",
                         cmd, opts, tenant_uuid, datastore)

        return result, tenant_uuid, tenant_name

def add_volume_to_volumes_table(tenant_uuid, datastore, vol_name, vol_size_in_MB):
    """ Insert volume to volumes table. """
    _auth_mgr = get_auth_mgr()

    logging.debug("add to volumes table(%s %s %s %s)", tenant_uuid, datastore,
                  vol_name, vol_size_in_MB)
    try:
        _auth_mgr.conn.execute(
            "INSERT INTO volumes(tenant_id, datastore, volume_name, volume_size) VALUES (?, ?, ?, ?)",
            (tenant_uuid, datastore, vol_name, vol_size_in_MB)
            )
        _auth_mgr.conn.commit()
    except sqlite3.Error as e:
        logging.error("Error %s when insert into volumes table for tenant_id %s and datastore %s",
                      e, tenant_uuid, datastore)
        return str(e)

    return None

def get_row_from_tenants_table(conn, tenant_uuid):
    """ Get a row from tenants table for a given tenant """

    try:
        cur = conn.execute(
            "SELECT * FROM tenants WHERE id=?",
            (tenant_uuid,)
            )
    except sqlite3.Error as e:
        logging.error("Error: %s when querying tenants table for tenant %s",
                      e, tenant_uuid)
        return str(e), None

    result = cur.fetchone()
    return None, result

def get_row_from_vms_table(conn, tenant_uuid):
    """ Get rows from vms table for a given tenant """

    try:
        cur = conn.execute(
            "SELECT * FROM vms WHERE tenant_id=?",
            (tenant_uuid,)
            )
    except sqlite3.Error as e:
        logging.error("Error: %s when querying vms table for tenant %s", e, tenant_uuid)
        return str(e), None

    result = cur.fetchall()
    return None, result

def get_row_from_privileges_table(conn, tenant_uuid):
    """ Get rows from privileges table for a given tenant """

    try:
        cur = conn.execute(
            "SELECT * FROM privileges WHERE tenant_id=?",
            (tenant_uuid,)
            )
    except sqlite3.Error as e:
        logging.error("Error: %s when querying privileges table for tenant %s", e, tenant_uuid)
        return str(e), None

    result = cur.fetchall()
    return None, result

