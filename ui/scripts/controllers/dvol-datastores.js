
/* global define $ */

define([], function() {
  'use strict';

  return function($scope, $rootScope, DialogService, DvolDatastoreGridService, DvolTenantService, GridUtils) {

    var datastoresGridActions = [
      {
        id: 'add-datastores-button',
        label: 'Add',
        iconClass: 'esx-icon-datastore-add',
        tooltipText: 'Add Datastores',
        enabled: true,
        onClick: function() {  // (evt, action)
          DialogService.showDialog('dvol.add-datastores', {
            save: function(selectedDatastoresRows) {
              var selectedTenant = $scope.tenantsGrid.selectedItems[0];
              if (!selectedTenant) return; // TODO: async error
              if (!selectedDatastoresRows) return;
              var datastores = selectedDatastoresRows.map(function(dr) {
                return {
                  datastore: dr.id,
                  permissions: {
                    create: false,
                    mount: false,
                    remove: false,
                    maxVolume: 0,
                    totalVolume: 0
                  }
                };
              });
              var firstDatastore = datastores[0]; // will be only one in SINGLE mode
              DvolTenantService.addDatastores(selectedTenant.id, datastores)
                .then(datastoresGrid.refresh)
                .then(function() {
                  DialogService.showDialog('dvol.edit-datastore', {
                    permissions: firstDatastore.permissions,
                    save: function(editedPermissions) {
                      DvolTenantService.updateDatastore(selectedTenant.id, { datastore: firstDatastore.datastore, permissions: editedPermissions })
                        .then(datastoresGrid.refresh);
                    }
                  });
                });
            },
            datastoresAlreadyInTenant: DvolTenantService.state.tenants[$scope.tenantsGrid.selectedItems[0].id].datastores
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
          var datastoreId = $scope.datastoresGrid.selectedItems[0].id || $scope.datastoresGrid.selectedItems[0].moid;
          DvolTenantService.get($scope.tenantsGrid.selectedItems[0].id)
          .then(function(tenant) {
            var datastore = tenant.datastores[datastoreId];
            DialogService.showDialog('dvol.edit-datastore', {
              permissions: datastore.permissions,
              save: function(editedPermissions) {
                DvolTenantService.updateDatastore(tenant.id, { datastore: datastoreId, permissions: editedPermissions })
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
          DvolTenantService.removeDatastore(selectedTenant.id, selectedDatastore.moid || selectedDatastore.id)
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
      if (!selectedTenantRow) return [];
      var selectedTenant = DvolTenantService.state.tenants[selectedTenantRow.id];
      var filteredDatastores = allDatastores.filter(function(d) {
        return selectedTenant.datastores[d.id || d.moid];
      }).map(function(d) {
        return {
          datastore: d,
          permissions: selectedTenant.datastores[d.id || d.moid].permissions
        };
      });
      return filteredDatastores;
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
