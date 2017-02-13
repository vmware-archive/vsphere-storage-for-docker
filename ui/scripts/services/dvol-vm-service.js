/* global define */

define([], function() {
  'use strict';

  return function(VMService) {

    function get() {
      return VMService.getVMsForList(true);
    }

    this.get = get;

  };

});
