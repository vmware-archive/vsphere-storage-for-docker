/* global define */

define([], function() {
  'use strict';

  return function(StorageService) {

    function get() {
      return StorageService.getDatastores();
    }

    this.get = get;

  };

});
