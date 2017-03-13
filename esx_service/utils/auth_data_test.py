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

# Tests for auth.py

import unittest
import os
import os.path
import auth_data
import uuid
import auth_data_const
import auth
import log_config

class TestAuthDataModel(unittest.TestCase):
    """
    Test the Authorization data model via the AuthorizationDataManager

    """
    db_path = "/tmp/test-auth.db"

    def setUp(self):
        """ Create the auth DB and connect to the DB for each test """

        # TBD: we should not unlink the DB on start ever.
        # instead, we should make sure the DB is clean on TEST exit
        try:
            os.unlink(self.db_path)
        except:
            pass

        self.auth_mgr = auth_data.AuthorizationDataManager(self.db_path)
        self.auth_mgr.connect()
        if self.auth_mgr.allow_all_access():
            raise unittest.SkipTest("Allow All Access mode: no need to test authorization")

    def tearDown(self):
        """ Tear down the auth DB after each test """

        try:
            os.unlink(self.db_path)
        except:
            pass


    def get_privileges(self):
        privileges = [{'datastore_url': 'datastore1_url',
                       'allow_create': 0,
                       'max_volume_size': 0,
                       'usage_quota': 0}]
        return privileges

    def get_default_datastore(self):
        default_datastore = 'default_ds'
        return default_datastore

    def get_datastore_url(self, datastore):
        datastore_url = datastore+"_url"
        return datastore_url

    def test_create_tenant(self):
        """ Test create_tenant() API """

        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid)]
        privileges = self.get_privileges()
        default_datastore = self.get_default_datastore()
        default_datastore_url = self.get_datastore_url(default_datastore)
        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        # Check tenants table
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = [tenant1.id,
                           'tenant1',
                           'Some tenant']

        actual_output = [tenants_row[auth_data_const.COL_ID],
                         tenants_row[auth_data_const.COL_NAME],
                         tenants_row[auth_data_const.COL_DESCRIPTION]
                        ]

        self.assertEqual(actual_output, expected_output)

        # check vms table
        error_info, vms_row = auth.get_row_from_vms_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = [vm1_uuid,
                           tenant1.id]
        self.assertEqual(len(vms_row), 1)

        actual_output = [vms_row[0][auth_data_const.COL_VM_ID],
                         vms_row[0][auth_data_const.COL_TENANT_ID]
                        ]
        self.assertEqual(actual_output, expected_output)

        # check privileges table
        error_info, privileges_row = auth.get_row_from_privileges_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        self.assertEqual(len(privileges_row), 1)

        expected_privileges = [tenant1.id,
                               privileges[0][auth_data_const.COL_DATASTORE_URL],
                               privileges[0][auth_data_const.COL_ALLOW_CREATE],
                               privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                               privileges[0][auth_data_const.COL_USAGE_QUOTA]
                              ]

        expected_output = [expected_privileges
                          ]

        actual_privileges = [privileges_row[0][auth_data_const.COL_TENANT_ID],
                             privileges_row[0][auth_data_const.COL_DATASTORE_URL],
                             privileges_row[0][auth_data_const.COL_ALLOW_CREATE],
                             privileges_row[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                             privileges_row[0][auth_data_const.COL_USAGE_QUOTA]
                             ]

        actual_output = [actual_privileges]
        self.assertEqual(actual_output, expected_output)

    def test_add_vms(self):
        """ Test add_vms() API """

        vms = []
        privileges = []
        default_datastore = self.get_default_datastore()
        default_datastore_url = self.get_datastore_url(default_datastore)
        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        vm1_uuid = str(uuid.uuid4())
        vm2_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid), (vm2_uuid)]
        error_info = tenant1.add_vms(self.auth_mgr.conn, vms)
        self.assertEqual(error_info, None)

         # check vms table
        error_info, vms_row = auth.get_row_from_vms_table(self.auth_mgr.conn,tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = [(vm1_uuid, tenant1.id),
                           (vm2_uuid, tenant1.id)
                          ]
        self.assertEqual(len(vms_row), 2)

        actual_output = [(vms_row[0][auth_data_const.COL_VM_ID],
                          vms_row[0][auth_data_const.COL_TENANT_ID]),
                         (vms_row[1][auth_data_const.COL_VM_ID],
                          vms_row[1][auth_data_const.COL_TENANT_ID]),
                        ]
        self.assertEqual(actual_output, expected_output)

    def test_remove_vms(self):
        """
        """

        privileges = self.get_privileges()
        default_datastore = self.get_default_datastore()
        default_datastore_url = self.get_datastore_url(default_datastore)

        vm1_uuid = str(uuid.uuid4())
        vm2_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid), (vm2_uuid)]
        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        error_info = tenant1.remove_vms(self.auth_mgr.conn, vms)
        self.assertEqual(error_info, None)
        error_info, vms_row = auth.get_row_from_vms_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        self.assertEqual(vms_row, [])


    def test_set_name(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid)]

        privileges = self.get_privileges()
        default_datastore = self.get_default_datastore()
        default_datastore_url = self.get_datastore_url(default_datastore)
        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        error_info = tenant1.set_name(self.auth_mgr.conn, 'tenant1', 'new_tenant1')
        self.assertEqual(error_info, None)
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = 'new_tenant1'
        actual_output = tenants_row[auth_data_const.COL_NAME]
        self.assertEqual(actual_output, expected_output)


    def test_set_description(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid)]
        privileges = self.get_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        error_info = tenant1.set_description(self.auth_mgr.conn, 'new description')
        self.assertEqual(error_info, None)
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = 'new description'
        actual_output = tenants_row[auth_data_const.COL_DESCRIPTION]
        self.assertEqual(actual_output, expected_output)

    def test_set_default_datastore(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid)]
        privileges = self.get_privileges()
        default_datastore = self.get_default_datastore()
        default_datastore_url = self.get_datastore_url(default_datastore)
        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        default_datastore = 'new_default_ds'
        default_datastore_url = self.get_datastore_url(default_datastore)
        error_info = tenant1.set_default_datastore(self.auth_mgr.conn, default_datastore_url)
        self.assertEqual(error_info, None)
        # Check tenants table
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = 'new_default_ds_url'
        actual_output = tenants_row[auth_data_const.COL_DEFAULT_DATASTORE_URL]
        self.assertEqual(actual_output, expected_output)

    def test_add_datastore_access_privileges(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid)]
        privileges = []

        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        datastore_url = self.get_datastore_url('datastore1')
        privileges = [{'datastore_url': datastore_url,
                       'allow_create': 0,
                       'max_volume_size': 0,
                       'usage_quota': 0}]

        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)

        #check privileges table
        error_info, privileges_row = auth.get_row_from_privileges_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        self.assertEqual(len(privileges_row), 1)
        expected_privileges = [tenant1.id,
                               privileges[0][auth_data_const.COL_DATASTORE_URL],
                               privileges[0][auth_data_const.COL_ALLOW_CREATE],
                               privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                               privileges[0][auth_data_const.COL_USAGE_QUOTA]
                              ]

        actual_privileges = [privileges_row[0][auth_data_const.COL_TENANT_ID],
                             privileges_row[0][auth_data_const.COL_DATASTORE_URL],
                             privileges_row[0][auth_data_const.COL_ALLOW_CREATE],
                             privileges_row[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                             privileges_row[0][auth_data_const.COL_USAGE_QUOTA]
                             ]
        self.assertEqual(actual_privileges, expected_privileges)

    def get_tenant_idx(self, tenants_list, tenant_uuid):
        idx = -1
        for i in range(0, len(tenants_list)):
            if tenants_list[i].id == tenant_uuid:
                return i

        return idx

    def test_list_tenants(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid)]
        privileges = []
        default_datastore = self.get_default_datastore()
        default_datastore_url = self.get_datastore_url(default_datastore)
        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        vm2_uuid = str(uuid.uuid4())
        vm3_uuid = str(uuid.uuid4())
        vms = [(vm2_uuid), (vm3_uuid)]
        privileges = []
        error_info, tenant2 = self.auth_mgr.create_tenant(name='tenant2',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant2.id))

        datastore_url = self.get_datastore_url('datastore1')
        privileges = [{'datastore_url': datastore_url,
                       'allow_create': 0,
                       'max_volume_size': 0,
                       'usage_quota': 0}]

        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)
        error_info, tenants_list = self.auth_mgr.list_tenants()
        self.assertEqual(error_info, None)

        #Check tenants.id tenant.name, tenant.description and tenant.default_datastore
        tenant1_idx = self.get_tenant_idx(tenants_list, tenant1.id)
        tenant2_idx = self.get_tenant_idx(tenants_list, tenant2.id)
        self.assertNotEqual(tenant1_idx, -1)
        self.assertNotEqual(tenant2_idx, -1)
        # check for tenant1
        tenant1_expected_output = [
                                   tenant1.id,
                                   'tenant1',
                                   'Some tenant',
                                   '',
                                  ]
        tenant1_actual_output = [
                                 tenants_list[tenant1_idx].id,
                                 tenants_list[tenant1_idx].name,
                                 tenants_list[tenant1_idx].description,
                                 tenants_list[tenant1_idx].default_datastore_url,
                                ]
        self.assertEqual(tenant1_actual_output, tenant1_expected_output)

        # check vms
        tenant1_expected_output = [(vm1_uuid),
                                  ]
        tenant1_actual_output = [(tenants_list[tenant1_idx].vms[0])
                                ]

        self.assertEqual(tenant1_actual_output, tenant1_expected_output)

        # check privileges
        tenant1_expected_output = []
        tenant1_actual_output = tenants_list[tenant1_idx]
        tenant1_expected_output = [tenant1.id,
                                   privileges[0][auth_data_const.COL_DATASTORE_URL],
                                   privileges[0][auth_data_const.COL_ALLOW_CREATE],
                                   privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                                   privileges[0][auth_data_const.COL_USAGE_QUOTA]
                                  ]

        tenant1_actual_output = [tenants_list[tenant1_idx].privileges[0].tenant_id,
                                 tenants_list[tenant1_idx].privileges[0].datastore_url,
                                 tenants_list[tenant1_idx].privileges[0].allow_create,
                                 tenants_list[tenant1_idx].privileges[0].max_volume_size,
                                 tenants_list[tenant1_idx].privileges[0].usage_quota
                                ]

        self.assertEqual(tenant1_actual_output, tenant1_expected_output)

        # check for tenant2
        tenant2_expected_output = [
                                   tenant2.id,
                                   'tenant2',
                                   'Some tenant',
                                   '',
                                  ]
        tenant2_actual_output = [
                                 tenants_list[tenant2_idx].id,
                                 tenants_list[tenant2_idx].name,
                                 tenants_list[tenant2_idx].description,
                                 tenants_list[tenant2_idx].default_datastore_url,
                                ]
        self.assertEqual(tenant2_actual_output, tenant2_expected_output)

        # check vms
        self.assertEqual(len(tenants_list[tenant2_idx].vms), 2)
        tenant2_expected_output = [(vm2_uuid),
                                   (vm3_uuid)
                                  ]
        tenant2_actual_output = [(tenants_list[tenant2_idx].vms[0]),
                                 (tenants_list[tenant2_idx].vms[1])
                                ]
        self.assertEqual(tenant2_actual_output, tenant2_expected_output)

        # check privileges
        tenant2_expected_output = []
        tenant2_actual_output = tenants_list[tenant2_idx].privileges

        self.assertEqual(tenant2_actual_output, tenant2_expected_output)


    def test_remove_tenants(self):
        vms = [(str(uuid.uuid4()))]

        privileges = self.get_privileges()
        default_datastore = self.get_default_datastore()
        default_datastore_url = self.get_datastore_url(default_datastore)

        error_info, tenant1 = self.auth_mgr.create_tenant(name='tenant1',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        vms = [(str(uuid.uuid4())), (str(uuid.uuid4()))]
        privileges = []
        error_info, tenant2 = self.auth_mgr.create_tenant(name='tenant2',
                                                          description='Some tenant',
                                                          vms=vms,
                                                          privileges=privileges)

        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant2.id))

        tenant2.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        error_info = self.auth_mgr.remove_tenant(tenant2.id, False)
        self.assertEqual(error_info, None)

        # Check tenants table
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant2.id)
        self.assertEqual(error_info, None)
        self.assertEqual(tenants_row, None)

        # check vms table
        error_info, vms_row = auth.get_row_from_vms_table(self.auth_mgr.conn, tenant2.id)
        self.assertEqual(error_info, None)
        self.assertEqual(vms_row, [])

        # check privileges table
        error_info, privileges_row = auth.get_row_from_privileges_table(self.auth_mgr.conn, tenant2.id)
        self.assertEqual(error_info, None)
        self.assertEqual(privileges_row, [])

if __name__ == "__main__":
    log_config.configure()
    unittest.main()
