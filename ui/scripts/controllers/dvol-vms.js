
/* global define $ */

define([], function() {
  'use strict';

  return function($scope, $rootScope, DialogService, DvolVmGridService, DvolTenantService, GridUtils) {

    var vmsGridActions = [
      {
        id: 'add-vms-button',
        label: 'Add',
        iconClass: 'esx-icon-vm',
        tooltipText: 'Add Virtual Machines',
        enabled: true,
        onClick: function() {  // (evt, action)
          var selectedTenant = $scope.tenantsGrid.selectedItems[0];
          DialogService.showDialog('dvol.add-vms', {
            save: function(selectedVmsRows) {
              if (!selectedTenant) return; // TODO: async error
              if (!selectedVmsRows) return;
              var selectedVmsIds = selectedVmsRows.map(function(vm) {
                return vm.moid || vm.id;
              });
              if (selectedVmsIds.length < 1) return;
              DvolTenantService.addVms(selectedTenant.id, selectedVmsIds)
                .then(vmsGrid.refresh);
            },
            vmsAlreadyInTenant: DvolTenantService.state.tenants[$scope.tenantsGrid.selectedItems[0].id].vms
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
          DvolTenantService.removeVm(selectedTenant.id, selectedVm.moid || selectedVm.id)
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
      if (!selectedTenantRow) return [];
      var selectedTenant = DvolTenantService.state.tenants[selectedTenantRow.id];
      var filteredVms = allVms.filter(function(vm) {
        return selectedTenant.vms.indexOf(vm.moid) >= 0;
      });
      return filteredVms;
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
