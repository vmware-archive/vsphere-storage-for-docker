/* global define */

define([], function() {
  'use strict';

  return function($q) {

    var dvolContextMenu = [{
      title: 'Some menu item',
      toolTip: 'Menu item tooltip',
      id: 'menuItemID',
      iconClass: 'esx-icon-example',
      enabled: true,
      children: [{
        title: 'Some child menu item',
        toolTip: 'Menu item tooltip',
        id: 'childMenuItemID',
        iconClass: 'esx-icon-example',
        enabled: true
      }]
    }];

    this.reconcile = function(context, objects, highlightPath) {

      var deferred = $q.defer();

      var traverse = function(menu, opaque) {
        menu.forEach(function(menuItem) {
          if (menuItem.update) {
            menuItem.update(opaque);
          }

          if (menuItem.state &&
            highlightPath &&
            highlightPath.indexOf(menuItem.state) !== -1) {
            menuItem.highlight = true;
          } else {
            menuItem.highlight = false;
          }

          if (menuItem.children) {
            traverse(menuItem.children, opaque);
          }
        });

        return menu;
      };

      switch (context) {
      case 'storage':
        /* falls through */
      case 'storage.adapter':
        /* falls through */
      case 'storage.datastore':
        /* falls through */
      case 'storage.device.disk':
        /* falls through */
      case 'storage.device.cdrom':
        /* falls through */
      case 'networking':
        /* falls through */
      case 'network.portgroup':
        /* falls through */
      case 'network.vswitch':
        /* falls through */
      case 'network.vmknic':
        /* falls through */
      case 'network.pnic':
        /* falls through */
      case 'client.help':
        /* falls through */
      case 'client.user':
        /* falls through */
      case 'host':
        /* falls through */
      case 'vm':
        /* falls through */
      case 'vm.none':
        /* falls through */
      case 'dvol':
        deferred.resolve({
          menu: traverse(dvolContextMenu, objects),
          title: 'Context title',
          iconClass: 'esx-icon-example'
        });
        break;

      default:
        deferred.resolve();
        break;
      }

      return deferred.promise;
    };
  };
});
