/* global define */

define([], function() {
  'use strict';

  return function($rootScope, $scope, $log, $state, $filter, $timeout, GridUtils, vuiConstants, DialogService,
    DvolTenantService, DvolTenantGridService, DvolVsanService) {

    // -----------------------------
    //
    // Temporary
    //
    // Here when the app is loaded we're temporarily making
    // a call to DvolVsanService to test connectivity
    //
    // TODO: remove this
    //

    var mockTenant = {
      name: 'ui-created-tenant',
      description: 'a mock tenant named ui-created-tenant'
    };

    var mockDatastore = {
      name: 'datastore1'
    };

    DvolVsanService.createTenant(mockTenant)
    .then(DvolVsanService.listTenants())
    .then(DvolVsanService.addDatastoreAccessForTenant({
      name: mockTenant.name,
      datastore: mockDatastore.name,
      rights: ['create', 'mount'],
      volume_maxsize: '550MB',
      volume_totalsize: '3TB'
    }))
    .then(DvolVsanService.modifyDatastoreAccessForTenant({
      name: mockTenant.name,
      datastore: mockDatastore.name,
      add_rights: ['delete'],
      remove_righs: ['mount'],
      volume_maxsize: '650MB',
      volume_totalsize: '4TB'
    }))
    .then(DvolVsanService.removeTenant({
      name: mockTenant.name
    }))
    .then(DvolVsanService.addVMsToTenant({
      name: mockTenant.name,
      vms: ['virual-machine-1', 'virtual-machine-2', 'virtual-machine-3']
    }))
    .then(DvolVsanService.removeVMsFromTenant({
      name: mockTenant.name,
      vms: ['virtual-machine-1']
    }))
    .then(DvolVsanService.listVMsForTenant({
      name: mockTenant.name
    }))
    .then(DvolVsanService.getDatastoreAccessPrivileges())
    .then(DvolVsanService.createDatastoreAccessPrivileges({
      datastore: mockDatastore.name,
      create_volumes: true,
      delete_volumes: false,
      mount_volumes: false,
      max_volume_size: '600MB',
      usage_quota: '3TB'
    }))
    .then(DvolVsanService.removeDatastoreAccessForTenant({
      name: mockTenant.name,
      datastore: mockDatastore.name
    }))
    .then(DvolVsanService.listDatastoreAccessForTenant({
      name: mockTenant.name
    }));


    //
    // -----------------------------
    //

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
              DvolTenantService.add(newTenant, vms)
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
        enabled: true,
        onClick: function() {
          if ($scope.tenantsGrid.selectedItems.length < 1) return;
          DvolTenantService.get($scope.tenantsGrid.selectedItems[0].id)
          .then(function(tenant) {
            DialogService.showDialog('dvol.edit-tenant', {
              tenant: tenant,
              editMode: true,
              save: function(newTenant) {
                DvolTenantService.update(newTenant)
                  .then(tenantsGrid.refresh);
              }
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
          DvolTenantService.remove(selectedTenant.id);
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

    $scope.$watch('tenantsGrid.selectedItems', function() {
      var editAction = findAction($scope.tenantsGrid.actionBarOptions.actions, 'edit-tenant-button');
      var removeAction = findAction($scope.tenantsGrid.actionBarOptions.actions, 'remove-tenant-button');
      if ($scope.tenantsGrid.selectedItems.length < 1) {
        editAction.enabled = false;
        removeAction.enabled = false;
      } else {
        editAction.enabled = true;
        removeAction.enabled = true;
      }
      $rootScope.vmsGrid && $rootScope.vmsGrid.refresh();
      $rootScope.datastoresGrid && $rootScope.datastoresGrid.refresh();
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
