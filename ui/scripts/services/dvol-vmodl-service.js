/* global define $ */

define([], function() {
  'use strict';

  return function(DvolSoapService) {

    function parsePrivileges(privilegesEl) {
      var privileges = {};
      ['datastore',
      'create_volumes',
      'delete_volumes',
      'mount_volumes',
      'max_volume_size',
      'usage_quota']
      .forEach(function(prop) {
        privileges[prop] = $($(privilegesEl).find(prop)[0]).text();
      });
      return privileges;
    }

    function parseTenant(tenantEl) {
      var tenant = {};
      ['name',
      'description',
      'default_datastore']
      .forEach(function(prop) {
        tenant[prop] = $($(tenantEl).find(prop)[0]).text();
      });
      tenant.vms = $(tenantEl).find('vms').toArray().map(function(vmEl) {
        return $(vmEl).text();
      });
      tenant.default_privileges = parsePrivileges($(tenantEl).find('default_privileges')[0]);
      return tenant;
    }

    function parseVm(vmEl) {
      return $(vmEl).text();
    }

    function parseDatastore(datastoreEl) {
      var ds = {};
      ds.datastore = $($(datastoreEl).find('datastore')[0]).text();
      ds.permissions = {};
      [
        'create_volumes',
        'delete_volumes',
        'mount_volumes',
        'max_volume_size',
        'usage_quota'
      ]
      .forEach(function(prop) {
        ds.permissions[prop] = $($(datastoreEl).find(prop)[0]).text();
      });
      return ds;
    }

    this.listTenants = function() {
      return DvolSoapService.request('ListTenants')
      .then(function(soapResponse) {
        var doc = $.parseXML(soapResponse);
        var listTenantsResponse = $(doc).find('ListTenantsResponse');
        var tenantEls = $(listTenantsResponse).find('returnval');
        var tenants = tenantEls.toArray().map(parseTenant);
        return tenants;
      });
    };

    this.createTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<description>' + args.description + '</description>',
        '<default_datastore>' + args.default_datastore + '</default_datastore>',
        '<default_privileges>' + args.default_privileges + '</default_privileges>'
      ].join('');
      return DvolSoapService.request('CreateTenant', argsSOAP);
    };

    //
    // modifyTenant not yet supported by the VMODL API
    //

    // this.modifyTenant = function(args) {
    //   var argsSOAP = [
    //     '<name>' + args.name + '</name>',
    //     '<description>' + args.description + '</description>',
    //     '<default_datastore>' + args.default_datastore + '</default_datastore>',
    //     '<default_privileges>' + args.default_privileges + '</default_privileges>'
    //   ].join('');
    //   return DvolSoapService.request('ModifyTenant', argsSOAP);
    // };

    this.removeTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>'
      ].join('');
      return DvolSoapService.request('RemoveTenant', argsSOAP);
    };

    this.addDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<datastore>' + args.datastore + '</datastore>',
        '<rights>' + JSON.stringify(args.rights) + '</rights>',
        '<volume_maxsize>' + args.volume_maxsize + '</volume_maxsize>',
        '<volume_totalsize>' + args.volume_totalsize + '</volume_totalsize>'
      ].join('');
      return DvolSoapService.request('AddDatastoreAccessForTenant', argsSOAP);
    };

    this.modifyDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<datastore>' + args.datastore + '</datastore>',
        '<add_rights>' + JSON.stringify(args.add_rights) + '</add_rights>',
        '<remove_rights>' + JSON.stringify(args.remove_rights) + '</remove_rights>',
        '<max_volume_size>' + args.max_volume_size + '</max_volume_size>',
        '<usage_quota>' + args.usage_quota + '</usage_quota>'
      ].join('');
      return DvolSoapService.request('ModifyDatastoreAccessForTenant', argsSOAP);
    };

    this.addVMsToTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<vms>' + JSON.stringify(args.vms) + '</vms>'
      ].join('');
      return DvolSoapService.request('AddVMsToTenant', argsSOAP);
    };

    this.removeVMsFromTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<vms>' + JSON.stringify(args.vms) + '</vms>'
      ].join('');
      return DvolSoapService.request('RemoveVMsFromTenant', argsSOAP);
    };

    this.listVmsForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>'
      ].join('');
      return DvolSoapService.request('ListVMsForTenant', argsSOAP)
      .then(function(soapResponse) {
        var doc = $.parseXML(soapResponse);
        var listVmsResponse = $(doc).find('ListVMsForTenantResponse');
        var vmEls = $(listVmsResponse).find('returnval');
        var vms = vmEls.toArray().map(parseVm);
        return vms;
      });
    };

    this.getDatastoreAccessPrivileges = function() {
      return DvolSoapService.request('GetDatastoreAccessPrivileges');
    };

    this.createDatastoreAccessPrivileges = function(args) {
      var argsSOAP = [
        '<datastore>' + args.datastore + '</datastore>',
        '<create_volumes>' + args.create_volumes + '</create_volumes>',
        '<delete_volumes>' + args.delete_volumes + '</delete_volumes>',
        '<mount_volumes>' + args.mount_volumes + '</mount_volumes>',
        '<max_volume_size>' + args.max_volume_size + '</max_volume_size>',
        '<usage_quota>' + args.usage_quota + '</usage_quota>'
      ].join('');
      return DvolSoapService.request('CreateDatastoreAccessPrivileges', argsSOAP);
    };

    this.removeDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<datastore>' + args.datastore + '</datastore>'
      ].join('');
      return DvolSoapService.request('RemoveDatastoreAccessForTenant', argsSOAP);
    };

    this.listDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>'
      ].join('');
      return DvolSoapService.request('ListDatastoreAccessForTenant', argsSOAP)
      .then(function(soapResponse) {
        var doc = $.parseXML(soapResponse);
        var listDatastoresResponse = $(doc).find('ListDatastoreAccessForTenantResponse');
        var datastoreEls = $(listDatastoresResponse).find('returnval');
        var datastores = datastoreEls.toArray().map(parseDatastore);
        return datastores;
      });
    };

  };

});
