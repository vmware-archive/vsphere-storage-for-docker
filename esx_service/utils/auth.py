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
import log_config
import error_code
import vmdk_utils
from error_code import ErrorCode
from error_code import error_code_to_message

# All supported vmdk commands that need authorization checking
CMD_CREATE = 'create'
CMD_REMOVE = 'remove'
CMD_ATTACH = 'attach'
CMD_DETACH = 'detach'
CMD_GET    = 'get'

SIZE = 'size'

# thread local storage in this module namespace
thread_local = threadutils.get_local_storage()

def get_auth_mgr():
    """ Get a connection to auth DB. """
    global thread_local
    if not hasattr(thread_local, '_auth_mgr'):
        try:
            thread_local._auth_mgr = auth_data.AuthorizationDataManager()
            thread_local._auth_mgr.connect()
        except (auth_data.DbConnectionError, auth_data.DbAccessError, auth_data.DbUpgradeError) as err:
            return str(err), None
    return None, thread_local._auth_mgr

def get_default_tenant():
    """
        Get DEFAULT tenant by querying the auth DB or from hardcoded defaults.
        VM which does not belong to any tenant explicitly will
        be assigned to DEFAULT tenant if DEFAULT tenant exists
        Returns: (err, uuid, name)
        -- error_msg: return None on success or error info on failure
        -- tenant_uuid: return DEFAULT tenant uuid on success,
           return None on failure or DEFAULT tenant does not exist
        -- tenant_name: return DEFAULT tenant name on success,
           return None on failure or DEFAULT tenant does not exist
    """
    tenant_uuid = None
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg, None, None

    if _auth_mgr.allow_all_access():
        return None, auth_data_const.DEFAULT_TENANT_UUID, auth_data_const.DEFAULT_TENANT

    try:
        cur = _auth_mgr.conn.execute(
            "SELECT id FROM tenants WHERE name = ?",
            (auth_data_const.DEFAULT_TENANT, )
            )
        result = cur.fetchone()
    except sqlite3.Error as e:
        error_msg = "Error {0} when querying from tenants table for tenant {1}".format(e, auth_data_const.DEFAULT_TENANT)
        logging.error(error_msg)
        return str(e), None, None
    if result:
        # found DEFAULT tenant
        tenant_uuid = result[0]
        logging.debug("Found DEFAULT tenant, tenant_uuid %s, tenant_name %s", tenant_uuid, auth_data_const.DEFAULT_TENANT)
        return None, tenant_uuid, auth_data_const.DEFAULT_TENANT
    else:
        # cannot find DEFAULT tenant
        err_msg = error_code_to_message[ErrorCode.TENANT_NOT_EXIST].format(auth_data_const.DEFAULT_TENANT)
        logging.debug(err_msg)
        return None, None, None


def get_all_ds_privileges(tenant_uuid):
    """ Get _ALL_DS privilege from AuthDB. This privilege is 'catch-all' fallbacks for datastores not explicitly bound
        to the given tenant_uuid'
        Return value:
        -- error_msg: return None on success or error info on failure
        -- privileges: return a list of privileges on datastore "_ALL_DS"
           return None on failure or the privilege does not exist
    """
    logging.debug("get_all_ds_privileges")
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg, None

    if _auth_mgr.allow_all_access():
        return None, _auth_mgr.get_all_ds_privileges_dict()

    privileges = []

    try:
        cur = _auth_mgr.conn.execute(
            "SELECT * FROM privileges WHERE tenant_id = ? and datastore_url = ?",
            (tenant_uuid, auth_data_const.ALL_DS_URL,)
            )
        privileges = cur.fetchone()
    except sqlite3.Error as e:
        error_msg = "Error {} when querying privileges table for all ds privilege for tenant_uuid {}".format(e, tenant_uuid)
        logging.error(error_msg)
        return str(e), None

    return None, privileges

def get_tenant(vm_uuid):
    """
        Get tenant which owns this VM by querying the auth DB.
        Return: error_msg, tenant_uuid, tenant_name
        -- error_msg: return None on success or error info on failure
        -- tenant_uuid: return tenant uuid which the VM with given vm_uuid is associated to,
           return None if the VM is not associated to any tenant
        -- tenant_name: return tenant name which the VM with given vm_uuid is associated to,
           return None if the VM is not associated to any tenant
    """
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg, None, None
    logging.debug("auth.get_tenant: allow_all: %s, uuid: %s", _auth_mgr.allow_all_access(), vm_uuid)
    if _auth_mgr.allow_all_access():
        logging.debug("returning default info")
        return None, auth_data_const.DEFAULT_TENANT_UUID, auth_data_const.DEFAULT_TENANT

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
    else:
        error_msg, tenant_uuid, tenant_name = get_default_tenant()
        if error_msg:
            return error_msg, None, None
        if not tenant_uuid:
             vm_name = vmdk_utils.get_vm_name_by_uuid(vm_uuid)
             if vm_name:
                 err_msg = error_code_to_message[ErrorCode.VM_NOT_BELONG_TO_TENANT].format(vm_name)
             else:
                 err_msg = error_code_to_message[ErrorCode.VM_NOT_BELONG_TO_TENANT].format(vm_uuid)
             logging.debug(err_msg)
             return err_msg, None, None
        return None, tenant_uuid, tenant_name


def get_privileges(tenant_uuid, datastore_url):
    """ Return privileges for given (tenant_uuid, datastore_url) pair by
        querying the auth DB.
        Return value:
        -- error_msg: return None on success or error info on failure
        -- privilegs: return a list of privileges for given (tenant_uuid, datastore_url)
           return None on failure
    """
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg, None

    privileges = []
    try:
        cur = _auth_mgr.conn.execute(
            "SELECT * FROM privileges WHERE tenant_id = ? and datastore_url = ?",
            (tenant_uuid, datastore_url)
            )
        privileges = cur.fetchone()
    except sqlite3.Error as e:
        logging.error("Error %s when querying privileges table for tenant_id %s and datastore_url %s",
                      e, tenant_uuid, datastore_url)
        return str(e), None
    if privileges:
        return None, privileges
    else:
        # check if can get privileges on ALL_DS for this tenant
        error_msg, privileges = get_all_ds_privileges(tenant_uuid)
        return error_msg, privileges

def has_privilege(privileges, type=None):
    """ Check whether the param "privileges" has the specific type of privilege set.
        There are two types of privilege:
        - "mount_only" which means only mount and unmount are allowed
        - "full_access" which means create, remove, mount, unmount are
        allowed
        a privilege with "allow_create=False" has "mount_only" privilege
        a privilege with "allow_create=True" has "full_access" privilege
        if param "type" is not specified, which means only need to check
        whether the param "privileges" has the "mount_only" privilege or not;
        if param "allow_create" is specified, which means need to check
        whether the input "privileges" has "full_access" privilege or not
    """
    # privileges is None, return False
    if not privileges:
        return False
    if type:
        logging.debug("has_privilege: %s=%d", type, privileges[type])
        return privileges[type]

    return True

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

def get_total_storage_used(tenant_uuid, datastore_url):
    """ Return total storage used by (tenant_uuid, datastore_url)
        by querying auth DB.

        Return value:
        -- error_msg: return None on success or error info on failure
        -- total_storage_used: return total storage used for given (tenant_uuid, datastore_url)
                               return None on failure

    """
    total_storage_used = 0
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg, total_storage_used

    try:
        cur = _auth_mgr.conn.execute(
            "SELECT SUM(volume_size) FROM volumes WHERE tenant_id = ? and datastore_url = ?",
            (tenant_uuid, datastore_url)
            )
    except sqlite3.Error as e:
        logging.error("Error %s when querying storage table for tenant_id %s and datastore_url %s",
                      e, tenant_uuid, datastore_url)
        return str(e), total_storage_used
    result = cur.fetchone()
    if result:
        if result[0]:
            total_storage_used = result[0]
            logging.debug("total storage used for (tenant %s datastore_url %s) is %s MB", tenant_uuid,
                          datastore_url, total_storage_used)

    return None, total_storage_used

def check_usage_quota(opts, tenant_uuid, datastore_url, privileges):
    """ Check if the volume can be created without violating the quota. """
    if privileges:
        vol_size_in_MB = convert.convert_to_MB(get_vol_size(opts))
        error_msg, total_storage_used = get_total_storage_used(tenant_uuid, datastore_url)
        if error_msg:
            # cannot get the total_storage_used, to be safe, return False
            return False
        usage_quota = privileges[auth_data_const.COL_USAGE_QUOTA]
        logging.debug("total_storage_used=%d, usage_quota=%d", total_storage_used, usage_quota)
        # if usage_quota which read from DB is 0, which means
        # no usage_quota, function should return True
        if usage_quota == 0:
            return True
        return vol_size_in_MB + total_storage_used <= usage_quota
    else:
        # no privileges
        return True

def check_privileges_for_command(cmd, opts, tenant_uuid, datastore_url, privileges):
    """
        Check whether the (tenant_uuid, datastore) has the privileges to run
        the given command.

    """
    result = None
    if not privileges:
        result = error_code_to_message[ErrorCode.PRIVILEGE_NO_PRIVILEGE]
        return result

    cmd_need_mount_privilege = [CMD_ATTACH, CMD_DETACH]
    if cmd in cmd_need_mount_privilege:
        if not has_privilege(privileges):
            result = error_code_to_message[ErrorCode.PRIVILEGE_NO_MOUNT_PRIVILEGE]

    if cmd == CMD_CREATE:
        if not has_privilege(privileges, auth_data_const.COL_ALLOW_CREATE):
            result = error_code_to_message[ErrorCode.PRIVILEGE_NO_CREATE_PRIVILEGE]
        if not check_max_volume_size(opts, privileges):
            result = error_code_to_message[ErrorCode.PRIVILEGE_MAX_VOL_EXCEED]
        if not check_usage_quota(opts, tenant_uuid, datastore_url, privileges):
            result = error_code_to_message[ErrorCode.PRIVILEGE_USAGE_QUOTA_EXCEED]

    if cmd == CMD_REMOVE:
        if not has_privilege(privileges, auth_data_const.COL_ALLOW_CREATE):
            result = error_code_to_message[ErrorCode.PRIVILEGE_NO_DELETE_PRIVILEGE]

    return result

def err_msg_no_table(table_name):
    error_msg = "table " + table_name + " does not exist"
    logging.error(error_msg)
    return error_msg

def tables_exist():
    """ Check tables needed for authorization exist or not. """
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg, False

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'tenants';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table tenants exists or not", e)
        return str(e), False

    if not result:
        error_msg = err_msg_no_table('tenant')
        return error_msg, False

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'vms';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table vms exists or not", e)
        return str(e), False

    if not result:
        error_msg = err_msg_no_table('vms')
        return error_msg, False

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'privileges';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table privileges exists or not", e)
        return str(e), False

    if not result:
        error_msg = err_msg_no_table('privileges')
        return error_msg, False

    try:
        cur = _auth_mgr.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'volumes';")
        result = cur.fetchall()
    except sqlite3.Error as e:
        logging.error("Error %s when checking whether table volumes exists or not", e)
        return str(e), False

    if not result:
        error_msg = err_msg_no_table('volumes')
        return error_msg, False

    return None, True

def authorize(vm_uuid, datastore_url, cmd, opts):
    """ Check whether the command can be run on this VM.

        Return value: result, tenant_uuid, tenant_name

        - result: return None if the command can be run on this VM, otherwise, return
        corresponding error message
        - tenant_uuid: If the VM belongs to a tenant, return tenant_uuid, otherwise, return
        None
        - tenant_name: If the VM belongs to a tenant, return tenant_name, otherwise, return
        None
    """
    logging.debug("Authorize: cmd=%s opts=`%s' vm_uuid=%s, datastore_url=%s", cmd, opts, vm_uuid, datastore_url)
    # The possible value of "datastore_url" can be url of real datastore or "_VM_DS://"
    error_msg, _auth_mgr = get_auth_mgr()
    if error_msg:
        return error_msg, None, None

    if _auth_mgr.allow_all_access():
        return None, auth_data_const.DEFAULT_TENANT_UUID, auth_data_const.DEFAULT_TENANT

    # If table "tenants", "vms", "privileges" or "volumes" does not exist
    # don't need auth check
    if not tables_exist():
        error_msg = "Required tables do not exist in auth db"
        logging.error(error_msg)
        return error_msg, None, None

    error_msg, tenant_uuid, tenant_name = get_tenant(vm_uuid)
    if error_msg:
        return error_msg, None, None

    if not tenant_uuid:
        # This VM does not associate any tenant(including DEFAULT tenant),
        # need reject the request
        vm_name = vmdk_utils.get_vm_name_by_uuid(vm_uuid)
        err_msg = error_code_to_message[ErrorCode.VM_NOT_BELONG_TO_TENANT].format(vm_name)
        logging.debug(err_msg)
        return err_msg, None, None
    else:
        error_msg, privileges = get_privileges(tenant_uuid, datastore_url)
        if error_msg:
            return error_msg, None, None

        result = check_privileges_for_command(cmd, opts, tenant_uuid, datastore_url, privileges)
        logging.debug("authorize: vmgroup_name=%s, datastore_url=%s, privileges=%s, result=%s",
                      tenant_name, datastore_url, privileges, result)

        if result is None:
            logging.info("db_mode='%s' cmd=%s opts=%s vmgroup=%s datastore_url=%s is allowed to execute",
                         _auth_mgr.mode, cmd, opts, tenant_name, datastore_url)

        return result, tenant_uuid, tenant_name

def add_volume_to_volumes_table(tenant_uuid, datastore_url, vol_name, vol_size_in_MB):
    """
        Insert volume to volumes table.
        Return None on success or error string.
    """
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg

    logging.debug("add to volumes table(%s %s %s %s)", tenant_uuid, datastore_url,
                  vol_name, vol_size_in_MB)

    if _auth_mgr.allow_all_access():
        logging.debug("Skipping Add volume to DB %s (allow_all_access)", tenant_uuid)
        if tenant_uuid != auth_data_const.DEFAULT_TENANT_UUID:
            return _auth_mgr.err_config_init_needed()
        else:
            logging.info("No access control, skipping volumes tracing in auth DB")
            return None

    try:
        _auth_mgr.conn.execute(
            "INSERT INTO volumes(tenant_id, datastore_url, volume_name, volume_size) VALUES (?, ?, ?, ?)",
            (tenant_uuid, datastore_url, vol_name, vol_size_in_MB)
            )
        _auth_mgr.conn.commit()
    except sqlite3.Error as e:
        logging.error("Error %s when insert into volumes table for tenant_id %s and datastore_url %s",
                      e, tenant_uuid, datastore_url)
        return str(e)

    return None

def remove_volume_from_volumes_table(tenant_uuid, datastore_url, vol_name):
    """
        Remove volume from volumes table.
        Return None on success or error string.
    """
    err_msg, _auth_mgr = get_auth_mgr()
    if err_msg:
        return err_msg

    logging.debug("remove volumes from volumes table(%s %s %s)", tenant_uuid, datastore_url,
                  vol_name)

    if _auth_mgr.allow_all_access():
        logging.debug("Skipping Rm volume from DB %s (allow_all_access)", tenant_uuid)
        return None

    try:
        _auth_mgr.conn.execute(
                    "DELETE FROM volumes WHERE tenant_id = ? AND datastore_url = ? AND volume_name = ?",
                    [tenant_uuid, datastore_url, vol_name]
            )
        _auth_mgr.conn.commit()
    except sqlite3.Error as e:
        logging.error("Error %s when remove from volumes table for tenant_id %s and datastore_url %s",
                      e, tenant_uuid, datastore_url)
        return str(e)

    return None

def get_row_from_tenants_table(conn, tenant_uuid):
    """
        Get a row from tenants table for a given tenant.

        Return value:
        -- error_msg: return None on success or error string
        -- result: returns a row in tenants table with given tenant_uuid on success,
           return None on failure
    """

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
    """
        Get rows from vms table for a given tenant.

        Return value:
        -- error_msg: return None on success or error string
        -- result: returns rows in vms table with given tenant_uuid on success,
           return None on failure
    """

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
    """
        Get rows from privileges table for a given tenant

        Return value:
        -- error_msg: return None on success or error string
        -- result: returns rows in privileges table with given tenant_uuid on success,
           return None on failure
    """

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

def main():
    log_config.configure()

if __name__ == "__main__":
    main()


