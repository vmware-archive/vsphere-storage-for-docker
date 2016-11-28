/* global define */

define([], function() {
  'use strict';

  function formatCapacity(rawCap) {
    return String(rawCap * Math.pow(10, -9)).substr(0, 5) + ' GB';
  }

  return function(DvolDatastoreService, GridUtils, vuiConstants, StorageUtil) {

    function mapDatastoresToGrid(datastores) {
      return datastores.map(function(ds) {
        var datastore = ds;
        var permissions;
        if (ds.datastore) {
          permissions = datastore.permissions;
          datastore = datastore.datastore;
        }
        var capacity = formatCapacity(datastore.capacity);
        var freeSpace = formatCapacity(datastore.freeSpace);
        var colData = {
          id: datastore.moid,
          moid: datastore.moid,
          name: datastore.name,
          driveType: datastore.driveType,
          capacity: capacity,
          freeSpace: freeSpace,
          type: datastore.type
        };
        if (permissions) {
          Object.keys(permissions).forEach(function(k) {
            colData[k] = permissions[k];
          });
        }
        return colData;
      });
    }

    function makeColumnDefs(grid) {
      return [
        {
          displayName: 'Datastore',
          field: 'name',
          width: '30%',
          autoHighlight: false,
          template: function(dataItem) {
            var moid = StorageUtil.urlEncodeMOID(dataItem.moid);
            var href = '#/host/storage/datastores/' + moid;
            var content = dataItem.name;
            if (grid && null) {
              return GridUtils.contextMenuAction(grid,
               content, StorageUtil.getIcon(dataItem),
               'storage.datastore', [dataItem.moid], null, href);
            }
            var icon = 'esx-icon-datastore';
            return '<div data-moid="' + dataItem.moid + '">' +
              '<i class="' + icon + '"></i>' + dataItem.name + '</div>';
          }
        },
        {
          field: 'driveType',
          displayName: 'Drive Type'
        },
        {
          field: 'type',
          displayName: 'Type'
        },
        {
          field: 'capacity',
          displayName: 'Capacity',
          visible: false
        },
        {
          field: 'freeSpace',
          displayName: 'Free',
          visible: false
        },
        {
          field: 'id',
          displayName: 'ID',
          visible: false
        }
      ];
    }

    var permColumnDefs = [
      {
        field: 'create_volumes',
        displayName: 'Create Volumes'
      },
      {
        field: 'mount_volumes',
        displayName: 'Mount Volumes'
      },
      {
        field: 'delete_volumes',
        displayName: 'Delete Volumes'
      },
      {
        field: 'max_volume_size',
        displayName: 'Max Volume'
      },
      {
        field: 'usage_quota',
        displayName: 'Usage Quota'
      }
    ];

    function makeDatastoresGrid(id, actions, filterFn, perms) {

      var datastoresGrid;

      var showPermissions = perms;

      var actionBarOptions = {
        actions: actions
      };

      datastoresGrid = GridUtils.Grid({
        id: id,
        columnDefs: makeColumnDefs().concat(showPermissions ? permColumnDefs : []),
        actionBarOptions: actionBarOptions,
        selectionMode: vuiConstants.grid.selectionMode.SINGLE,
        selectedItems: [],
        data: mapDatastoresToGrid([])
      });

      datastoresGrid.search = null;

      function refresh() {
        var p;
        if (filterFn) {
          p = DvolDatastoreService.get()
          .then(filterFn)
          .then(function(filteredDatastores) {
            datastoresGrid.data = mapDatastoresToGrid(filteredDatastores);
          });
        } else {
          p = DvolDatastoreService.get()
          .then(function(datastores) {
            datastoresGrid.data = mapDatastoresToGrid(datastores);
          });
        }
        return p;
      }

      refresh();

      return {
        grid: datastoresGrid,
        refresh: refresh
      };

    }

    this.makeDatastoresGrid = makeDatastoresGrid;

  };

});
