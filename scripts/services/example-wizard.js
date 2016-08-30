define(['angular'], function (angular) {
   'use strict';

   return function (
      $q,
      $log,
      $filter,
      vuiConstants) {

      $log = $log.getInstance('ExampleWizardService', true);
      $log.debug('In example wizard service global scope');

      var translate = $filter('translate');

      var testPageOne = {
         title: 'Test one',
         state: vuiConstants.wizard.pageState.INCOMPLETE,
         contentUrl: 'plugins/example/views/wizard/test-page-one.html',
         description: 'Test page description'
      };

      var testPageTwo = {
         title: 'Test two',
         state: vuiConstants.wizard.pageState.INCOMPLETE,
         contentUrl: 'plugins/example/views/wizard/test-page-two.html',
         description: 'Test page description'
      };

      this.showWizard = function (wizard) {
         switch(wizard) {
            case 'example.wizard':
               return {
                  show: true,
                  title: 'Title',
                  iconClass: 'esx-icon-new-vm',
                  data: {
                     testPageOne: testPageOne,
                     testPageTwo: testPageTwo
                  },
                  pages: [
                     testPageOne,
                     testPageTwo
                  ]
               };
               break;
         }
      };
   };
});
