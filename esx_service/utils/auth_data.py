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
# limitations under the License

""" VM based authorization for docker volumes and tenant management.

"""

import sqlite3
import uuid
import os
import vmdk_utils
import vmdk_ops
import logging
import auth_data_const
import error_code
import threadutils
import log_config
import auth
from error_code import ErrorCode

AUTH_DB_PATH = '/etc/vmware/vmdkops/auth-db'

# DB schema and VMODL version
DB_MAJOR_VER = 1
DB_MINOR_VER = 0
VMODL_MAJOR_VER = 1
VMODL_MINOR_VER = 0

def all_columns_set(privileges):
        if not privileges:
            return False

        all_columns = [
                        auth_data_const.COL_DATASTORE_URL,
                        auth_data_const.COL_ALLOW_CREATE,
                        auth_data_const.COL_MAX_VOLUME_SIZE,
                        auth_data_const.COL_USAGE_QUOTA
                      ]
        for col in all_columns:
            if not col in privileges:
                return False

        return True

def get_version_str(major_ver, minor_ver):
    res = str(major_ver) + "." + str(minor_ver)
    return res

class DbConnectionError(Exception):
    """ An exception thrown when connection to a sqlite database fails. """

    def __init__(self, db_path):
        self.db_path = db_path

    def __str__(self):
        return "DB connection error %s" % self.db_path

class DatastoreAccessPrivilege:
    """ This class abstract the access privilege to a datastore .
    """
    def __init__(self, tenant_id, datastore_url, allow_create, max_volume_size, usage_quota):
            """ Constuct a DatastoreAccessPrivilege object. """
            self.tenant_id = tenant_id
            self.datastore_url = datastore_url
            self.allow_create = allow_create
            self.max_volume_size = max_volume_size
            self.usage_quota = usage_quota

def create_datastore_access_privileges(privileges):
    """
        Return a list of DatastoreAccessPrivilege object with given input
        @Param privileges: a list of dict dictionary, each element in this list represents
        a row in privileges table in auth-db
    """
    ds_access_privileges = []
    for p in privileges:
        dp = DatastoreAccessPrivilege(tenant_id = p[auth_data_const.COL_TENANT_ID],
                                      datastore_url = p[auth_data_const.COL_DATASTORE_URL],
                                      allow_create = p[auth_data_const.COL_ALLOW_CREATE],
                                      max_volume_size = p[auth_data_const.COL_MAX_VOLUME_SIZE],
                                      usage_quota = p[auth_data_const.COL_USAGE_QUOTA])
        ds_access_privileges.append(dp)

    return ds_access_privileges

def create_vm_list(vms):
    """
        Return a list of vm_uuid with given input
        @Param vms: a list of tuple, each tuple has format like this (vm_uuid,)
        Example:
        Input: [(u'564d6857-375a-b048-53b5-3feb17c2cdc4',), (u'564dca08-2772-aa20-a0a0-afae6b255fee',)]
        Output: [u'564d6857-375a-b048-53b5-3feb17c2cdc4', u'564dca08-2772-aa20-a0a0-afae6b255fee']
    """
    vm_list = []
    for v in vms:
        vm_list.append(v[0])

    return vm_list

class DockerVolumeTenant:
    """ This class abstracts the operations to manage a DockerVolumeTenant.

    The interfaces it provides includes:
    - add VMs to tenant
    - revmove VMs from tenant
    - change tenant name and description
    - set datastore and privileges for a tenant

    """

    def __init__(self, name, description, vms, privileges, id=None, default_datastore_url=None):
            """ Constuct a DockerVOlumeTenant object. """
            self.name = name
            self.description = description
            self.vms = vms
            self.privileges = privileges
            self.default_datastore_url=default_datastore_url
            if not id:
                self.id = str(uuid.uuid4())
            else:
                self.id = id

    def add_vms(self, conn, vms):
        """ Add vms in the vms table for this tenant. """
        tenant_id = self.id
        vms = [(vm_id, tenant_id) for (vm_id) in vms]
        if vms:
            try:
                conn.executemany(
                  "INSERT INTO vms(vm_id, tenant_id) VALUES (?, ?)",
                  vms
                )
                conn.commit()
            except sqlite3.Error as e:

                logging.error("Error %s when inserting into vms table with vms %s",
                              e, vms)
                return str(e)

        return None


    def remove_vms(self, conn, vms):
        """ Remove vms from the vms table for this tenant. """
        tenant_id = self.id
        vms = [(vm_id, tenant_id) for (vm_id) in vms]
        try:
            conn.executemany(
                    "DELETE FROM vms WHERE vm_id = ? AND tenant_id = ?",
                    vms
            )
            conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when removing from vms table with vms %s",
                          e, vms)
            return str(e)

        return None

    def replace_vms(self, conn, vms):
        """ Remove vms from the vms table for this tenant. """
        tenant_id = self.id
        vms = [(vm_id, tenant_id) for (vm_id) in vms]
        try:
            # Delete old VMs
            conn.execute(
                    "DELETE FROM vms WHERE tenant_id = ?",
                    [tenant_id]
            )

            conn.executemany(
                  "INSERT INTO vms(vm_id, tenant_id) VALUES (?, ?)",
                  vms
            )
            conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when replace vms table with vms %s",
                          e, vms)
            return str(e)

        return None

    def set_name(self, conn, name, new_name):
        """ Set name column in tenant table for this tenant. """
        logging.debug("set_name: name=%s, new_name=%s", name, new_name)
        tenant_id = self.id
        try:
            conn.execute(
                "UPDATE tenants SET name = ? WHERE id = ?",
                (new_name, tenant_id)
            )
            conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when updating tenants table with tenant_id"
                          "tenant_id %s", e, tenant_id)
            return str(e)

        # rename in the DB succeeds
        # rename the old symbol link /vmfs/volumes/datastore_name/tenant_name
        # to a new name /vmfs/volumes/datastore_name/new_tenant_name
        # which still point to path /vmfs/volumes/datastore_name/tenant_uuid
        for (datastore, url_name, path) in vmdk_utils.get_datastores():
            dock_vol_path = os.path.join("/vmfs/volumes", datastore, vmdk_ops.DOCK_VOLS_DIR)
            tenant_path = os.path.join(dock_vol_path, tenant_id)
            logging.debug("set_name: try to update the symlink to path %s", tenant_path)

            if os.path.isdir(tenant_path):
                exist_symlink_path = os.path.join(dock_vol_path, name)
                new_symlink_path = os.path.join(dock_vol_path, new_name)
                if os.path.isdir(exist_symlink_path):
                    logging.info("Renaming the symlink %s to %s", exist_symlink_path, new_symlink_path)
                    os.rename(exist_symlink_path, new_symlink_path)
                else:
                    logging.warning("symlink %s does not point to a directory", exist_symlink_path)
                    if not os.path.isdir(new_symlink_path):
                        os.symlink(tenant_path, new_symlink_path)
                        logging.info("Symlink %s is created to point to path %s", new_symlink_path, path)

        return None


    def set_description(self, conn, description):
        """ Set description column in tenant table for this tenant. """
        tenant_id = self.id
        try:
            conn.execute(
                "UPDATE tenants SET description = ? WHERE id = ?",
                (description, tenant_id)
                )
            conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when updating tenants table with tenant_id"
                          "tenant_id %s", e, tenant_id)
            return str(e)
        return None


    def set_default_datastore(self, conn, datastore_url):
        """ Set default_datastore for this tenant."""
        tenant_id = self.id
        try:
            conn.execute(
                "UPDATE tenants SET default_datastore_url = ? WHERE id = ?",
                (datastore_url, tenant_id)
                )
            conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when setting default datastore for tenant_id %s",
                          e, tenant_id)
            return str(e)
        return None

    def get_default_datastore(self, conn):
        """
        Get default_datastore for this tenant

        Return value:
            error_msg: return None on success or error info on failure
            datastore: return default_datastore name on success or None on failure
        """
        error_msg, result = auth.get_row_from_tenants_table(conn, self.id)
        if error_msg:
            logging.error("Error %s when getting default datastore for tenant_id %s",
                          error_msg, self.id)
            return str(error_msg), None
        else:
            datastore_url = result[auth_data_const.COL_DEFAULT_DATASTORE_URL]
            logging.debug("get_default_datastore: datastore_url=%s", datastore_url)
            if not datastore_url:
                # datastore_url read from DB is empty
                return None, None
            else:
                datastore = vmdk_utils.get_datastore_name(datastore_url)
                return None, datastore

    def set_datastore_access_privileges(self, conn, privileges):
        """ Set datastore and privileges for this tenant.

            "privileges"" is an array of dict
            each dict represent a privilege that the tenant has for a given datastore

            Example:
            privileges = [{'datastore_url': 'datastore1_url',
                           'allow_create': 1,
                           'max_volume_size': 0,
                           'usage_quota': 0},
                          {'datastore_url': 'datastore2_url',
                           'allow_create": 0,
                           'max_volume_size': 0,
                           'usage_quota': 0}]

        """
        tenant_id = self.id
        for p in privileges:
            p[auth_data_const.COL_TENANT_ID] = tenant_id
            if not all_columns_set(p):
                return "Not all columns are set in 'privileges''"

        try:
            conn.executemany(
                """
                INSERT OR IGNORE INTO privileges VALUES
                (:tenant_id, :datastore_url, :allow_create,
                 :max_volume_size, :usage_quota)
                """,
                privileges
            )

            for p in privileges:
                # privileges ia an array of dict
                # each dict represent a privilege that the tenant has for a given datastore
                # for each dict, add a new element which maps 'tenant_id' to tenant_id
                p[auth_data_const.COL_TENANT_ID] = tenant_id
                column_list = ['tenant_id', 'datastore_url', 'allow_create',
                               'max_volume_size', 'usage_quota']
                update_list = []
                update_list = [p[col] for col in column_list]
                update_list.append(tenant_id)
                update_list.append(p[auth_data_const.COL_DATASTORE_URL])

                logging.debug("set_datastore_access_privileges: update_list %s", update_list)

                conn.execute(
                    """
                    UPDATE OR IGNORE privileges SET
                        tenant_id = ?,
                        datastore_url = ?,
                        allow_create = ?,
                        max_volume_size = ?,
                        usage_quota = ?
                    WHERE tenant_id = ? AND datastore_url = ?
                    """,
                    update_list
                )
            conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when setting datastore and privileges for tenant_id %s",
                          e, tenant_id)
            return str(e)

        return None

    def remove_datastore_access_privileges(self, conn, datastore_url):
        """ Remove privileges from privileges table for this tenant. """
        tenant_id = self.id
        try:
            conn.execute(
                    "DELETE FROM privileges WHERE tenant_id = ? AND datastore_url = ?",
                    [tenant_id, datastore_url]
            )
            conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when removing from privileges table with tenant_id%s and "
                "datastore %s", e, tenant_id, datastore_url)
            return str(e)

        return None


class AuthorizationDataManager:
    """ This class abstracts the creation, modification and retrieval of
    authorization data used by vmdk_ops as well as the VMODL interface for
    Docker volume management.

    init arg:
    The constructor of this class takes "db_path" as an argument.
    "db_path" specifies the path of sqlite3 database
    If caller does not pass the value of this argument, function "get_auth_db_path"
    will be called to figure out the value
    otherwise, the value passed by caller will be used

    """

    def __init__(self, db_path=None):
        if not db_path:
            self.db_path = self.get_auth_db_path()
        else:
            self.db_path = db_path
        self.conn = None

    def __del__(self):
        if self.conn:
            self.conn.close()

    def get_auth_db_path(self):
        """ Return the path of authorization database.

        DB tables should be stored in VSAN datastore
        DB file should be stored under /vmfs/volume/VSAN_datastore/
        See issue #618
        Currently, it is hardcoded.

        """
        return AUTH_DB_PATH

    def create_default_tenant(self):
        """ Create DEFAULT tenant """
        error_msg, tenant = self.create_tenant(
                                           name=auth.DEFAULT_TENANT,
                                           description="This is a default tenant",
                                           vms=[],
                                           privileges=[])
        if error_msg:
            err = error_code.error_code_to_message[ErrorCode.TENANT_CREATE_FAILED].format(auth.DEFAULT_TENANT, error_msg)
            logging.warning(err)

    def create_default_privileges(self):
        """
        create DEFAULT privilege
        This privilege will match any <tenant(DEFAULT and normal), datastore> pair
        which does not have an entry in privileges table explicitly
        this privilege will have full permission (create, delete, and mount)
        and no max_volume_size and usage_quota limitation
        """
        privileges = [{'datastore_url': auth.DEFAULT_DS_URL,
                       'allow_create': 1,
                       'max_volume_size': 0,
                       'usage_quota': 0}]

        error_msg, tenant = self.get_tenant(auth.DEFAULT_TENANT)
        if error_msg:
            err = error_code.error_code_to_message[ErrorCode.TENANT_NOT_EXIST].format(auth.DEFAULT_TENANT)
            logging.warning(err)
            return

        error_msg = tenant.set_datastore_access_privileges(self.conn, privileges)
        if error_msg:
            err = error_code.error_code_to_message[ErrorCode.TENANT_SET_ACCESS_PRIVILEGES_FAILED].format(auth.DEFAULT_TENANT, auth.DEFAULT_DS, error_msg)
            logging.warning(err)

    def get_db_version(self):
        """
        Get DB schema version from auth-db
        """
        major_ver = 0
        minor_ver = 0

        # check table "versions" exist or not
        # if table "versions" does not exist, return major_ver=0 and minor_ver=0
        try:
            cur = self.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'versions';")
            result = cur.fetchall()
        except sqlite3.Error as e:
            logging.error("Error %s when checking whether table versions exists or not", e)
            return str(e), major_ver, minor_ver

        if not result:
        # table "versions" does not exist
            return None, major_ver, minor_ver

        try:
            cur = self.conn.execute("SELECT * FROM versions WHERE id = 0 ")
            result = cur.fetchall()
        except sqlite3.Error as e:
            logging.error("Error %s when querying from table tenants versions", e)
            return str(e), major_ver, minor_ver

        major_ver = result[0][1]
        minor_ver = result[0][2]
        logging.debug("get_db_version: major_ver=%d minor_ver=%d", major_ver, minor_ver)
        return None, major_ver, minor_ver

    def need_upgrade_db(self):
        """
            Check whether auth-db schema need to be upgraded or not
        """
        major_ver = 0
        minor_ver = 0
        error_msg, major_ver, minor_ver = self.get_db_version()
        if error_msg:
            logging.error("need_upgrade_db: fail to get version info of auth-db, cannot do upgrade")
            return False

        if major_ver != DB_MAJOR_VER or minor_ver != DB_MINOR_VER:
            auth_db_ver = get_version_str(major_ver, minor_ver)
            curr_db_ver = get_version_str(DB_MAJOR_VER, DB_MINOR_VER)
            logging.error("version %s in auth-db does not match latest DB version %s",
                          auth_db_ver, curr_db_ver)
            logging.error("DB upgrade is not supported. Please remove the DB file at %s. All existing configuration "
                          "will be removed and need to be recreated after removing the DB file.", AUTH_DB_PATH)
            return True

        return False

    def connect(self):
        """ Connect to a sqlite database file given by `db_path`.

        Ensure foreign key
        constraints are enabled on the database and set the return type for
        select operations to dict-like 'Rows' instead of tuples.

        Raises a ConnectionFailed exception when connect fails.

        """
        need_create_table = False

        if not os.path.isfile(self.db_path):
            logging.debug("auth DB %s does not exist, try to create table", self.db_path)
            need_create_table = True

        self.conn = sqlite3.connect(self.db_path)

        if not self.conn:
            raise DbConnectionError(self.db_path)

        if not need_create_table:
            if os.path.isfile(self.db_path) and self.need_upgrade_db():
                # Currently, when need upgrade, raise exception
                # TODO: add code to handle DB schema upgrade
                # when schema changes
                raise DbConnectionError(self.db_path)

        # Return rows as Row instances instead of tuples
        self.conn.row_factory = sqlite3.Row

        if need_create_table:
            self.create_tables()
            self.create_default_tenant()
            self.create_default_privileges()

    def create_tables(self):
        """ Create tables used for per-datastore authorization.

        This function should only be called once per datastore.
        It will raise an exception if the schema file isn't
        accessible or the tables already exist.

        """
        try:
            self.conn.execute(
                '''
                    PRAGMA foreign_key = ON;
                '''

            )
            self.conn.execute(
                '''
                    CREATE TABLE tenants(
                    -- uuid for the tenant, which is generated by create_tenant() API
                    id TEXT PRIMARY KEY NOT NULL,
                    -- name of the tenant, which is specified by user when creating the tenant
                    -- this field can be changed later by using set_name() API
                    name TEXT UNIQUE NOT NULL,
                    -- brief description of the tenant, which is specified by user when creating the tenant
                    -- this field can be changed laster by using set_description API
                    description TEXT,
                    -- default_datastore url
                    default_datastore_url TEXT
                    )
                '''
            )

            self.conn.execute(
                '''
                CREATE TABLE vms(
                -- uuid for the VM, which is generated when VM is created
                -- this uuid will be passed in to executeRequest()
                -- this field need to be specified when adding a VM to a tenant
                vm_id TEXT PRIMARY KEY NOT NULL,
                -- id in tenants table
                tenant_id TEXT NOT NULL,
                FOREIGN KEY(tenant_id) REFERENCES tenants(id)
                );
                '''
            )


            self.conn.execute(
                '''
                CREATE TABLE privileges(
                -- id in tenants table
                tenant_id TEXT NOT NULL,
                -- datastore url
                datastore_url TEXT NOT NULL,
                -- a boolean value, if it is set to True, tenant has full
                -- privilege on this datastore; it it is set to False
                -- tenant only has mount/unmount privilege on this datastore
                allow_create INTEGER,
                -- The unit of "max_volume_size" is "MB"
                max_volume_size INTEGER,
                -- The unit of usage_quota is "MB"
                usage_quota INTEGER,
                PRIMARY KEY (tenant_id, datastore_url),
                FOREIGN KEY(tenant_id) REFERENCES tenants(id)
                );
                '''
            )

            self.conn.execute(
                '''
                CREATE TABLE volumes (
                -- id in tenants table
                tenant_id TEXT NOT NULL,
                -- datastore url
                datastore_url TEXT NOT NULL,
                volume_name TEXT,
                -- The unit of "volume_size" is "MB"
                volume_size INTEGER,
                PRIMARY KEY(tenant_id, datastore_url, volume_name),
                FOREIGN KEY(tenant_id) REFERENCES tenants(id)
                );
                '''
            )

            self.conn.execute(
                '''
                CREATE TABLE versions (
                id INTEGER PRIMARY KEY NOT NULL,
                -- DB major version
                major_ver INTEGER NOT NULL,
                -- DB minor version
                minor_ver INTEGER NOT NULL,
                -- VMODL major version
                vmodl_major_ver INTEGER NOT NULL,
                -- VMODL minor version
                vmodl_minor_ver INTEGER NOT NULL
                );
                '''
            )

            # insert latest DB version and VMODL version to table "versions"
            self.conn.execute(
                " INSERT INTO versions(id, major_ver, minor_ver, vmodl_major_ver, vmodl_minor_ver) VALUES (?, ?, ?, ?, ?)",
                (0, DB_MAJOR_VER, DB_MINOR_VER, VMODL_MAJOR_VER, VMODL_MINOR_VER)
            )

            self.conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when creating auth DB tables", e)
            return str(e)

        return None

    def create_tenant(self, name, description, vms, privileges):
        """ Create a tenant in the database.

        A tenant id will be auto-generated and returned.
        vms are list of vm_id. Privileges are dictionaries
        with keys matching the row names in the privileges table. Tenant id is
        filled in for both the vm and privileges tables.

        """
        logging.debug ("create_tenant name=%s", name)
        if privileges:
            for p in privileges:
                if not all_columns_set(p):
                    error_msg = "Not all columns are set in privileges"
                    return error_msg, None

        # Create the entry in the tenants table
        default_datastore_url=""
        tenant = DockerVolumeTenant(name=name,
                                    description=description,
                                    vms=vms,
                                    privileges=privileges,
                                    default_datastore_url=default_datastore_url)

        id = tenant.id
        try:
            self.conn.execute(
                "INSERT INTO tenants(id, name, description, default_datastore_url) VALUES (?, ?, ?, ?)",
                (id, name, description, default_datastore_url)
            )

            # Create the entries in the vms table
            vms = [(vm_id, id) for (vm_id) in vms]

            if vms:
                self.conn.executemany(
                "INSERT INTO vms(vm_id, tenant_id) VALUES (?, ?)",
                vms
                )

            if privileges:
                for p in privileges:
                    p[auth_data_const.COL_TENANT_ID] = id

                self.conn.executemany(
                    """
                    INSERT INTO privileges VALUES
                    (:tenant_id, :datastore_url, :allow_create,
                    :max_volume_size, :usage_quota)
                    """,
                    privileges
                )
            self.conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when creating tenant for tenant_name %s tenant_id %s",
                          e, tenant.name, tenant.id)
            return str(e), tenant

        return None, tenant

    def get_tenant(self, tenant_name):
        """ Return a DockerVolumeTenant object which match the given tenant_name """
        logging.debug("get_tenant: tenant_name=%s", tenant_name)
        tenant = None
        try:
            cur = self.conn.execute(
                "SELECT * FROM tenants WHERE name = ?",
                (tenant_name,)
            )
            result = cur.fetchall()

            for r in result:
                # loop through each tenant
                id = r[auth_data_const.COL_ID]
                name = r[auth_data_const.COL_NAME]
                description = r[auth_data_const.COL_DESCRIPTION]
                default_datastore_url = r[auth_data_const.COL_DEFAULT_DATASTORE_URL]

                logging.debug("id=%s name=%s description=%s default_datastore_url=%s",
                              id, name, description, default_datastore_url)

                # search vms for this tenant
                vms = []
                cur = self.conn.execute(
                    "SELECT * FROM vms WHERE tenant_id = ?",
                    (id,)
                )
                vms = cur.fetchall()

                logging.debug("vms=%s", vms)
                vm_list = create_vm_list(vms)

                # search privileges and default_privileges for this tenant
                privileges = []
                cur = self.conn.execute(
                    "SELECT * FROM privileges WHERE tenant_id = ?",
                    (id,)
                )
                privileges = cur.fetchall()
                ds_access_privileges = create_datastore_access_privileges(privileges)

                logging.debug("privileges=%s", privileges)
                logging.debug("ds_access_privileges=%s", ds_access_privileges)

                tenant = DockerVolumeTenant(name=name,
                                            description=description,
                                            vms=vm_list,
                                            privileges=ds_access_privileges,
                                            id=id,
                                            default_datastore_url=default_datastore_url)
        except sqlite3.Error as e:
            logging.error("Error %s when get tenant %s", e, tenant_name)
            return str(e), tenant

        return None, tenant

    def list_tenants(self):
        """ Return a list of DockerVolumeTenants objects. """
        tenant_list = []
        try:
            cur = self.conn.execute(
            "SELECT * FROM tenants"
            )
            result = cur.fetchall()

            for r in result:
                # loop through each tenant
                id = r[auth_data_const.COL_ID]
                name = r[auth_data_const.COL_NAME]
                description = r[auth_data_const.COL_DESCRIPTION]
                default_datastore_url = r[auth_data_const.COL_DEFAULT_DATASTORE_URL]

                # search vms for this tenant
                vms = []
                cur = self.conn.execute(
                    "SELECT * FROM vms WHERE tenant_id = ?",
                    (id,)
                )
                vms = cur.fetchall()
                vm_list = create_vm_list(vms)
                # search privileges and default_privileges for this tenant
                privileges = []
                cur = self.conn.execute(
                    "SELECT * FROM privileges WHERE tenant_id = ?",
                    (id,)
                )
                privileges = cur.fetchall()
                ds_access_privileges = create_datastore_access_privileges(privileges)

                logging.debug("privileges=%s", privileges)
                logging.debug("ds_access_privileges=%s", ds_access_privileges)

                tenant = DockerVolumeTenant(name=name,
                                            description=description,
                                            vms=vm_list,
                                            privileges=ds_access_privileges,
                                            id=id,
                                            default_datastore_url=default_datastore_url)
                tenant_list.append(tenant)
        except sqlite3.Error as e:
            logging.error("Error %s when listing all tenants", e)
            return str(e), tenant_list

        return None, tenant_list

    def remove_volumes_from_volumes_table(self, tenant_id):
        """ Remove all volumes from volumes table. """
        try:
            self.conn.execute(
                    "DELETE FROM volumes WHERE tenant_id = ?",
                    [tenant_id]
            )

            self.conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when removing volumes from volumes table for tenant_id %s",
                          e, tenant_id)
            return str(e)

        return None

    def _remove_volumes_for_tenant(self, tenant_id):
        """ Delete all volumes belongs to this tenant.

            Do not use it outside of removing a tenant.

        """
        try:
            cur = self.conn.execute(
            "SELECT name FROM tenants WHERE id = ?",
            (tenant_id,)
            )
            result = cur.fetchone()
        except sqlite3.Error as e:
            logging.error("Error %s when querying from tenants table", e)
            return str(e)

        error_msg = ""
        if result:
            logging.debug("remove_volumes_for_tenant: %s %s", tenant_id, result)
            tenant_name = result[0]
            vmdks = vmdk_utils.get_volumes(tenant_name)
            # Delete all volumes for this tenant.
            dir_paths = set()
            for vmdk in vmdks:
                vmdk_path = os.path.join(vmdk['path'], "{0}".format(vmdk['filename']))
                dir_paths.add(vmdk['path'])
                logging.debug("path=%s filename=%s", vmdk['path'], vmdk['filename'])
                logging.debug("Deleting volume path%s", vmdk_path)
                err = vmdk_ops.removeVMDK(vmdk_path=vmdk_path,
                                          vol_name=vmdk_utils.strip_vmdk_extension(vmdk['filename']),
                                          vm_name=None,
                                          tenant_uuid=tenant_id,
                                          datastore=vmdk['datastore'])
                if err:
                    logging.error("remove vmdk %s failed with error %s", vmdk_path, err)
                    error_msg += str(err)

            VOL_RM_LOG_PREFIX = "Tenant <name> %s removal: "
            # delete the symlink /vmfs/volume/datastore_name/tenant_name
            # which point to /vmfs/volumes/datastore_name/tenant_uuid
            for (datastore, url_name, path) in vmdk_utils.get_datastores():
                dock_vol_path = os.path.join("/vmfs/volumes", datastore, vmdk_ops.DOCK_VOLS_DIR)
                tenant_path = os.path.join(dock_vol_path, tenant_id)
                logging.debug(VOL_RM_LOG_PREFIX + "try to remove symlink to %s", tenant_name, tenant_path)

                if os.path.isdir(tenant_path):
                    exist_symlink_path = os.path.join(dock_vol_path, tenant_name)
                    if os.path.isdir(exist_symlink_path):
                        os.remove(exist_symlink_path)
                        logging.debug(VOL_RM_LOG_PREFIX + "removing symlink %s", tenant_name, exist_symlink_path)

            # Delete path /vmfs/volumes/datastore_name/tenant_uuid
            logging.debug("Deleting dir paths %s", dir_paths)
            for path in list(dir_paths):
                try:
                    os.rmdir(path)
                except os.error as e:
                    msg = "remove dir {0} failed with error {1}".format(path, e)
                    logging.error(msg)
                    error_msg += str(err)

        err = self.remove_volumes_from_volumes_table(tenant_id)
        if err:
            logging.error("Failed to remove volumes from database %s", err)
            error_msg += str(err)

        if error_msg:
            return error_msg

        return None

    def remove_tenant(self, tenant_id, remove_volumes):
        """ Remove a tenant with given id.

            A row with given tenant_id will be removed from table tenants, vms,
            and privileges.
            All the volumes created by this tenant will be removed if remove_volumes
            is set to True.

        """
        logging.debug("remove_tenant: tenant_id%s, remove_volumes=%d", tenant_id, remove_volumes)
        if remove_volumes:
            error_msg = self._remove_volumes_for_tenant(tenant_id)
            if error_msg:
                return error_msg

        try:
            self.conn.execute(
                    "DELETE FROM vms WHERE tenant_id = ?",
                    [tenant_id]
            )

            self.conn.execute(
                    "DELETE FROM privileges WHERE tenant_id = ?",
                    [tenant_id]
            )
            self.conn.execute(
                    "DELETE FROM tenants WHERE id = ?",
                    [tenant_id]
            )

            self.conn.commit()
        except sqlite3.Error as e:
            logging.error("Error %s when removing tables", e)
            return str(e)

        return None

    def get_tenant_name(self, tenant_uuid):
        """ Return tenant_name which match the given tenant_uuid """
        logging.debug("get_tenant_name: tenant_uuid=%s", tenant_uuid)
        try:
            cur = self.conn.execute(
                "SELECT * FROM tenants WHERE id=?",
                (tenant_uuid,)
                )
        except sqlite3.Error as e:
            logging.error("Error: %s when querying tenants table for tenant %s",
                        e, tenant_uuid)
            return str(e), None

        result = cur.fetchone()
        if result:
            tenant_name = result[1]
            logging.debug("get_tenant_name: tenant_uuid=%s tenant_name=%s", tenant_uuid, tenant_name)
            return None, tenant_name
        else:
            error_msg =  error_code.error_code_to_message[ErrorCode.TENANT_NAME_NOT_FOUND].format(tenant_uuid)
            logging.debug("get_tenant_name:"+error_msg)
            return error_msg, None

def main():
    log_config.configure()

if __name__ == "__main__":
    main()
