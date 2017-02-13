/* global define */


define([
  'angular',
  //
  // Controllers
  //
  'plugins/docker-volume-plugin/scripts/controllers/dvol-main.js',
  'plugins/docker-volume-plugin/scripts/controllers/dvol-add-tenant.js',
  'plugins/docker-volume-plugin/scripts/controllers/dvol-add-vms.js',
  'plugins/docker-volume-plugin/scripts/controllers/dvol-add-datastores.js',
  'plugins/docker-volume-plugin/scripts/controllers/dvol-edit-datastore.js',
  'plugins/docker-volume-plugin/scripts/controllers/dvol-vms.js',
  'plugins/docker-volume-plugin/scripts/controllers/dvol-datastores.js',
  //
  // Services
  //
  'plugins/docker-volume-plugin/scripts/services/dvol-context-menu.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-dialog-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-vm-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-vm-grid-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-tenant-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-datastore-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-datastore-grid-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-tenant-grid-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-soap-service.js',
  'plugins/docker-volume-plugin/scripts/services/dvol-vmodl-service.js',
  'services/grid-utils',
  'services/vm/vm'
  //
], function(
  angular,
  //
  // Controllers
  //
  DvolMainController,
  DvolAddTenantController,
  DvolAddVmsController,
  DvolAddDatastoresController,
  DvolEditDatastoreController,
  DvolVms,
  DvolDatastores,
  //
  // Services
  //
  DvolContextMenuService,
  DvolDialogService,
  DvolVmService,
  DvolVmGridService,
  DvolTenantService,
  DvolDatastoreService,
  DvolDatastoreGridService,
  DvolTenantGridService,
  DvolSoapService,
  DvolVmodlService,
  GridUtils,
  VMService,
  StorageService,
  vuiConstants
) {

  'use strict';

  return angular.module('esxUiApp.plugins.dvol', [
    'ui.router'
  ])
  .controller({
    'DvolMainController': DvolMainController,
    'DvolAddTenantController': DvolAddTenantController,
    'DvolAddVmsController': DvolAddVmsController,
    'DvolAddDatastoresController': DvolAddDatastoresController,
    'DvolEditDatastoreController': DvolEditDatastoreController,
    'DvolVms': DvolVms,
    'DvolDatastores': DvolDatastores
  })
  .service({
    'DvolDialogService': DvolDialogService,
    'DvolContextMenuService': DvolContextMenuService,
    'DvolVmService': DvolVmService,
    'DvolVmGridService': DvolVmGridService,
    'DvolTenantService': DvolTenantService,
    'DvolDatastoreService': DvolDatastoreService,
    'DvolDatastoreGridService': DvolDatastoreGridService,
    'DvolTenantGridService': DvolTenantGridService,
    'DvolSoapService': DvolSoapService,
    'DvolVmodlService': DvolVmodlService,
    'GridUtils': GridUtils,
    'vuiConstants': vuiConstants
  })
  .run(function($rootScope, $filter, PluginService) {
    var translate = $filter('translate');

    PluginService.register({
      name: 'docker-volume-plugin',
      version: '1.0.0',
      api: '=1.0.0',
      stylesheets: [
        'styles/main.css'
      ],
      contextMenuServices: [
        'DvolContextMenuService'
      ],
      dialogServices: [
        'DvolDialogService'
      ],
      navigator: [{
        title: translate('dvol.menu.title'),
        icon: 'icon-example-menu',
        state: 'host.docker-volume-plugin',
        onContext: function(e) {
          $rootScope.contextMenu.show('dvol', ['object'], e);
        },
        children: [{
          title: translate('dvol.menu.titleChildOne'),
          icon: 'icon-example-menu',
          state: 'host.docker-volume-plugin.tenants',
          onContext: function(e) {
            $rootScope.contextMenu.show('dvol', ['object'],
              e);
          }
        }]
      }],
      states: [{
        name: 'host.docker-volume-plugin',
        options: {
          url: '/docker-volume-plugin',
          views: {
            'content@host': {
              templateUrl: 'plugins/docker-volume-plugin/views/dvol-main.html'
            }
          }
        }
      }, {
        name: 'host.docker-volume-plugin.tenants',
        options: {
          url: '/tenants'
        }
      }]
    });
  });
});
