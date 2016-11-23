/* global define $ */

define([], function() {
  'use strict';

  return function($scope, $q, DialogService, DvolTenantService, DvolDatastoreGridService, GridUtils) {

    function filterDatastoresNotInThisTenant(allDatastores) {
      var selectedTenantRow = DialogService.currentDialog().opaque.selectedTenantRow;
      if (!selectedTenantRow) return $q.reject('ERROR: attempting to get datastores for a tenant but no tenant is selected');
      return DvolTenantService.listDatastoreAccessForTenant(selectedTenantRow.name)
      .then(function(tenantDatastoreAccesses) {
        var availableDatastores = [];
        allDatastores.forEach(function(ds) {
          var alreadyInTenant;
          tenantDatastoreAccesses.forEach(function(tds) {
            if (ds.name === tds.datastore) {
              alreadyInTenant = true;
            }
          });
          if (!alreadyInTenant) {
            availableDatastores.push(ds);
          }
        });
        return availableDatastores;
      });
    }


    var grid = DvolDatastoreGridService.makeDatastoresGrid('availableDatastoresGrid', [], filterDatastoresNotInThisTenant);

    $scope.availableDatastoresGrid = grid.grid;

    var datastoreSearchOptions = {
      filters: [
        {
          field: 'name',
          operator: 'contains'
        }
      ],
      placeholder: 'Search'
    };

    GridUtils.addSearch($scope.availableDatastoresGrid, datastoreSearchOptions);

    DialogService.setConfirmOptions({
      label: 'Add',
      onClick: function() {
        DialogService.currentDialog().opaque.save($scope.availableDatastoresGrid.selectedItems);
        return true;
      }
    });

  };

});
