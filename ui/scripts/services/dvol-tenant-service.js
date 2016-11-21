/* global define _ */

define([], function() {
  'use strict';

  return function(DvolVmodlService) {

    var state = {};

    //
    // Local Application State
    // -----------------------
    //
    // The "state" variable is a representation of the tenants list in memory
    //
    // READING from "state"
    //
    // it's exposed for read-only use via this service's getState() method returning a copy
    // it's meant to be available to the rest of the UI
    // for situations where up-to-date global state isn't required
    // and/or hasn't been requested (e.g. via the refresh buttons) by the user
    //
    // WRITING to "state"
    //
    // currently all mutable state for this UI is contained within the Tenant List
    // so "state" is just a list of tenants (in the form of an object keyed by tenant id)
    // AND any mutation to application state must go through this service
    //
    // "state" is private to this service

    function setState(tenantsArr) {
      var tenantsObj = {};
      tenantsArr.forEach(function(t) {
        tenantsObj[t.id] = t;
      });
      state.tenants = tenantsObj;
      return getState();
    }

    function getState() {
      return _.clone(state);
    }

    //
    // it's expected that any action that communicates with the server will be followed by:
    //
    // 1) obtain the current Tenant List from the server
    //
    // 2) call setState, passing this current Tenant List (as an array) to update local application state
    //
    //
    // IMPORTANT: EVERY API CALL SHOULD CONCLUDE BY CALLING updateLocalState
    //
    function updateLocalState(tenantId) {
      //
      // this should be called after every API action
      // to update the local state with the current system state
      //
      // the VMODL api does not currently return the results for a given action
      // e.g. addTenant might eventually return the newly added tenant
      //
      // for now, we're simulating that behavior here
      // we return tenant list (array) unless a tenantId is given
      // if so, return that Tenant
      //
      return getAll()
      .then(function(tenants) {
        setState(tenants);
        return tenantId ? getState(tenantId) : tenants;
      });
    }

    //
    // a few utils
    //

    function dedupe(a) {
      return a.filter(function(item, pos) {
        return a.indexOf(item) === pos;
      });
    }

    function pickTenantById(tenantId, tenants) {
      var matches = tenants.filter(function(t) {
        return t.id === tenantId;
      });
      if (matches.length === 0) {
        return null;
      }
      if (matches.length > 1) {
        throw Error('NonUniqueError: pickTenantById returned multiple matches for id: ' + tenantId);
      }
      return matches[0];
    }

    function getRightsFromPermissions(perms) {
      return ['create', 'mount', 'delete'].filter(function(r) {
        return perms[r + '_volumes'];
      });
    }

    //
    //
    //
    // The VMODL API Wrapper
    //

    //
    // eventually the VMODL api will support get Tenant by ID
    // for now we just use getAll
    //
    function get(tenantId) {
      return getAll()
      .then(function(tenants) {
        return setState(tenants)[tenantId];
      });
    }

    function getAll() {
      return DvolVmodlService.listTenants()
      .then(setState);
    }

    //
    // In the UI, creating a tenant can optionally involve
    // associating a set of VMs with that tenant
    // These actions are distinct in the VMODL
    // so we make the calls in series here
    //
    function add(tenant, vms) {
      var tenantArgs = {
        name: tenant.name,
        description: tenant.description,
        default_privileges: tenant.default_privileges
      };
      return   DvolVmodlService.createTenant(tenantArgs)
      .then(function() {
        var vmsArgs = {
          name: tenant.name,
          vms: vms
        };
        return DvolVmodlService.addVMsToTenant(vmsArgs);
      })
      .then(function() {
        return updateLocalState(tenant.name);
      });
    }

    function remove(tenantId) {
      DvolVmodlService.removeTenant(tenantId)
      .then(function() {
        return updateLocalState();
      });
    }

    function removeDatastore(tenantId, datastoreId) {
      return DvolVmodlService.removeDatastoreAccessForTenant(tenantId, datastoreId)
      .then(function() {
        return updateLocalState(tenantId);
      });
    }

    //
    // NOTE: the VMODL api supports removing multiple VMs in same call
    // Our original removeVm implementation supports only one,
    // so we just call the api with a single vm
    // and not take advantage of the multiple vm option for now
    //
    function removeVm(tenantId, vmId) {
      return DvolVmodlService.removeVMsFromTenant(tenantId, [vmId])
      .then(function() {
        return updateLocalState(tenantId);
      });
    }

    function addVms(tenantId, vmIds) {
      return DvolVmodlService.addVMsToTenant({
        name: tenantId,
        vms: vmIds
      })
      .then(function() {
        return updateLocalState(tenantId);
      });
    }

    function addDatastore(tenantId, datastore) {
      return DvolVmodlService.addDatastoreAccessForTenant({
        name: tenantId,
        datastore: datastore.datastore.name,
        rights: getRightsFromPermissions(datastore.permissions),
        volume_maxsize: datastore.permissions.volume_maxsize,
        volume_totalsize: datastore.permissions.volume_totalsize
      })
      .then(function() {
        return updateLocalState(tenantId);
      });
    }

    function updateDatastore(tenantId, updatedDatastore) {
      var add_rights = [];
      var remove_rights = [];
      ['create', 'mount', 'delete'].forEach(function(p) {
        if (updatedDatastore.permissions[p + '_volumes']) {
          add_rights.push(p);
        } else {
          remove_rights.push(p);
        }
      });
      return DvolVmodlService.modifyDatastoreAccessForTenant({
        name: tenantId,
        datastore: updatedDatastore.datastore.name,
        add_rights: add_rights,
        remove_rights: remove_rights,
        volume_maxsize: updatedDatastore.permissions.volume_maxsize,
        volume_totalsize: updatedDatastore.permissions.volume_totalsize
      })
      .then(function() {
        return updateLocalState(tenantId);
      });
    }

    function wrapActionWithSetState(action) {
      //
      // actionWithSetState receives arguments passed to action
      // follows up the action by getting the current Tenant List from the VMODL api
      // and the returns action's original return value
      //
      function actionWithSetState() {
        var theActionsOriginalReturnValue;
        action.apply(this, arguments)
        .then(function(rv) {
          theActionsOriginalReturnValue = rv;
          return getAll();
        })
        .then(function(tenants) {
          setState(tenants);
          return theActionsOriginalReturnValue || tenants;
        });
      }
      return actionWithSetState;
    }

    this.getAll = wrapActionWithSetState(getAll);
    this.removeDatastore = wrapActionWithSetState(removeDatastore);
    this.removeVm = wrapActionWithSetState(removeVm);
    this.remove = wrapActionWithSetState(remove);
    this.get = wrapActionWithSetState(get);
    this.add = wrapActionWithSetState(add);
    this.addVms = wrapActionWithSetState(addVms);
    this.addDatastore = wrapActionWithSetState(addDatastore);
    this.updateDatastore = wrapActionWithSetState(updateDatastore);
    this.getState = wrapActionWithSetState(getState);

  };

});
