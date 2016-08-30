define([
   'angular',
   'plugins/esxui-example-plugin/scripts/controllers/example-main.js',
   'plugins/esxui-example-plugin/scripts/controllers/example-tab-one.js',
   'plugins/esxui-example-plugin/scripts/controllers/example-tab-two.js',
   'plugins/esxui-example-plugin/scripts/controllers/wizard/test-page-one.js',
   'plugins/esxui-example-plugin/scripts/controllers/wizard/test-page-two.js',
   'plugins/esxui-example-plugin/scripts/services/example.js',
   'plugins/esxui-example-plugin/scripts/services/example-dialog.js',
   'plugins/esxui-example-plugin/scripts/services/example-wizard.js',
   'plugins/esxui-example-plugin/scripts/services/example-context-menu.js'
], function(
   angular,
   ExampleMainController,
   ExampleTabOneController,
   ExampleTabTwoController,
   ExampleWizardTestPageOneController,
   ExampleWizardTestPageTwoController,
   ExampleService,
   ExampleDialogService,
   ExampleWizardService,
   ExampleContextMenuService) {

   'use strict';

   return angular.module('esxUiApp.plugins.example', [
      'ui.router',
   ]).controller({
      'ExampleMainController': ExampleMainController,
      'ExampleTabOneController': ExampleTabOneController,
      'ExampleTabTwoController': ExampleTabTwoController,
      'ExampleWizardTestPageOneController': ExampleWizardTestPageOneController,
      'ExampleWizardTestPageTwoController': ExampleWizardTestPageTwoController
   }).service({
      'ExampleService': ExampleService,
      'ExampleDialogService': ExampleDialogService,
      'ExampleWizardService': ExampleWizardService,
      'ExampleContextMenuService': ExampleContextMenuService
   }).run(function ($rootScope, $filter, PluginService) {
      var translate = $filter('translate');

      PluginService.register({
         name: 'esxui-example-plugin',
         version: '1.0.0',
         api: '=1.0.0',
         stylesheets: [
            'styles/main.css'
         ],
         contextMenuServices: [
            'ExampleContextMenuService'
         ],
         dialogServices: [
            'ExampleDialogService'
         ],
         wizardServices: [
            'ExampleWizardService'
         ],
         navigator: [{
            title: translate('example.menu.title'),
            icon: 'icon-example-menu',
            state: 'host.esxui-example-plugin',
            onContext: function (e) {
               $rootScope.contextMenu.show('example', ['object'], e);
            },
            children: [{
               title: translate('example.menu.titleChildOne'),
               icon: 'icon-example-menu',
               state: 'host.esxui-example-plugin.one',
               onContext: function (e) {
                  $rootScope.contextMenu.show('example', ['object'], e);
               }
            }, {
               title: translate('example.menu.titleChildTwo'),
               icon: 'icon-example-menu',
               state: 'host.esxui-example-plugin.two',
               onContext: function (e) {
                  $rootScope.contextMenu.show('example', ['object'], e);
               }
            }]
         }],
         states: [{
            name: 'host.esxui-example-plugin',
            options: {
               url: '/example',
               views: {
                  'content@host': {
                     controller: 'ExampleMainController',
                     templateUrl: 'plugins/esxui-example-plugin/views/example-main.html'
                  }
               }
            }
         }, {
            name: 'host.esxui-example-plugin.one',
            options: {
               url: '/one'
            }
         }, {
            name: 'host.esxui-example-plugin.two',
            options: {
               url: '/two'
            }
         }]
      });
   });
});
