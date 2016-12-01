/* global define */

define([], function() {
  'use strict';

  return function($rootScope, $scope, $log, $state, $filter, $timeout, GridUtils, vuiConstants, DialogService,
    DvolTenantService, DvolTenantGridService) {

    var translate = $filter('translate');

    var tenantGridActions = [
      {
        id: 'add-tenant-button',
        label: 'Add',
        iconClass: 'vui-icon-action-add',
        tooltipText: 'Add Tenant',
        enabled: true,
        onClick: function() {
          DialogService.showDialog('dvol.add-tenant', {
            tenant: {},
            save: function(newTenant, vms) {
              var vmIds = vms.map(function(vm) {
                return vm.name;
              });
              DvolTenantService.createTenant(newTenant, vmIds)
                .then(tenantsGrid.refresh);
            }
          });
        }
      },
      {
        id: 'edit-tenant-button',
        label: 'Edit',
        iconClass: 'vui-icon-action-edit',
        tooltipText: 'Edit Tenant',
        enabled: true,  // enable this once the API supports it
        onClick: function() {
          if ($scope.tenantsGrid.selectedItems.length < 1) return;
          DvolTenantService.getTenantByName($scope.tenantsGrid.selectedItems[0].name)
          .then(function(tenant) {
            DialogService.showDialog('dvol.edit-tenant', {
              tenant: tenant,
              editMode: true
              //
              // The vmodl api doesn't currently support the modifyTenant action
              // ,
              // save: function(newTenantValues) {
              //
              // Once it does we might do something like this
              //
              // DvolTenantService.modifyTenant(newTenantValues)
              // .then(tenantsGrid.refresh);
              //
              // }
            });

          });
        }
      },
      {
        id: 'remove-tenant-button',
        label: 'Remove',
        iconClass: 'vui-icon-action-delete',
        tooltipText: 'Remove Tenant',
        enabled: false,
        onClick: function() {
          var selectedTenant = $scope.tenantsGrid.selectedItems[0];
          if (!selectedTenant) return;
          DvolTenantService.removeTenant(selectedTenant.name);
          tenantsGrid.refresh();
        }
      },
      {
        id: 'refresh-tenants-button',
        label: 'Refresh',
        iconClass: 'esx-icon-action-refresh',
        tooltipText: 'Refresh Tenants',
        enabled: true,
        onClick: function() {
          tenantsGrid.refresh();
        }
      }
    ];

    var tenantsGrid = DvolTenantGridService.makeTenantsGrid(tenantGridActions);
    $scope.tenantsGrid = tenantsGrid.grid;

    var tenantSearchOptions = {
      filters: [
        {
          field: 'name',
          operator: 'contains'
        },
        {
          field: 'description',
          operator: 'contains'
        }
      ],
      placeholder: 'Search'
    };

    GridUtils.addSearch($scope.tenantsGrid, tenantSearchOptions);

    function findAction(actions, actionId) {
      return actions.filter(function(a) {
        return a.id === actionId;
      })[0];
    }

    $scope.$watch('tenantsGrid.selectedItems', function(newVal, oldVal) {
      var editAction = findAction($scope.tenantsGrid.actionBarOptions.actions, 'edit-tenant-button');
      var removeAction = findAction($scope.tenantsGrid.actionBarOptions.actions, 'remove-tenant-button');
      if ($scope.tenantsGrid.selectedItems.length < 1) {
        editAction.enabled = false;
        removeAction.enabled = false;
      } else {
        editAction.enabled = true;
        removeAction.enabled = true;
      }
      if (newVal !== oldVal && $scope.tenantsGrid.selectedItems.length > 0) {
        $rootScope.vmsGrid && $rootScope.vmsGrid.refresh();
        $rootScope.datastoresGrid && $rootScope.datastoresGrid.refresh();
      }
    });

    //
    // TENANT DETAIL TABS
    //

    var tabs = {
      vms: {
        label: translate('dvol.tenantDetailTabs.vms.label'),
        tooltipText: translate('dvol.tenantDetailTabs.vms.tooltip'),
        contentUrl: 'plugins/docker-volume-plugin/views/dvol-vms.html'
      },
      datastores: {
        label: translate('dvol.tenantDetailTabs.datastores.label'),
        tooltipText: translate(
          'dvol.tenantDetailTabs.datastores.tooltip'),
        contentUrl: 'plugins/docker-volume-plugin/views/dvol-datastores.html'
      }
    };

    $scope.tenantDetailTabs = {
      tabs: Object.keys(tabs).map(function(key) {
        return tabs[key];
      }),
      tabType: vuiConstants.tabs.type.SECONDARY,
      selectedTabIndex: 0
    };

    var defaultTabIndex = 0;
    $scope.tenantDetailTabs.selectedTabIndex = defaultTabIndex;
    $scope.tenantDetailTabs.tabs[defaultTabIndex].loaded = true;

  };
});
