define(['angular'], function (angular) {
   'use strict';

   return function ($log) {
      $log = $log.getInstance('ExampleService', true);
      $log.debug('In example service global scope');

      this.serviceMethod = function () {
         $log.debug('serviceMethod was called');
      };
   };
});
