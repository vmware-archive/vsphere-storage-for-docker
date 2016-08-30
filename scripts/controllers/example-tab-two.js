define(['angular'], function(angular) {
   'use strict';

   return function($log, $scope, ExampleService, DialogService, WizardService) {
      $log = $log.getInstance('ExampleTabTwoController', true);
      $log.debug('In example tab two controller global scope');

      var testButtonTwo = {
         id: 'testButtonTwo',
         label: 'Test button two',
         tooltipText: 'Test tooltip',
         enabled: true,
         iconClass: 'esx-icon-vm-console',
         onClick: function () {
            DialogService.showDialog('example.about', {
               test: 'test'
            });
         }
      };

      var testButtonThree = {
         id: 'testButtonThree',
         label: 'Test button three',
         tooltipText: 'Test tooltip',
         enabled: true,
         iconClass: 'esx-icon-new-vm',
         onClick: function () {
            WizardService.showWizard('example.wizard');
         }
      };

      $scope.actionBar = {
         actions: [
            testButtonTwo,
            testButtonThree
         ]
      };
   };
});
