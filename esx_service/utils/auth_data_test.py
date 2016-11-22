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

class TestAuthDataModel(unittest.TestCase):
    """
    Test the Authorization data model via the AuthorizationDataManager

    """
    db_path = "/tmp/test-auth.db"
    
    def setUp(self):
        """ Create the auth DB and connect to the DB for each test """

        try:
            os.unlink(self.db_path)
        except:
            pass
            
        self.auth_mgr = auth_data.AuthorizationDataManager(self.db_path)
        self.auth_mgr.connect()
       
    def tearDown(self):
        """ Tear down the auth DB after each test """

        os.unlink(self.db_path)

    def get_privileges(self):
        privileges = [{'datastore': 'datastore1',
                       'create_volume': 0,
                       'delete_volume': 0,
                       'mount_volume': 0,
                       'max_volume_size': 0,
                       'usage_quota': 0}]
        return privileges
    
    def get_default_datastore_and_privileges(self):
        default_datastore = 'default_ds'
        default_privileges = {'datastore': default_datastore,
                              'create_volume': 0,
                              'delete_volume': 0,
                              'mount_volume': 0,
                              'max_volume_size': 0,
                              'usage_quota': 0}
        return default_datastore, default_privileges
                
    def test_create_tenant(self):
        """ Test create_tenant() API """

        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1')]
        
        #privileges = []
        privileges = self.get_privileges()
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)                                            
        self.assertTrue(uuid.UUID(tenant1.id))

        # Check tenants table
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = [tenant1.id,
                           'tenant1',
                           'Some tenant',
                           'default_ds']
        actual_output = [tenants_row[auth_data_const.COL_ID],
                         tenants_row[auth_data_const.COL_NAME],
                         tenants_row[auth_data_const.COL_DESCRIPTION],
                         tenants_row[auth_data_const.COL_DEFAULT_DATASTORE]
                        ]
                          
        self.assertEqual(actual_output, expected_output)

        # check vms table 
        error_info, vms_row = auth.get_row_from_vms_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = [vm1_uuid,
                           'vm1',
                           tenant1.id]
        self.assertEqual(len(vms_row), 1)

        actual_output = [vms_row[0][auth_data_const.COL_VM_ID],
                         vms_row[0][auth_data_const.COL_VM_NAME],
                         vms_row[0][auth_data_const.COL_TENANT_ID]
                        ]
        self.assertEqual(actual_output, expected_output)

        # check privileges table
        error_info, privileges_row = auth.get_row_from_privileges_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        self.assertEqual(len(privileges_row), 2)

        expected_privileges = [tenant1.id,
                               privileges[0][auth_data_const.COL_DATASTORE],
                               privileges[0][auth_data_const.COL_CREATE_VOLUME],
                               privileges[0][auth_data_const.COL_DELETE_VOLUME],
                               privileges[0][auth_data_const.COL_MOUNT_VOLUME],
                               privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                               privileges[0][auth_data_const.COL_USAGE_QUOTA]
                              ]
        expected_default_privileges = [tenant1.id,
                                       default_privileges[auth_data_const.COL_DATASTORE],
                                       default_privileges[auth_data_const.COL_CREATE_VOLUME],
                                       default_privileges[auth_data_const.COL_DELETE_VOLUME],
                                       default_privileges[auth_data_const.COL_MOUNT_VOLUME],
                                       default_privileges[auth_data_const.COL_MAX_VOLUME_SIZE],
                                       default_privileges[auth_data_const.COL_USAGE_QUOTA]
                                      ]

        expected_output = [expected_privileges, 
                           expected_default_privileges
                          ]
         
        actual_privileges = [privileges_row[0][auth_data_const.COL_TENANT_ID],
                             privileges_row[0][auth_data_const.COL_DATASTORE],
                             privileges_row[0][auth_data_const.COL_CREATE_VOLUME],
                             privileges_row[0][auth_data_const.COL_DELETE_VOLUME],
                             privileges_row[0][auth_data_const.COL_MOUNT_VOLUME],
                             privileges_row[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                             privileges_row[0][auth_data_const.COL_USAGE_QUOTA]
                             ]
        actual_default_privileges = [privileges_row[1][auth_data_const.COL_TENANT_ID],
                                     privileges_row[1][auth_data_const.COL_DATASTORE],
                                     privileges_row[1][auth_data_const.COL_CREATE_VOLUME],
                                     privileges_row[1][auth_data_const.COL_DELETE_VOLUME],
                                     privileges_row[1][auth_data_const.COL_MOUNT_VOLUME],
                                     privileges_row[1][auth_data_const.COL_MAX_VOLUME_SIZE],
                                     privileges_row[1][auth_data_const.COL_USAGE_QUOTA]
                                    ]
        actual_output = [actual_privileges, actual_default_privileges]
        self.assertEqual(actual_output, expected_output)        
            
    def test_add_vms(self): 
        """ Test add_vms() API """

        vms = []
        privileges = []
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        
        vm1_uuid = str(uuid.uuid4())
        vm2_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1'), (vm2_uuid, 'vm2')]
        error_info = tenant1.add_vms(self.auth_mgr.conn, vms)
        self.assertEqual(error_info, None)

         # check vms table 
        error_info, vms_row = auth.get_row_from_vms_table(self.auth_mgr.conn,tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = [(vm1_uuid, 'vm1', tenant1.id),
                           (vm2_uuid, 'vm2', tenant1.id) 
                          ]
        self.assertEqual(len(vms_row), 2)

        actual_output = [(vms_row[0][auth_data_const.COL_VM_ID],
                          vms_row[0][auth_data_const.COL_VM_NAME],
                          vms_row[0][auth_data_const.COL_TENANT_ID]),
                         (vms_row[1][auth_data_const.COL_VM_ID],
                          vms_row[1][auth_data_const.COL_VM_NAME],
                          vms_row[1][auth_data_const.COL_TENANT_ID]),
                        ]
        self.assertEqual(actual_output, expected_output)
    
    def test_remove_vms(self):
        """
        """
            
        privileges = self.get_privileges()
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        
        vm1_uuid = str(uuid.uuid4())
        vm2_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1'), (vm2_uuid, 'vm2')]
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        error_info = tenant1.remove_vms(self.auth_mgr.conn, vms)
        self.assertEqual(error_info, None)
        error_info, vms_row = auth.get_row_from_vms_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        self.assertEqual(vms_row, [])
        
    
    def test_set_name(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1')]
        
        privileges = self.get_privileges()
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        error_info = tenant1.set_name(self.auth_mgr.conn, 'new_tenant1')
        self.assertEqual(error_info, None)
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = 'new_tenant1'
        actual_output = tenants_row[auth_data_const.COL_NAME]
        self.assertEqual(actual_output, expected_output)

    
    def test_set_description(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1')]
        privileges = self.get_privileges()
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        error_info = tenant1.set_description(self.auth_mgr.conn, 'new description')
        self.assertEqual(error_info, None)
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = 'new description'
        actual_output = tenants_row[auth_data_const.COL_DESCRIPTION]
        self.assertEqual(actual_output, expected_output)
    
    def test_set_default_datastore_and_privileges(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1')]
        privileges = self.get_privileges()
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))

        default_datastore = 'new_default_ds'
        default_privileges = {'datastore': default_datastore,
                              'create_volume': 1,
                              'delete_volume': 1,
                              'mount_volume': 1,
                              'max_volume_size': 0,
                              'usage_quota': 0}
        error_info = tenant1.set_default_datastore_and_privileges(self.auth_mgr.conn, default_datastore, default_privileges)
        self.assertEqual(error_info, None)
        # Check tenants table
        error_info, tenants_row = auth.get_row_from_tenants_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        expected_output = 'new_default_ds'
        actual_output = tenants_row[auth_data_const.COL_DEFAULT_DATASTORE]
        self.assertEqual(actual_output, expected_output)

        #check privileges table
        error_info, privileges_row = auth.get_row_from_privileges_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        self.assertEqual(len(privileges_row), 2)
        expected_default_privileges = [tenant1.id,
                                       default_privileges[auth_data_const.COL_DATASTORE],
                                       default_privileges[auth_data_const.COL_CREATE_VOLUME],
                                       default_privileges[auth_data_const.COL_DELETE_VOLUME],
                                       default_privileges[auth_data_const.COL_MOUNT_VOLUME],
                                       default_privileges[auth_data_const.COL_MAX_VOLUME_SIZE],
                                       default_privileges[auth_data_const.COL_USAGE_QUOTA]
                                      ]
                 
        actual_default_privileges = [privileges_row[1][auth_data_const.COL_TENANT_ID],
                                     privileges_row[1][auth_data_const.COL_DATASTORE],
                                     privileges_row[1][auth_data_const.COL_CREATE_VOLUME],
                                     privileges_row[1][auth_data_const.COL_DELETE_VOLUME],
                                     privileges_row[1][auth_data_const.COL_MOUNT_VOLUME],
                                     privileges_row[1][auth_data_const.COL_MAX_VOLUME_SIZE],
                                     privileges_row[1][auth_data_const.COL_USAGE_QUOTA]
                                    ]
        self.assertEqual(actual_default_privileges, expected_default_privileges)                
                                                      
    
    def test_add_datastore_access_privileges(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1')]
        privileges = []
        
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
               
        privileges = [{'datastore': 'datastore1',
                       'create_volume': 1,
                       'delete_volume': 1,
                       'mount_volume': 0,
                       'max_volume_size': 0,
                       'usage_quota': 0}]
        
        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)

        #check privileges table
        error_info, privileges_row = auth.get_row_from_privileges_table(self.auth_mgr.conn, tenant1.id)
        self.assertEqual(error_info, None)
        self.assertEqual(len(privileges_row), 2)
        expected_privileges = [tenant1.id,
                               privileges[0][auth_data_const.COL_DATASTORE],
                               privileges[0][auth_data_const.COL_CREATE_VOLUME],
                               privileges[0][auth_data_const.COL_DELETE_VOLUME],
                               privileges[0][auth_data_const.COL_MOUNT_VOLUME],
                               privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                               privileges[0][auth_data_const.COL_USAGE_QUOTA]
                              ]
                 
        actual_privileges = [privileges_row[0][auth_data_const.COL_TENANT_ID],
                             privileges_row[0][auth_data_const.COL_DATASTORE],
                             privileges_row[0][auth_data_const.COL_CREATE_VOLUME],
                             privileges_row[0][auth_data_const.COL_DELETE_VOLUME],
                             privileges_row[0][auth_data_const.COL_MOUNT_VOLUME],
                             privileges_row[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                             privileges_row[0][auth_data_const.COL_USAGE_QUOTA]
                             ]
        self.assertEqual(actual_privileges, expected_privileges)
    

    def test_list_tenants(self):
        vm1_uuid = str(uuid.uuid4())
        vms = [(vm1_uuid, 'vm1')]
        privileges = []
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        
        vm2_uuid = str(uuid.uuid4())
        vm3_uuid = str(uuid.uuid4())
        vms = [(vm2_uuid, 'vm2'), (vm3_uuid, 'vm3')]
        privileges = []
        error_info, tenant2 = self.auth_mgr.create_tenant('tenant2', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant2.id))
        
        privileges = [{'datastore': 'datastore1',
                       'create_volume': 0,
                       'delete_volume': 0,
                       'mount_volume': 0,
                       'max_volume_size': 0,
                       'usage_quota': 0}]

        error_info = tenant1.set_datastore_access_privileges(self.auth_mgr.conn, privileges)
        self.assertEqual(error_info, None)
        error_info, tenants_list = self.auth_mgr.list_tenants()
        self.assertEqual(error_info, None)
        
        #Check tenants.id tenant.name, tenant.description and tenant.default_datastore
        self.assertEqual(len(tenants_list), 2)

        # check for tenant1
        tenant1_expected_output = [
                                   tenant1.id,
                                   'tenant1',
                                   'Some tenant',
                                   'default_ds',
                                  ]
        tenant1_actual_output = [
                                 tenants_list[0].id,
                                 tenants_list[0].name,
                                 tenants_list[0].description,
                                 tenants_list[0].default_datastore,
                                ]
        self.assertEqual(tenant1_actual_output, tenant1_expected_output)

        # check vms
        tenant1_expected_output = [(vm1_uuid, 'vm1', tenant1.id),
                                  ]
        tenant1_actual_output = [(tenants_list[0].vms[0][auth_data_const.COL_VM_ID],
                                 tenants_list[0].vms[0][auth_data_const.COL_VM_NAME],
                                 tenants_list[0].vms[0][auth_data_const.COL_TENANT_ID])
                                ]

        self.assertEqual(tenant1_actual_output, tenant1_expected_output)

        # check default_privileges
        tenant1_expected_output = [tenant1.id,
                                   default_privileges[auth_data_const.COL_DATASTORE],
                                   default_privileges[auth_data_const.COL_CREATE_VOLUME],
                                   default_privileges[auth_data_const.COL_DELETE_VOLUME],
                                   default_privileges[auth_data_const.COL_MOUNT_VOLUME],
                                   default_privileges[auth_data_const.COL_MAX_VOLUME_SIZE],
                                   default_privileges[auth_data_const.COL_USAGE_QUOTA]
                                   ]
        tenant1_actual_output = [tenants_list[0].default_privileges[0][auth_data_const.COL_TENANT_ID],
                                 tenants_list[0].default_privileges[0][auth_data_const.COL_DATASTORE],
                                 tenants_list[0].default_privileges[0][auth_data_const.COL_CREATE_VOLUME],
                                 tenants_list[0].default_privileges[0][auth_data_const.COL_DELETE_VOLUME],
                                 tenants_list[0].default_privileges[0][auth_data_const.COL_MOUNT_VOLUME],
                                 tenants_list[0].default_privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                                 tenants_list[0].default_privileges[0][auth_data_const.COL_USAGE_QUOTA]
                                 ]
                
        # check privileges
        tenant1_expected_output = [tenant1.id,
                                   privileges[0][auth_data_const.COL_DATASTORE],
                                   privileges[0][auth_data_const.COL_CREATE_VOLUME],
                                   privileges[0][auth_data_const.COL_DELETE_VOLUME],
                                   privileges[0][auth_data_const.COL_MOUNT_VOLUME],
                                   privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                                   privileges[0][auth_data_const.COL_USAGE_QUOTA]
                                   ]
        tenant1_actual_output = [tenants_list[0].privileges[0][auth_data_const.COL_TENANT_ID],
                                 tenants_list[0].privileges[0][auth_data_const.COL_DATASTORE],
                                 tenants_list[0].privileges[0][auth_data_const.COL_CREATE_VOLUME],
                                 tenants_list[0].privileges[0][auth_data_const.COL_DELETE_VOLUME],
                                 tenants_list[0].privileges[0][auth_data_const.COL_MOUNT_VOLUME],
                                 tenants_list[0].privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                                 tenants_list[0].privileges[0][auth_data_const.COL_USAGE_QUOTA]
                                 ]
                        
        self.assertEqual(tenant1_actual_output, tenant1_expected_output)


        # check for tenant2
        tenant2_expected_output = [
                                   tenant2.id,
                                   'tenant2',
                                   'Some tenant',
                                   'default_ds',
                                  ]
        tenant2_actual_output = [
                                 tenants_list[1].id,
                                 tenants_list[1].name,
                                 tenants_list[1].description,
                                 tenants_list[1].default_datastore,
                                ]
        self.assertEqual(tenant2_actual_output, tenant2_expected_output)

        # check vms
        self.assertEqual(len(tenants_list[1].vms), 2)
        tenant2_expected_output = [(vm2_uuid, 'vm2', tenant2.id),
                                   (vm3_uuid, 'vm3', tenant2.id)
                                  ]
        tenant2_actual_output = [(tenants_list[1].vms[0][auth_data_const.COL_VM_ID],
                                 tenants_list[1].vms[0][auth_data_const.COL_VM_NAME],
                                 tenants_list[1].vms[0][auth_data_const.COL_TENANT_ID]),
                                 (tenants_list[1].vms[1][auth_data_const.COL_VM_ID],
                                 tenants_list[1].vms[1][auth_data_const.COL_VM_NAME],
                                 tenants_list[1].vms[1][auth_data_const.COL_TENANT_ID]),
                                ]
        self.assertEqual(tenant2_actual_output, tenant2_expected_output)

        # check default_privileges
        tenant2_expected_output = [tenant2.id,
                                   default_privileges[auth_data_const.COL_DATASTORE],
                                   default_privileges[auth_data_const.COL_CREATE_VOLUME],
                                   default_privileges[auth_data_const.COL_DELETE_VOLUME],
                                   default_privileges[auth_data_const.COL_MOUNT_VOLUME],
                                   default_privileges[auth_data_const.COL_MAX_VOLUME_SIZE],
                                   default_privileges[auth_data_const.COL_USAGE_QUOTA]
                                   ]
        tenant2_actual_output = [tenants_list[1].default_privileges[0][auth_data_const.COL_TENANT_ID],
                                 tenants_list[1].default_privileges[0][auth_data_const.COL_DATASTORE],
                                 tenants_list[1].default_privileges[0][auth_data_const.COL_CREATE_VOLUME],
                                 tenants_list[1].default_privileges[0][auth_data_const.COL_DELETE_VOLUME],
                                 tenants_list[1].default_privileges[0][auth_data_const.COL_MOUNT_VOLUME],
                                 tenants_list[1].default_privileges[0][auth_data_const.COL_MAX_VOLUME_SIZE],
                                 tenants_list[1].default_privileges[0][auth_data_const.COL_USAGE_QUOTA]
                                 ]   
        self.assertEqual(tenant2_actual_output, tenant2_expected_output)

        # check privileges
        tenant2_expected_output = []
        tenant2_actual_output = tenants_list[1].privileges
        
        self.assertEqual(tenant2_actual_output, tenant2_expected_output)
          
         
    def test_remove_tenants(self):
        vms = [(str(uuid.uuid4()), 'vm1')]
        
        privileges = self.get_privileges()
        default_datastore, default_privileges = self.get_default_datastore_and_privileges()
        error_info, tenant1 = self.auth_mgr.create_tenant('tenant1', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
        self.assertEqual(error_info, None)
        self.assertTrue(uuid.UUID(tenant1.id))
        
        vms = [(str(uuid.uuid4()), 'vm2'), (str(uuid.uuid4()), 'vm3')]
        privileges = []
        error_info, tenant2 = self.auth_mgr.create_tenant('tenant2', 'Some tenant', default_datastore,
                                              default_privileges, vms, privileges)
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
    unittest.main()
