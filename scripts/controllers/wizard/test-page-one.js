define(['angular'], function(angular) {
   'use strict';

   return function($log, $scope, vuiConstants) {
      $log = $log.getInstance('ExampleWizardTestPageOneController', true);
      $log.debug('In example test wizard controller one global scope');

      $scope.wizardOptions.currentPage.state = vuiConstants.wizard.pageState.INCOMPLETE;

      $scope.wizardOptions.currentPage.onCommit = function () {
         $log.debug('commit');
         $scope.wizardOptions.currentPage.state = vuiConstants.wizard.pageState.COMPLETED;
         return true;
      };
   };
});
