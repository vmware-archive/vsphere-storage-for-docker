/* global define _ */

define([], function() {
  'use strict';

  return function(DvolVmodlService) {

    //
    //
    //
    // The VMODL API Wrapper
    //

    //
    // eventually the VMODL api will support get Tenant by Name
    // for now we just use listTenants
    //
    function getTenantByName(tenantName) {
      return listTenants()
      .then(function(tenants) {
        var matches = tenants.filter(function(t) {
          return t.name === tenantName;
        });
        return matches.length > 0 && matches[0];
      });
    }

    function listTenants() {
      return DvolVmodlService.listTenants();
    }

    //
    // In the UI, creating a tenant can optionally involve
    // associating a set of VMs with that tenant
    // These actions are distinct in the VMODL
    // so we make the calls in series here
    //
    function createTenant(tenant, vms) {
      var tenantArgs = {
        name: tenant.name,
        description: tenant.description,
        default_privileges: tenant.default_privileges
      };
      var p = DvolVmodlService.createTenant(tenantArgs);
      if (vms && vms.length > 0) {
        return p.then(function() {
          var vmsArgs = {
            name: tenant.name,
            vms: vms
          };
          return DvolVmodlService.addVMsToTenant(vmsArgs);
        });
      }
      return p;
    }

    function removeTenant(tenantId) {
      DvolVmodlService.removeTenant({
        name: tenantId
      });
    }

    function listDatastoreAccessForTenant(tenantId) {
      return DvolVmodlService.listDatastoreAccessForTenant({
        name: tenantId
      });
    }


    function removeDatastoreAccessForTenant(tenantId, datastoreId) {
      return DvolVmodlService.removeDatastoreAccessForTenant({
        name: tenantId,
        datastore: datastoreId
      });
    }

    //
    // The VMODL api supports removing multiple VMs in the same call
    // Our original removeVmFromTenant implementation supports only one,
    // So we just call the api with a singleton vm
    // not taking advantage of the multiple vm option for now
    //
    function removeVmFromTenant(tenantId, vmId) {
      return DvolVmodlService.removeVMsFromTenant({
        name: tenantId,
        vms: [vmId]
      });
    }

    function listVmsForTenant(tenantId) {
      return DvolVmodlService.listVmsForTenant({
        name: tenantId
      });
    }

    function addVmsToTenant(tenantId, vmIds) {
      return DvolVmodlService.addVMsToTenant({
        name: tenantId,
        vms: vmIds
      });
    }

    //
    // a utility function
    //
    function transformPermissionsToRights(perms) {
      return ['create', 'mount', 'delete'].filter(function(r) {
        return perms[r + '_volumes'];
      });
    }
    //
    function addDatastoreAccessForTenant(tenantId, datastore) {
      return DvolVmodlService.addDatastoreAccessForTenant({
        name: tenantId,
        datastore: datastore.datastore.name,
        rights: transformPermissionsToRights(datastore.permissions),
        volume_maxsize: datastore.permissions.volume_maxsize,
        volume_totalsize: datastore.permissions.volume_totalsize
      });
    }

    function modifyDatastoreAccessForTenant(tenantId, updatedDatastore) {
      var addRights = [];
      var removeRights = [];
      ['create', 'mount', 'delete'].forEach(function(p) {
        if (updatedDatastore.permissions[p + '_volumes']) {
          addRights.push(p);
        } else {
          removeRights.push(p);
        }
      });
      return DvolVmodlService.modifyDatastoreAccessForTenant({
        name: tenantId,
        datastore: updatedDatastore.datastore,
        add_rights: addRights,
        remove_rights: removeRights,
        max_volume_size: updatedDatastore.permissions.max_volume_size,
        usage_quota: updatedDatastore.permissions.usage_quota
      });
    }

    this.listTenants = listTenants;
    this.getTenantByName = getTenantByName;
    this.createTenant = createTenant;
    this.removeTenant = removeTenant;
    this.listVmsForTenant = listVmsForTenant;
    this.addVmsToTenant = addVmsToTenant;
    this.addDatastoreAccessForTenant = addDatastoreAccessForTenant;
    this.listDatastoreAccessForTenant = listDatastoreAccessForTenant;
    this.modifyDatastoreAccessForTenant = modifyDatastoreAccessForTenant;
    this.removeDatastoreAccessForTenant = removeDatastoreAccessForTenant;
    this.removeVmFromTenant = removeVmFromTenant;

  };

});
