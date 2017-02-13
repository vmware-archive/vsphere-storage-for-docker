
/* global define $ _ */

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
              var defaultPrivileges = {
                create_volumes: selectedTenant.default_privileges.create_volumes === 'true' ? true : false,
                mount_volumes: selectedTenant.default_privileges.mount_volumes === 'true' ? true : false,
                delete_volumes: selectedTenant.default_privileges.delete_volumes === 'true' ? true : false,
                max_volume_size: selectedTenant.default_privileges.max_volume_size,
                usage_quota: selectedTenant.default_privileges.usage_quota
              };
              var datastores = selectedDatastoresRows.map(function(dr) {
                return {
                  datastore: dr.name,
                  permissions: defaultPrivileges
                };
              });
              //
              // The add datastores grid is set currently to support only single select
              // so we just take the first (and theoretically the only) item in the array
              //
              var originalDatastore = datastores[0]; // will be only one in SINGLE mode
              DvolTenantService.addDatastoreAccessForTenant(selectedTenant.name, originalDatastore)
                .then(datastoresGrid.refresh)
                .then(function() {
                  var originalPermissions = _.clone(originalDatastore.permissions);
                  DialogService.showDialog('dvol.edit-datastore', {
                    permissions: originalDatastore.permissions,
                    save: function(editedPermissions) {
                      DvolTenantService.modifyDatastoreAccessForTenant(
                        selectedTenant.name,
                        {
                          datastore: originalDatastore.datastore,
                          permissions: editedPermissions
                        },
                        originalPermissions
                      )
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
          var tenantName;
          var datastoreName;
          var originalPermissions;
          if ($scope.datastoresGrid.selectedItems.length < 1) return;
          tenantName = $scope.tenantsGrid.selectedItems[0].name;
          datastoreName = $scope.datastoresGrid.selectedItems[0].name;
          DvolTenantService.listDatastoreAccessForTenant(tenantName)
          .then(function(datastoreAccesses) {
            originalPermissions = datastoreAccesses.filter(function(d) {
              return d.datastore === datastoreName;
            })[0].permissions;
            DialogService.showDialog('dvol.edit-datastore', {
              permissions: originalPermissions,
              save: function(editedPermissions) {
                DvolTenantService.modifyDatastoreAccessForTenant(
                  tenantName,
                  {
                    datastore: datastoreName,
                    permissions: editedPermissions
                  },
                  originalPermissions
                )
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
          DvolTenantService.removeDatastoreAccessForTenant(selectedTenant.name, selectedDatastore.name)
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
      var selectedTenantRow = $scope.tenantsGrid.selectedItems[0];
      if (!selectedTenantRow) return $q.reject('ERROR: attempting to get datastores for a tenant but no tenant is selected');
      return DvolTenantService.listDatastoreAccessForTenant(selectedTenantRow.name)
      .then(function(tenantDatastoreAccesses) {
        var filteredDatastores = [];
        allDatastores.forEach(function(ds) {
          tenantDatastoreAccesses.forEach(function(tds) {
            if (ds.name === tds.datastore) {
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
