/* global define $ */

define([], function() {
  'use strict';

  return function($scope, DialogService, DvolDatastoreGridService, GridUtils) {

    var datastoresAlreadyInTenant = DialogService.currentDialog().opaque.datastoresAlreadyInTenant;
    function filterFn(datastores) {
      return datastores.filter(function(d) {
        return !datastoresAlreadyInTenant[d.id || d.moid];
      });
    }
    var grid = DvolDatastoreGridService.makeDatastoresGrid('availableDatastoresGrid', [], filterFn);

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
        DialogService.currentDialog().opaque.save($scope.availableDatastoresGrid
          .selectedItems);
        return true;
      }
    });

  };

});
