/* global define */

define([], function() {
  'use strict';

  return function(StorageService) {

    //
    //  MAYBE this: GetDatastoreAccessPrivileges
    //  NOTE: need to clarify
    //

    function get() {
      return StorageService.getDatastores();
    }

    this.get = get;

  };

});
