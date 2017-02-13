
/* global define $ */

define([], function() {
  'use strict';

  return function($scope, $q, $rootScope, DialogService, DvolVmGridService, DvolTenantService, GridUtils) {

    var vmsGridActions = [
      {
        id: 'add-vms-button',
        label: 'Add',
        iconClass: 'esx-icon-vm',
        tooltipText: 'Add Virtual Machines',
        enabled: true,
        onClick: function() {
          var selectedTenant = $scope.tenantsGrid.selectedItems[0];
          DialogService.showDialog('dvol.add-vms', {
            selectedTenantRow: selectedTenant,
            save: function(selectedVmsRows) {
              if (!selectedTenant) return; // TODO: async error
              if (!selectedVmsRows) return;
              var selectedVmsIds = selectedVmsRows.map(function(vm) {
                return vm.name;
              });
              if (selectedVmsIds.length < 1) return;
              DvolTenantService.addVmsToTenant(selectedTenant.id, selectedVmsIds)
                .then(vmsGrid.refresh);
            }
          });
        }
      },
      {
        id: 'remove-vm-button',
        label: 'Remove',
        iconClass: 'vui-icon-action-delete',
        tooltipText: 'Remove Virtual Machine',
        enabled: true,
        onClick: function() {
          var selectedTenant = $scope.tenantsGrid.selectedItems[0];
          if (!selectedTenant) return;
          var selectedVm = $scope.vmsGrid.selectedItems[0];
          if (!selectedVm) return;
          DvolTenantService.removeVmFromTenant(selectedTenant.name, selectedVm.name)
            .then(vmsGrid.refresh);
        }
      },
      {
        id: 'refresh-vms-button',
        label: 'Refresh',
        iconClass: 'esx-icon-action-refresh',
        tooltipText: 'Refresh Virtual Machines',
        enabled: true,
        onClick: function() {
          vmsGrid.refresh();
        }
      }
    ];

    function filterVmsForThisTenant(allVms) {
      var selectedTenantRow = $scope.tenantsGrid.selectedItems[0];
      if (!selectedTenantRow) return $q.reject('ERROR: attempting to get VMs for a tenant but no tenant is selected');
      return DvolTenantService.listVmsForTenant(selectedTenantRow.name)
      .then(function(tenantVms) {
        var filteredVms = [];
        allVms.forEach(function(vm) {
          tenantVms.forEach(function(tvm) {
            if (vm.name === tvm) {
              filteredVms.push(vm);
            }
          });
        });
        return filteredVms;
      });
    }

    var vmsGrid = DvolVmGridService.makeVmsGrid('vmsGrid', vmsGridActions, filterVmsForThisTenant, 'SINGLE', true);
    $scope.vmsGrid = vmsGrid.grid;
    $rootScope.vmsGrid = vmsGrid;

    var vmSearchOptions = {
      filters: [
        {
          field: 'name',
          operator: 'contains'
        }
      ],
      placeholder: 'Search'
    };

    GridUtils.addSearch($scope.vmsGrid, vmSearchOptions);

    function findAction(actions, actionId) {
      return actions.filter(function(a) {
        return a.id === actionId;
      })[0];
    }

    $scope.$watch('vmsGrid.selectedItems', function() {
      var removeAction = findAction($scope.vmsGrid.actionBarOptions.actions, 'remove-vm-button');
      if ($scope.vmsGrid.selectedItems.length < 1) {
        removeAction.enabled = false;
      } else {
        removeAction.enabled = true;
      }
    });

  };

});
