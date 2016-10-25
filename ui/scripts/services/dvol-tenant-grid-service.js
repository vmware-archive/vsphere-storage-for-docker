/* global define angular */

define([], function() {
  'use strict';

  return function(DvolTenantService, GridUtils, vuiConstants) {

    function mapTenantsToGrid(tenants) {
      return tenants.map(function(tenant) {
        return tenant;
      });
    }

    var columnDefs = [
      {
        field: 'name',
        displayName: 'Name',
        width: '30%',
        template: function(dataItem) {
          var icon = 'esx-icon-datastore-register';
          var displayName = dataItem.name || '';
          return '<div>' + '<i class="' + icon + '"></i>' + displayName || '' + '</div>';
        }
      },
      {
        field: 'description',
        displayName: 'Description',
        width: '50%',
        template: function(dataItem) {
          if (angular.isDefined(dataItem.description)) {
            return dataItem.description;
          }
          return '';
        }
      },
      {
        width: '20%',
        field: 'id',
        displayName: 'ID'
      }
    ];


    var gridProps = {
      id: 'tenantsGrid',
      idDataField: 'id',
      columnDefs: columnDefs,
      sortMode: vuiConstants.grid.sortMode.SINGLE,
      selectionMode: vuiConstants.grid.selectionMode.SINGLE,
      selectedItems: [],
      data: mapTenantsToGrid([]),
      searchable: true
    };

    function makeTenantsGrid(actions) {

      if (actions) {
        gridProps.actionBarOptions = gridProps.actionBarOptions || {};
        gridProps.actionBarOptions.actions = actions;
      }

      var tenantsGrid = GridUtils.Grid(gridProps);

      function refresh() {
        return DvolTenantService.getAll().then(function(tenants) {
          tenantsGrid.data = mapTenantsToGrid(tenants);
        });
      }

      refresh();

      return {
        grid: tenantsGrid,
        refresh: refresh
      };

    }

    this.makeTenantsGrid = makeTenantsGrid;

  };

});
