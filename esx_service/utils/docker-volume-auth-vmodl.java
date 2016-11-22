/*
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
*/

/*
*** NOTE: this is draft, the actual model will be done via Python extension"
*/

import vmodl.*;

/**
 *
 * DockerVolumeAuthManager is the top-level managed object that provides APIs to manage docker volume
 *  authorization objects such as tenants and datastore access privileges.
 *
 * <p>
 * <b>Tenants</b> are a group of VMs that are granted privileges to create, delete and mount
 * docker volumes (VMDKs) on one or more datastores. Tenants provide for full isolation of these
 * volumes such that one tenant cannot see or manipulate volumes of another tenant even if both
 * tenants have volumes residing on the same datastore. The sole exception to this rule is if the
 * <p>
 * <b>DatastoreAccessPrivileges</b> are predefined operations against a datastore and limits on those
 * operations.
 */
@managed public interface DockerVolumeAuthManager {

   /* An array of references to Tenant managed entities */
   @readonly @optional Tenant[] tenants();

   /* VMs that are not members of a tenant */
   @readonly @optional VirtualMachine[] availableVMs();

   @task Tenant createTenant(
         String name,
         String description,
         VirtualMachine[] vms,
         DatastoreAccessPrivileges[] privileges)
   throws AlreadyExists;

   @task Tenant removeTenant(String id, Boolean deleteVolumes);
   @task Tenant[] listTenants();
};

@managed public interface DockerVolumeTenant {
   /* A unique generated ID that cannot be changed */
   @readonly String id();

   /* A modifiable string */
   @readonly String name();
   @readonly String description();
   @readonly String defaultDatastore;
   @readonly DatastoreAccessPrivileges defaulPrivileges;
   @readonly @optional VirtualMachine[] vms();
   @readonly @optional DatastoreAccessPrivileges[] privileges();

   @task void addVms(VirtualMachine[] vm) throws AlreadyExists;
   @task void removeVms(VirtualMachine[] vm) throws NotFound;

   
   @task void setDatastoreAccessPrivileges(
         DatastoreAccessPrivileges[] privileges);
          
   @task void setName(String name);
   @task void setDescription(String name);

};


/* A set of privileges applied to a specific datastore for a given tenant */
@data public static class DatastoreAccessPrivileges {
   Datastore datastore;
   boolean createVolumes;
   boolean deleteVolumes;
   boolean mountVolumes;
   // The maximum size of any volume created in this datastore
   int maxVolumeSize;
   // The maximum amount of storage that can be used in a datastore in bytes
   int usageQuota;
}
