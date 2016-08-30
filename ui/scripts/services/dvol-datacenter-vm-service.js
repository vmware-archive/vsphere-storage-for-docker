/* global define */

define([], function() {
  'use strict';

  return function(VMService) {

    function get() {
      var progress = function(percent) {
        console.log('progress update: ' + percent + ' %');
      };
      return VMService.getVMsForList(true, progress);
    }

    this.get = get;

  };

});
