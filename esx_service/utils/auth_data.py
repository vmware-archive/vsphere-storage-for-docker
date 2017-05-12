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

"""
VM based authorization for docker volumes and tenant management.

Note that for external consumption we refer to a 'tenant' as a 'vmgroup'.
This way the code operates 'tenants' but user/admin operates 'vmgroups'
"""

import sqlite3
import uuid
import os
import vmdk_utils
import vmdk_ops
import logging
import auth_data_const
import threadutils
import log_config
import auth
from error_code import ErrorCode
from error_code import error_code_to_message

AUTH_DB_PATH = '/etc/vmware/vmdkops/auth-db' # location of auth.db symlink
CONFIG_DB_NAME = "vmdkops_config.db"         # name of the configuration DB file
DB_STATE_CHECK_SEC = 300 # interval to check for DB state, in seconds
DB_REF = "Config DB "  # we will use it in logging

# DB schema and VMODL version
# Bump the DB_MINOR_VER to 1.2
# in DB version 1.1, _DEFAULT_TENANT will be created using a constant UUID
# in DB version 1.2, VM name is persisted along with VM uuid in the vms table
DB_MAJOR_VER = 1
DB_MINOR_VER = 2
VMODL_MAJOR_VER = 1
VMODL_MINOR_VER = 0

UPGRADE_README = "https://github.com/vmware/docker-volume-vsphere/blob/master/docs/misc/UpgradeFrom_Pre0.11.1.md"


class DBMode(object):
    """
    Modes of Auth DB. Calculated on first access and used to make decisions
    about the need for authentication.
    TODO: Also will be used in changes discovery.
    """

    Unknown = 0        # The value was not calculated yet
    NotConfigured = 1  # path does not exist, or an empty DB (which we delete)
                       # this turn on "no access control" mode for external code
    SingleNode = 2     # path exists and is a non-empty DB. "legacy singe ESX" mode
    MultiNode = 3      # Normal op, symlink to nonempty DB. Used for a DB on shared storage,
    BrokenLink = 4     # symlink exists but points nowhere, or it's a file which is not the DB
    str_dict = {       # strings for print when requested:
        Unknown: "Unknown (not checked yet)",
        NotConfigured: "NotConfigured (no local DB, no symlink to shared DB)",
        SingleNode: "SingleNode (local DB exists)",
        MultiNode: "MultiNode (local symlink pointing to shared DB)",
        BrokenLink: "BrokenLink (Broken link or corrupted DB file)"
    }
    def __init__(self, value=Unknown):
        self.value = value
    def __str__(self):
        return DBMode.str_dict[self.value]
    def __eq__(self, other):
        return self.value == other


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

def get_dockvol_path_tenant_path(datastore_name, tenant_id):
    """ Return dockvol path and tenant_path for given datastore and tenant """

    # dockvol_path has the format like "/vmfs/volumes/<datastore_name>"
    # tenant_path has the format like "/vmfs/volumes/<datastore_name>/tenant_id"
    dockvol_path = os.path.join("/vmfs/volumes", datastore_name, vmdk_ops.DOCK_VOLS_DIR)
    tenant_path = os.path.join(dockvol_path, tenant_id)
    return dockvol_path, tenant_path

class DbConnectionError(Exception):
    """ Thrown when a client tries to establish a connection to the DB. """
    def __init__(self, path):
        super(DbConnectionError, self).__init__("DB connection error at {}".format(path))

class DbAccessError(Exception):
    """ Thrown when a client fails to run a SQL using an established connection. """
    def __init__(self, db_path, msg):
        super(DbAccessError, self).__init__("DB access error at {}: {}".format(db_path, msg))

class DbUpgradeError(Exception):
    """ Thrown when a client fails to upgrade existing db"""
    def __init__(self, db_path, msg):
        super(DbUpgradeError, self).__init__("DB upgrade error at {}: {}".format(db_path, msg))

class DatastoreAccessPrivilege:
    """
    This class abstracts the access privilege to a datastore.
    """
    def __init__(self, tenant_id, datastore_url, allow_create, max_volume_size, usage_quota):
        """ Construct a DatastoreAccessPrivilege object. """
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
        dp = DatastoreAccessPrivilege(tenant_id=p[auth_data_const.COL_TENANT_ID],
                                      datastore_url=p[auth_data_const.COL_DATASTORE_URL],
                                      allow_create=p[auth_data_const.COL_ALLOW_CREATE],
                                      max_volume_size=p[auth_data_const.COL_MAX_VOLUME_SIZE],
                                      usage_quota=p[auth_data_const.COL_USAGE_QUOTA])
        ds_access_privileges.append(dp)

    return ds_access_privileges

def create_vm_list(vms):
    """
        Return a list of (vm_uuid, vm_name) with given input
        @Param vms: a list of sqlite3.Row tuple.
        Each tuple has format like this (vm_uuid, tenant_id, vm_name)
        Example:
        Input: [(u'564d6857-375a-b048-53b5-3feb17c2cdc4', u'88d98e7a-8421-45c8-a2c4-7ffb6c437a4f', 'ubuntu-VM0.1'), (u'564dca08-2772-aa20-a0a0-afae6b255fee', u'88d98e7a-8421-45c8-a2c4-7ffb6c437a4f', 'ubuntu-VM1.1')]
        Output: [(u'564d6857-375a-b048-53b5-3feb17c2cdc4', 'ubuntu-VM0.1'), (u'564dca08-2772-aa20-a0a0-afae6b255fee', 'ubuntu-VM0.1')]
    """
    return [(v[0], v[2]) for v in vms]

class DockerVolumeTenant:
    """ This class abstracts the operations to manage a DockerVolumeTenant.

    The interfaces it provides includes:
    - add VMs to tenant
    - revmove VMs from tenant
    - change tenant name and description
    - set datastore and privileges for a tenant

    """

    def __init__(self, name, description, vms, privileges, id=None, default_datastore_url=None):
        """ Construct a DockerVolumeTenant object. """
        self.name = name
        self.description = description
        self.vms = vms
        self.privileges = privileges
        self.default_datastore_url = default_datastore_url
        if not id:
            self.id = str(uuid.uuid4())
        else:
            self.id = id

    def add_vms(self, conn, vms):
        """ Add vms in the vms table for this tenant. """
        tenant_id = self.id
        vms = [(vm_id, vm_name, tenant_id) for vm_id, vm_name in vms]
        if vms:
            try:
                conn.executemany(
                    "INSERT INTO vms(vm_id, vm_name, tenant_id) VALUES (?, ?, ?)",
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
        vms = [(vm_id, tenant_id) for vm_id, _ in vms]
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
        """ Update vms from the vms table which belong to this tenant. """
        tenant_id = self.id
        vms = [(vm_id, vm_name, tenant_id) for vm_id, vm_name in vms]
        try:
            # Delete old VMs
            conn.execute(
                "DELETE FROM vms WHERE tenant_id = ?",
                [tenant_id]
            )

            conn.executemany(
                "INSERT INTO vms(vm_id, vm_name, tenant_id) VALUES (?, ?, ?)",
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
        for (datastore, url, path) in vmdk_utils.get_datastores():
            dockvol_path, tenant_path = get_dockvol_path_tenant_path(datastore_name=datastore,
                                                                     tenant_id=tenant_id)
            logging.debug("set_name: try to update the symlink to path %s", tenant_path)

            if os.path.isdir(tenant_path):
                exist_symlink_path = os.path.join(dockvol_path, name)
                new_symlink_path = os.path.join(dockvol_path, new_name)
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
        logging.debug("set_default_datastore: for tenant=%s to datastore=%s", self.id, datastore_url)
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
        Get default_datastore url for this tenant

        Return value:
            error_msg: return None on success or error info on failure
            datastore_url: return default_datastore url on success or None on failure
        """
        error_msg, result = auth.get_row_from_tenants_table(conn, self.id)
        if error_msg:
            logging.error("Error %s when getting default datastore for tenant_id %s",
                          error_msg, self.id)
            return str(error_msg), None
        else:
            datastore_url = result[auth_data_const.COL_DEFAULT_DATASTORE_URL]
            logging.debug("auth.data.get_default_datastore: datastore_url=%s", datastore_url)
            if not datastore_url:
                # datastore_url read from DB is empty
                return None, None
            else:
                return None, datastore_url

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
    """
    This class abstracts the creation, modification and retrieval of
    authorization data used by vmdk_ops as well as the VMODL interface for
    Docker volume management.
    """

    @classmethod
    def ds_to_db_path(cls, datastore):
        """Form full path to DB based on expected location on the datastore"""
        return os.path.join("/vmfs/volumes", datastore, vmdk_ops.DOCK_VOLS_DIR, CONFIG_DB_NAME)

    def __init__(self, db_path=AUTH_DB_PATH):
        """
        The AuthorizationDataManager constructor gets passed an optional "db_path".
        "db_path" specifies the path of sqlite3 database. Default is defined in AUTH_DB_PATH.
        Note that regular control flow is as follows:
        'auth=AuthorizationDataManager(); auth.connect(); auth.<use>'.
        For creating a DB though, the control flow is as follows:
        'with  AuthorizationDataManager()as auth ; auth.new_db(). The create and use new instance.
        """

        # Note: Eventually we will place the DB file on VSAN ,most likely
        # under /vmfs/volume/vsanDatastore/DOCKVOL/etc/....
        # For now we refer to it from a fixed place in AUTH_DB_PATH
        # See issue #618 for more details.

        self.db_path = db_path
        self.conn = self.__mode = None

    def __del__(self):
        """ Destructor. For now, only closes the connection"""
        return self.__close()

    def __enter__(self):
        """Support for 'with' statement"""
        return self

    def __exit__(self, exception_type, exception_value, traceback):
        """Support for 'with' statement"""
        pass

    def __close(self):
        """ Close the connection to the DB"""
        if self.conn:
            self.conn.close()
            self.conn = None

    def __get_db_version(self):
        """
        Get DB schema version from auth-db
        """
        major_ver = minor_ver = 0

        # Check if "versions" exists.
        # If table "versions" does not exist, return major_ver=0 and minor_ver=0.
        try:
            cur = self.conn.execute("SELECT name FROM sqlite_master WHERE type = 'table' and name = 'versions';")
            result = cur.fetchall()
        except sqlite3.Error as e:
            logging.error("Error %s when checking whether table versions exists or not", e)
            return str(e), major_ver, minor_ver

        if not result:
        # Table "versions" does not exist.
            return None, major_ver, minor_ver

        try:
            cur = self.conn.execute("SELECT * FROM versions WHERE id = 0 ")
            result = cur.fetchall()
        except sqlite3.Error as e:
            logging.error("Error %s when querying from table tenants versions", e)
            return str(e), major_ver, minor_ver

        major_ver = result[0][1]
        minor_ver = result[0][2]
        logging.debug("__get_db_version: major_ver=%d minor_ver=%d", major_ver, minor_ver)
        return None, major_ver, minor_ver

    def __need_upgrade_db(self):
        """
            Check whether auth-db schema need to be upgraded or not
        """
        major_ver = 0
        minor_ver = 0
        error_msg, major_ver, minor_ver = self.__get_db_version()
        if error_msg:
            logging.error("__need_upgrade_db: fail to get version info of auth-db, cannot do upgrade")
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

    def _handle_upgrade_1_0_to_1_1(self):
        """Handle auth DB upgrade from version 1.0 to version 1.1"""
        # in DB version 1.0, _DEFAULT_TENANT is created using a random generated UUID
        # in DB version 1.1, _DEFAULT_TENANT is created using a constant UUID
        error_msg, tenant = self.get_tenant(auth_data_const.DEFAULT_TENANT)
        if error_msg:
            raise DbAccessError(self.db_path, error_msg)
        error_msg = """
                        Your ESX installation seems to be using configuration DB created by previous
                        version of vSphere Docker Volume Service, and requires upgrade.
                        See {0} for more information.  (_DEFAULT_UUID = {1}, expected = {2})
                     """.format(UPGRADE_README, tenant.id, auth_data_const.DEFAULT_TENANT_UUID)
        logging.error(error_msg)
        raise DbAccessError(self.db_path, error_msg)


    def handle_upgrade_1_1_to_1_2(self):
        """
        Upgrade the db from version 1.1 to 1.2
        In 1.1 the vms table had two columns namely vm_id and tenant_id
        In 1.2 the vms table has three columns namely vm_id, tenant_id and vm_name
        In the process of upgrading existing db from 1.1 to 1.2, we need to populate
        the vm names for existing records. We try to populate them and keep None for vms
        for which the name couldn't be found. The vmdk_ops admin code which tries to use
        this vm names handles names which are None
        In 1.2, "default_datastore" field must be set in tenants table, so the upgrade process
        will try to set the "default_datastore" field if needed
        In 1.2, for each tenant in tenants table, a privilege to "default_datastore" must be
        present, the upgrade process will try to create this privilege if needed
        In 1.2, for "_DEFAULT" tenant, privilege to "_DEFAULT_DS" need to be removed, and privilege
        to "__VM_DS" and "__ALL_DS" need to be inserted
        """
        try:
            logging.info("handle_upgrade_1_1_to_1_2: Start")
            self.conn.create_function('name_from_uuid', 1, vmdk_utils.get_vm_name_by_uuid)
            # Alter vms table to add a new column name vm_name to store vm name
            # update all the existing records with the vm_name.
            # If vm_name is not resolved, it is populated as None and handled appropriately later.
            # Finally update the db schema version
            script = """ALTER TABLE vms ADD COLUMN vm_name TEXT;
                        UPDATE vms SET vm_name=name_from_uuid(vm_id);
                        UPDATE versions SET major_ver = {}, minor_ver = {};
                     """
            sql_script = script.format(DB_MAJOR_VER, DB_MINOR_VER)
            self.conn.executescript(sql_script)

            logging.info("handle_upgrade_1_1_to_1_2: update vms table Done")

            # update the tenants table to set "default_datastore" to "__VM_DS" if "default_datastore" is ""
            self.conn.execute("UPDATE OR IGNORE tenants SET default_datastore_url = ?  where default_datastore_url = \"\"",
                              (auth_data_const.VM_DS_URL,))
            logging.info("handle_upgrade_1_1_to_1_2: update default_datastore in tenants table")

            cur = self.conn.execute("SELECT * FROM tenants")
            result = cur.fetchall()

            self.conn.execute("""INSERT OR IGNORE INTO privileges(tenant_id, datastore_url, allow_create, max_volume_size, usage_quota)
                                     SELECT tenants.id, tenants.default_datastore_url, 1, 0, 0 FROM tenants
                              """)
            logging.info("handle_upgrade_1_1_to_1_2: Insert privilege to default_datastore in privileges table")

            cur = self.conn.execute("SELECT * FROM tenants WHERE id = ?",
                                    (auth_data_const.DEFAULT_TENANT_UUID,)
                                   )

            result = cur.fetchall()
            logging.debug("handle_upgrade_1_1_to_1_2: Check DEFAULT tenant exist")
            if result:
                # _DEFAULT tenant exists
                # insert full access privilege to "__ALL_DS" for "_DEFAULT" tenant
                all_ds_privilege = (auth_data_const.DEFAULT_TENANT_UUID, auth_data_const.ALL_DS_URL, 1, 0, 0)
                self.conn.execute("INSERT INTO privileges(tenant_id, datastore_url, allow_create, max_volume_size, usage_quota) VALUES (?, ?, ?, ?, ?)",
                                  all_ds_privilege)
                logging.info("handle_upgrade_1_1_to_1_2: Insert privilege to __ALL_DS for _DEFAULT tenant in privileges table")
                # remove access privilege to "DEFAULT_DS"
                self.conn.execute("DELETE FROM privileges WHERE tenant_id = ? AND datastore_url = ?",
                                [auth_data_const.DEFAULT_TENANT_UUID, auth_data_const.DEFAULT_DS_URL])
                logging.info("handle_upgrade_1_1_to_1_2: Remove privilege to _DEFAULT_DS for _DEFAULT tenant in privileges table")
            self.conn.commit()
            return None
        except sqlite3.Error as e:
            error_msg = "Error when upgrading auth DB table({})".format(str(e))
            logging.error("handle_upgrade_1_1_to_1_2. %s", error_msg)
            raise DbUpgradeError(self.db_path, error_msg)

    def __handle_upgrade(self):
        error_msg, major_ver, minor_ver = self.__get_db_version()
        if error_msg:
            error_msg = "Failed to get auth-db schema version, cannot upgrade"
            logging.error("__handle_upgrade: %s", error_msg)
            raise DbUpgradeError(self.db_path, error_msg)

        # if db schema is latest, no need for upgrade
        if major_ver == DB_MAJOR_VER and minor_ver == DB_MINOR_VER:
            return

        if DB_MAJOR_VER == 1 and DB_MINOR_VER == 2:
            if major_ver == 1 and minor_ver == 1:
                self.handle_upgrade_1_1_to_1_2()
            else:
                error_msg = "Upgrade is not supported for auth-db schema version {}.{} to {}.{}. Refer to vDVS release versions".format(major_ver, minor_ver, DB_MAJOR_VER, DB_MINOR_VER)
                logging.error("__handle_upgrade: %s", error_msg)
                raise DbUpgradeError(self.db_path, error_msg)

    def __connect(self):
        """
        Private function for connecting and setting return type for select ops.
        Raises a DbConnectionError when fails to connect.
        """
        if self.conn:
            logging.info("AuthorizationDataManager: Reconnecting to %s on request", self.db_path)
            self.__close()

        try:
            self.conn = sqlite3.connect(self.db_path)
        except sqlite3.Error as e:
            logging.error("Failed to connect to DB (%s): %s", self.db_path, e)
            raise DbConnectionError(self.db_path)

        # Use return rows as Row instances instead of tuples
        self.conn.row_factory = sqlite3.Row

    def connect(self):
        """
        Connect to a sqlite database at `db_path`. Validates mode, checks for upgrades.
        If the DB does not exist, simply exists leving self.__mode as NotConfigured
        """

        self.__mode = self.__discover_mode_and_connect()

        if self.__mode == DBMode.BrokenLink:
            raise DbAccessError(self.db_path, DBMode(self.__mode))

        if self.__mode == DBMode.NotConfigured:
            logging.info("Auth DB %s is missing, allowing all access", self.db_path)
            return

        self.__handle_upgrade()

    @property
    def mode(self):
        """Getter for current mode. Note that it can only be tst by calling self.connect()"""
        return DBMode(self.__mode)
    @mode.setter
    def mode(self, mode):
        self.__mode = mode

    def __discover_mode_and_connect(self):
        """
        Checks correctness of symlink for <path>, location and content for the DB
        and returns the auth_data.DBMode
        Also does remediation like removing empty DB (for NotConfigured state)
        """


        logging.debug("Checking DB mode for %s...", self.db_path)
        # check if the path exists (without following symlinks)
        if not os.path.lexists(self.db_path):
            logging.debug(DB_REF + "does not exist. mode NotConfigured")
            return DBMode.NotConfigured

        if not os.path.exists(self.db_path):
            logging.error(DB_REF + "broken link: {}".format(self.db_path))
            return DBMode.BrokenLink

        # OK, DB exists (link or actual file) , so time to connect
        self.__connect()

        # it's a single node config file. Check if the DB is empty so we can drop it.
        if not os.path.islink(self.db_path):
            logging.debug(DB_REF + "exists and has modifications, mode SingleNode")
            return DBMode.SingleNode

        # let's check if the db content seems good
        (err, major, minor) = self.__get_db_version()
        if err:
            logging.error(DB_REF + "Location {}, error: {}".format(self.db_path, err))
            return DBMode.BrokenLink

        logging.debug(DB_REF + "found. maj_ver={} min_ver={} mode MultiNode".format(major, minor))
        return DBMode.MultiNode

    def allow_all_access(self):
        """Allow all access if we are in NotConfigured mode"""
        return self.__mode == DBMode.NotConfigured

    def is_connected(self):
        """helper for outside world"""
        return self.conn != None

    def get_info(self):
        """Returns a dict with useful info for misc. status commands"""

        link_location = db_location = "N/A"
        if self.mode == DBMode.MultiNode:
            db_location = os.readlink(self.db_path)
            link_location = self.db_path
        elif self.mode == DBMode.SingleNode:
            link_location = self.db_path
        elif self.mode == DBMode.BrokenLink:
            link_location = self.db_path

        return {"DB_Mode": str(self.mode),
                "DB_LocalPath": link_location,
                "DB_SharedLocation": db_location}

    def err_config_init_needed(self):
        """Return standard error msg for NotConfigured mode"""
        return error_code_to_message[ErrorCode.INIT_NEEDED]


    def new_db(self):
        """
        Create brand new DB content at self.db_path. Expects clean slate.
        :returns: None for success , a string (with error) for error
        """
        if not self.conn:
            self.__connect()
        err = self.__create_tables()
        if err:
            return err
        err = self.__create_default_tenant()
        if err:
            return err
        err = self.__create_all_ds_privileges_for_default_tenant()
        if err:
            return err
        err = self.__create_vm_ds_privileges_for_default_tenant()
        if err:
            return err
        return None

    def __create_tables(self):
        """ Create tables used for per-datastore authorization.

        This function should only be called once per datastore.
        It will raise an exception if the schema file isn't
        accessible or the tables already exist.
        """
        try:
            self.conn.execute('PRAGMA foreign_key = ON')
            self.conn.execute('''
                CREATE TABLE tenants(
                    -- uuid for the tenant. Generated by create_tenant() API
                    id TEXT PRIMARY KEY NOT NULL,
                    -- name of the tenant. Specified by user when creating the tenant
                    -- this field can be changed later by using set_name() API
                    name TEXT UNIQUE NOT NULL,
                    -- brief description of the tenant. Specified by user when creating the tenant
                    -- this field can be changed laster by using set_description API
                    description TEXT,
                    -- default_datastore url
                    default_datastore_url TEXT
                    )''')

            self.conn.execute('''
            CREATE TABLE vms(
                -- uuid for the VM, which is generated when VM is created
                -- this uuid will be passed in to executeRequest()
                -- this field need to be specified when adding a VM to a tenant
                vm_id TEXT PRIMARY KEY NOT NULL,
                -- id in tenants table
                tenant_id TEXT NOT NULL,
                -- name of the VM being added to the tenant
                vm_name TEXT,
                FOREIGN KEY(tenant_id) REFERENCES tenants(id)
                ); ''')

            self.conn.execute('''
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
                );''')

            self.conn.execute('''
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
                );''')

            self.conn.execute('''
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
                );''')

            # insert latest DB version and VMODL version to table "versions"
            self.conn.execute("INSERT INTO versions(id, major_ver, minor_ver, vmodl_major_ver, vmodl_minor_ver) " +
                              "VALUES (?, ?, ?, ?, ?)",
                              (0, DB_MAJOR_VER, DB_MINOR_VER, VMODL_MAJOR_VER, VMODL_MINOR_VER))
        except sqlite3.Error as e:
            logging.error("Error '%s` when creating auth DB tables", e)
            return str(e)

        return None

    def create_tenant(self, name, description, vms, privileges):
        """ Create a tenant in the database.
        If tenant_uuid is None, tenant id will be auto-generated and returned,
        otherwise, the uuid specified by param "tenant_uuid" will be used
        vms are list of (vm_id, vm_name). Privileges are dictionaries
        with keys matching the row names in the privileges table. Tenant id is
        filled in for both the vm and privileges tables.
        """

        logging.debug("create_tenant name=%s", name)
        if self.allow_all_access():
            return self.err_config_init_needed(), None

        if privileges:
            for p in privileges:
                if not all_columns_set(p):
                    error_msg = "Not all columns are set in privileges"
                    return error_msg, None

        if name == auth_data_const.DEFAULT_TENANT:
            tenant_uuid = auth_data_const.DEFAULT_TENANT_UUID
        else:
            tenant_uuid = None
        # Create the entry in the tenants table
        default_datastore_url = ""
        tenant = DockerVolumeTenant(name=name,
                                    description=description,
                                    vms=vms,
                                    privileges=privileges,
                                    default_datastore_url=default_datastore_url,
                                    id=tenant_uuid)

        id = tenant.id
        try:
            self.conn.execute(
                "INSERT INTO tenants(id, name, description, default_datastore_url) VALUES (?, ?, ?, ?)",
                (id, name, description, default_datastore_url)
            )

            if vms:
                # Create the entries in the vms table
                vms = [(vm_id, vm_name, id) for vm_id, vm_name in vms]

                self.conn.executemany(
                    "INSERT INTO vms(vm_id, vm_name, tenant_id) VALUES (?, ?, ?)",
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

    def __create_default_tenant(self):
        """ Create DEFAULT tenant in DB"""
        error_msg, tenant = self.create_tenant(
            name=auth_data_const.DEFAULT_TENANT,
            description=auth_data_const.DEFAULT_TENANT_DESCR,
            vms=[],
            privileges=[])
        if error_msg:
            err = error_code_to_message[ErrorCode.TENANT_CREATE_FAILED].format(auth_data_const.DEFAULT_TENANT, error_msg)
            logging.warning(err)
            return err
        error_msg = tenant.set_default_datastore(self.conn, auth_data_const.VM_DS_URL)
        if error_msg:
            err = error_code_to_message[ErrorCode.DS_DEFAULT_SET_FAILED].format(auth_data_const.DEFAULT_TENANT,
                                                                                error_msg)
            logging.warning(err)
            return err
        return None

    def get_all_ds_privileges_dict(self):
        """Form a dictionary with all_ds privileges used in creating of default tenant"""
        return  [{'datastore_url': auth_data_const.ALL_DS_URL,
                  'allow_create': 1,
                  'max_volume_size': 0,
                  'usage_quota': 0}]

    def get_vm_ds_privileges_dict(self):
        """Form a dictionary with VM_DS privileges used in creating of default tenant"""
        return  [{'datastore_url': auth_data_const.VM_DS_URL,
                  'allow_create': 1,
                  'max_volume_size': 0,
                  'usage_quota': 0}]

    def get_default_privileges_dict(self):
        """Form a dictionary with default privileges used in cresting of default tenant"""
        # _DEFAULT tenant is created with two privileges
        return  [{'datastore_url': auth_data_const.ALL_DS_URL,
                  'allow_create': 1,
                  'max_volume_size': 0,
                  'usage_quota': 0},
                  {'datastore_url': auth_data_const.VM_DS_URL,
                  'allow_create': 1,
                  'max_volume_size': 0,
                  'usage_quota': 0}]

    def __create_all_ds_privileges_for_default_tenant(self):
            """
            create _ALL_DS privilege for _DEFAULT tenant
            For given tenant, _ALL_DS privilege will match any datastore which does not have an entry
            in privileges table explicitly
            this privilege will have full permission (create, delete, and mount)
            and no max_volume_size and usage_quota limitation
            """
            logging.debug("__create_all_ds_privileges_for_default_tenant")
            privileges = self.get_all_ds_privileges_dict()

            error_msg, tenant = self.get_tenant(auth_data_const.DEFAULT_TENANT)
            if error_msg:
                err = error_code_to_message[ErrorCode.TENANT_NOT_EXIST].format(auth_data_const.DEFAULT_TENANT)
                logging.warning(err)
                return err

            error_msg = tenant.set_datastore_access_privileges(self.conn, privileges)
            if error_msg:
                err = error_code_to_message[ErrorCode.TENANT_SET_ACCESS_PRIVILEGES_FAILED].format(auth_data_const.DEFAULT_TENANT,
                                                                                                  auth_data_const.ALL_DS, error_msg)
                logging.warning(err)
                return err
            return None

    def __create_vm_ds_privileges_for_default_tenant(self):
        """
        create _VM_DS privilege for _DEFAULT tenant
        this privilege will have full permission (create, delete, and mount)
        and no max_volume_size and usage_quota limitation
        """
        logging.debug("__create_vm_ds_privileges_for_default_tenant")
        privileges = self.get_vm_ds_privileges_dict()
        error_msg, tenant = self.get_tenant(auth_data_const.DEFAULT_TENANT)
        if error_msg:
            err = error_code_to_message[ErrorCode.TENANT_NOT_EXIST].format(auth_data_const.DEFAULT_TENANT)
            logging.warning(err)
            return err

        error_msg = tenant.set_datastore_access_privileges(self.conn, privileges)
        if error_msg:
            err = error_code_to_message[ErrorCode.TENANT_SET_ACCESS_PRIVILEGES_FAILED].format(auth_data_const.DEFAULT_TENANT,
                                                                                              auth_data_const.VM_DS, error_msg)
            logging.warning(err)
            return err
        return None



    def get_tenant(self, tenant_name):
        """
        Return an (err, obj) where err is None or error code,
        and obj is an object which match the given tenant_name or None
        """
        logging.debug("auth_data.get_tenant: tenant_name=%s", tenant_name)

        if self.allow_all_access():
            if tenant_name == auth_data_const.DEFAULT_TENANT:
                return None, DockerVolumeTenant(name=tenant_name,
                                                description=auth_data_const.DEFAULT_TENANT_DESCR,
                                                vms=[],
                                                privileges=self.get_default_privileges_dict(),
                                                id=auth_data_const.DEFAULT_TENANT_UUID,
                                                default_datastore_url=auth_data_const.VM_DS_URL)
            else:
                return ErrorCode.INIT_NEEDED, None

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
            logging.error("Error %s in get_tenant(%s)", e, tenant_name)
            return ErrorCode.SQLITE3_ERROR, tenant

        return None, tenant

    def list_tenants(self):
        """ Return a list of DockerVolumeTenants objects. """
        if self.allow_all_access():
            _, tenant = self.get_tenant(auth_data_const.DEFAULT_TENANT)
            return None, [tenant]

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
        if self.allow_all_access():
            logging.info("[AllowAllAccess] skipping volume record removal for %s", tenant_id)
            return None

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


    def __remove_volumes_for_tenant(self, tenant_id):
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
                datastore_url = vmdk_utils.get_datastore_url(vmdk['datastore'])
                err = vmdk_ops.removeVMDK(vmdk_path=vmdk_path,
                                          vol_name=vmdk_utils.strip_vmdk_extension(vmdk['filename']),
                                          vm_name=None,
                                          tenant_uuid=tenant_id,
                                          datastore_url=datastore_url)
                if err:
                    logging.error("remove vmdk %s failed with error %s", vmdk_path, err)
                    error_msg += str(err)

            VOL_RM_LOG_PREFIX = "Tenant <name> %s removal: "
            # delete the symlink /vmfs/volume/datastore_name/tenant_name
            # which point to /vmfs/volumes/datastore_name/tenant_uuid
            for (datastore, _, path) in vmdk_utils.get_datastores():
                dockvol_path, tenant_path = get_dockvol_path_tenant_path(datastore_name=datastore,
                                                                         tenant_id=tenant_id)
                logging.debug(VOL_RM_LOG_PREFIX + "try to remove symlink to %s", tenant_name, tenant_path)

                if os.path.isdir(tenant_path):
                    exist_symlink_path = os.path.join(dockvol_path, tenant_name)
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
        """
        Remove a tenant with given id.
        A row with given tenant_id will be removed from table tenants, vms,
        and privileges.
        If remove_volumes is True -  all volumes for this tenant will be removed as well.
        Returns None for success, error string for errors.
        """
        logging.debug("remove_tenant: tenant_id%s, remove_volumes=%d", tenant_id, remove_volumes)

        if self.allow_all_access():
            return self.err_config_init_needed()

        if remove_volumes:
            error_msg = self.__remove_volumes_for_tenant(tenant_id)
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
        """ Return tenant_name which matches the given tenant_uuid """
        logging.debug("get_tenant_name: tenant_uuid=%s", tenant_uuid)

        if self.allow_all_access():
            if tenant_uuid == auth_data_const.DEFAULT_TENANT_UUID:
                return None, auth_data_const.DEFAULT_TENANT
            else:
                return self.err_config_init_needed(), None

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
            error_msg =  error_code_to_message[ErrorCode.TENANT_NAME_NOT_FOUND].format(tenant_uuid)
            logging.debug("get_tenant_name:"+error_msg)
            return error_msg, None

def main():
    log_config.configure()

if __name__ == "__main__":
    main()
