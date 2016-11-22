
/* global define $ */

define([], function() {
  'use strict';

  return function($scope, $rootScope, $q, DialogService, DvolDatastoreGridService, DvolTenantService, GridUtils) {

    var datastoresGridActions = [
      {
        id: 'add-datastores-button',
        label: 'Add',
        iconClass: 'esx-icon-datastore-add',
        tooltipText: 'Add Datastores',
        enabled: true,
        onClick: function() {  // (evt, action)
          DialogService.showDialog('dvol.add-datastores', {
            selectedTenantRow: $scope.tenantsGrid.selectedItems[0],
            save: function(selectedDatastoresRows) {
              var selectedTenant = $scope.tenantsGrid.selectedItems[0];
              if (!selectedTenant) return; // TODO: async error
              if (!selectedDatastoresRows) return;
              var datastores = selectedDatastoresRows.map(function(dr) {
                return {
                  datastore: dr.name,

                  //
                  // TODO: replace these hard-coded default perms with those that come back with (newly created) tenant
                  //

                  permissions: {
                    create_volumes: false,
                    mount_volumes: false,
                    delete_volumes: false,
                    max_volume_size: '',
                    usage_quota: ''
                  }
                };
              });
              //
              // originally there was support for multi-select of datastores
              // this is where we transition from that model to a single datastore at a time model
              // by taking only the first in the array
              //
              var firstDatastore = datastores[0]; // will be only one in SINGLE mode
              //
              //
              //
              DvolTenantService.addDatastoreAccessForTenant(selectedTenant.id, firstDatastore)
                .then(datastoresGrid.refresh)
                .then(function() {
                  DialogService.showDialog('dvol.edit-datastore', {
                    permissions: firstDatastore.permissions,
                    save: function(editedPermissions) {
                      DvolTenantService.modifyDatastoreAccessForTenant(
                        selectedTenant.id,
                        {
                          datastore: firstDatastore.datastore,
                          permissions: editedPermissions
                        },
                        //
                        // a bit of a hack here
                        // originally the UI expected an API that received a new permission set
                        // as the way to update a datastore
                        //
                        // the actual API uses an add_rights & remove_right approach to modifying
                        // the create, delete, mount permissions
                        //
                        // so we pass down the original permissions so the "deltas" can be figured out
                        //
                        firstDatastore.permissions)
                        .then(datastoresGrid.refresh);
                    }
                  });
                });
            }
          });
        }
      },
      {
        id: 'edit-datastore-button',
        label: 'Edit',
        iconClass: 'vui-icon-action-edit',
        enabled: true,
        onClick: function() {
          if ($scope.datastoresGrid.selectedItems.length < 1) return;
          var datastoreId = $scope.datastoresGrid.selectedItems[0].name;
          DvolTenantService.getTenantByName($scope.tenantsGrid.selectedItems[0].id)
          .then(function(tenant) {
            var datastore = tenant.datastores[datastoreId];
            DialogService.showDialog('dvol.edit-datastore', {
              permissions: datastore.permissions,
              save: function(editedPermissions) {
                DvolTenantService.modifyDatastoreAccessForTenant(tenant.id, { datastore: datastoreId, permissions: editedPermissions })
                  .then(datastoresGrid.refresh);
              }
            });
          });
        }
      },
      {
        id: 'remove-datastore-button',
        label: 'Remove',
        iconClass: 'vui-icon-action-delete',
        tooltipText: 'Remove Datastore',
        enabled: true,
        onClick: function() {
          var selectedTenant = $scope.tenantsGrid.selectedItems[0];
          if (!selectedTenant) return;
          var selectedDatastore = $scope.datastoresGrid.selectedItems[0];
          if (!selectedDatastore) return;
          DvolTenantService.removeDatastoreAccessForTenant(selectedTenant.id, selectedDatastore.name)
            .then(datastoresGrid.refresh);
        }
      },
      {
        id: 'refresh-datastores-button',
        label: 'Refresh',
        iconClass: 'esx-icon-action-refresh',
        tooltipText: 'Refresh Datastores',
        enabled: true,
        onClick: function() {
          datastoresGrid.refresh();
        }
      }
    ];

    function filterDatastoresForThisTenant(allDatastores) {
      // NOTE: selectedTenants from the grid doesn't have new datastores added (will not until grid refresh)
      // we don't want to refresh the grid because we'll lose tenant row selection
      var selectedTenantRow = $scope.tenantsGrid.selectedItems[0];
      if (!selectedTenantRow) return $q.reject('ERROR: attempting to filter datastores but no tenant is selected');
      return DvolTenantService.listDatastoreAccessForTenant(selectedTenantRow.id)
      .then(function(tenantDatastoreAccesses) {
        var filteredDatastores = [];
        allDatastores.forEach(function(ds) {
          tenantDatastoreAccesses.forEach(function(tds) {
            if (ds.name === tds.id) {
              filteredDatastores.push({
                datastore: ds,
                permissions: tds.permissions
              });
            }
          });
        });
        return filteredDatastores;
      });
    }

    var datastoresGrid = DvolDatastoreGridService.makeDatastoresGrid('datastoresGrid', datastoresGridActions, filterDatastoresForThisTenant, true);
    $scope.datastoresGrid = datastoresGrid.grid;
    $rootScope.datastoresGrid = datastoresGrid;

    var datastoreSearchOptions = {
      filters: [
        {
          field: 'name',
          operator: 'contains'
        }
      ],
      placeholder: 'Search'
    };

    GridUtils.addSearch($scope.datastoresGrid, datastoreSearchOptions);

    function findAction(actions, actionId) {
      return actions.filter(function(a) {
        return a.id === actionId;
      })[0];
    }

    $scope.$watch('datastoresGrid.selectedItems', function() {
      var editAction = findAction($scope.datastoresGrid.actionBarOptions.actions, 'edit-datastore-button');
      var removeAction = findAction($scope.datastoresGrid.actionBarOptions.actions, 'remove-datastore-button');
      if ($scope.datastoresGrid.selectedItems.length < 1) {
        editAction.enabled = false;
        removeAction.enabled = false;
      } else {
        editAction.enabled = true;
        removeAction.enabled = true;
      }
    });

  };

});
