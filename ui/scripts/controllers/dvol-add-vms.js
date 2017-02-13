/* global define $ */

define([], function() {
  'use strict';

  return function($scope, $q, DialogService, DvolTenantService, DvolVmGridService, GridUtils) {

    function filterVmsNotInThisTenant(allVms) {
      var selectedTenantRow = DialogService.currentDialog().opaque.selectedTenantRow;
      if (!selectedTenantRow) return $q.reject('ERROR: attempting to get VMs for a tenant but no tenant is selected');
      return DvolTenantService.listVmsForTenant(selectedTenantRow.name)
      .then(function(tenantVms) {
        var availableVms = [];
        allVms.forEach(function(vm) {
          var alreadyInTenant;
          tenantVms.forEach(function(tvm) {
            if (vm.name === tvm) {
              alreadyInTenant = true;
            }
          });
          if (!alreadyInTenant) {
            availableVms.push(vm);
          }
        });
        return availableVms;
      });
    }

    var grid = DvolVmGridService.makeVmsGrid('availableVmsGrid', [], filterVmsNotInThisTenant, 'MULTI', false);

    $scope.availableVmsGrid = grid.grid;

    var vmSearchOptions = {
      filters: [
        {
          field: 'name',
          operator: 'contains'
        }
      ],
      placeholder: 'Search'
    };

    GridUtils.addSearch($scope.availableVmsGrid, vmSearchOptions);

    DialogService.setConfirmOptions({
      label: 'Add',
      onClick: function() {
        DialogService.currentDialog().opaque.save($scope.availableVmsGrid
          .selectedItems);
        return true;
      }
    });

  };

});
