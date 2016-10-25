
/* global define $ */

define([], function() {
  'use strict';

  return function($scope, DialogService) {

    $scope.permissions = DialogService.currentDialog().opaque.permissions;

    DialogService.setConfirmOptions({
      label: 'Save',
      onClick: function() {
        DialogService.currentDialog().opaque.save($scope.permissions);
        return true;
      }
    });

  };

});
