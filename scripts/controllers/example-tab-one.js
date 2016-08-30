define(['angular'], function(angular) {
   'use strict';

   return function(
      $log, $rootScope, $scope,
      ExampleService, ExampleContextMenuService) {

      $log = $log.getInstance('ExampleTabOneController', true);
      $log.debug('In example tab one controller global scope');

      var actionButton = {
         id: 'exampleButton',
         label: 'Test button one',
         tooltipText: 'Test tooltip',
         enabled: true,
         iconClass: 'esx-icon-example',
         onClick: function (e) {
            var button = $(e.currentTarget);
            var pos = button.offset();
            var height = button.height();
            $rootScope.contextMenu.show('example', ['object'], e,
               pos.left - 1, pos.top + height + 4);
         }
      };

      $scope.actionBar = {
         actions: [
            actionButton
         ]
      };
   };
});
