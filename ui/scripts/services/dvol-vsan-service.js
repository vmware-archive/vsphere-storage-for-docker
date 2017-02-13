/* global define DEBUG */

DEBUG = true;

define(['angular'], function(angular) {
  'use strict';

  return function(
      $q, $log, $location, AuthService, StorageManager
  ) {

    var performRawSOAPRequest = function(
     type, moid, methodName, vers, args) {

      var _hostname = AuthService.getProvidedHostname();
      var _port = AuthService.getProvidedPort();
      var _csrfToken = StorageManager.get('csrf_token', null);

      var deferred = $q.defer();

      var version = vers;
      if (angular.isUndefined(version)) {
        version = '5.1';
      }

      var soapReq = '';
      soapReq += '<?xml version="1.0" encoding="UTF-8"?>';
      soapReq += '<soapenv:Envelope xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/" ';
      soapReq += 'xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" ';
      soapReq += 'xmlns:xsd="http://www.w3.org/2001/XMLSchema" ';
      soapReq += 'xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">';
      soapReq += '<soapenv:Body>';
      soapReq += '<' + methodName + ' xmlns="urn:vim25">';
      soapReq += '<_this type="' + type + '" >' + moid + '</_this>';
      if (angular.isDefined(args)) {
        soapReq += args;
      }
      soapReq += '</' + methodName + '>';
      soapReq += '</soapenv:Body>';
      soapReq += '</soapenv:Envelope>';

      var xhr = new XMLHttpRequest(),
        host = _hostname + ':' + _port,
        proxy = false;

      /* deal with proxying requests */
      if (host !== $location.host()) {
        host = $location.host() + ':' + $location.port() + '/vsan';
        proxy = true;
      } else {
        host = host + '/vsan';
      }

      xhr.open('POST', 'https://' + host, true);

      if (proxy) {
        $log.debug('using proxy ' + host);
        xhr.setRequestHeader('x-vsphere-proxy',
           'https://' + _hostname + ':' + _port + '/vsan');
      }

      xhr.setRequestHeader('Content-Type', 'text/xml; charset=utf-8');
      xhr.setRequestHeader('SOAPAction', 'urn:vim25/' + version);
      xhr.setRequestHeader('VMware-CSRF-Token', _csrfToken);

      if (DEBUG) {
        console.log(soapReq);
      }

      xhr.send(soapReq);

      xhr.onreadystatechange = function() {
        if (xhr.readyState === 4) {
          if (xhr.status === 200) {
            deferred.resolve(xhr.response);

            if (DEBUG) {
              console.log(xhr.response);
            }

          } else {
            deferred.reject();
          }
        }
      };

      return deferred.promise;

    };

    this.listTenants = function() {
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'ListTenants',
        '6.0',
        ''
      );
      return p;
    };

    this.createTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<description>' + args.description + '</description>',
        '<default_datastore>' + args.defaultDatastore + '</default_datastore>',
        '<default_privileges>' + args.defaultPrivileges + '</default_privileges>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'CreateTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.removeTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'RemoveTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.addDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<datastore>' + args.datastore + '</datastore>',
        '<rights>' + JSON.stringify(args.rights) + '</rights>',
        '<volume_maxsize>' + args.volume_maxsize + '</volume_maxsize>',
        '<volume_totalsize>' + args.volume_totalsize + '</volume_totalsize>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'AddDatastoreAccessForTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.modifyDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<datastore>' + args.datastore + '</datastore>',
        '<add_rights>' + JSON.stringify(args.add_rights) + '</add_rights>',
        '<remove_rights>' + JSON.stringify(args.remove_rights) + '</remove_rights>',
        '<volume_maxsize>' + args.volume_maxsize + '</volume_maxsize>',
        '<volume_totalsize>' + args.volume_totalsize + '</volume_totalsize>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'ModifyDatastoreAccessForTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.addVMsToTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<vms>' + JSON.stringify(args.vms) + '</vms>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'AddVMsToTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.removeVMsFromTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<vms>' + JSON.stringify(args.vms) + '</vms>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'RemoveVMsFromTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.listVMsForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'ListVMsForTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.getDatastoreAccessPrivileges = function() {
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'GetDatastoreAccessPrivileges',
        '6.0',
        ''
      );
      return p;
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
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'CreateDatastoreAccessPrivileges',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.removeDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>',
        '<datastore>' + args.datastore + '</datastore>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'RemoveDatastoreAccessForTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

    this.listDatastoreAccessForTenant = function(args) {
      var argsSOAP = [
        '<name>' + args.name + '</name>'
      ].join('');
      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'ListDatastoreAccessForTenant',
        '6.0',
        argsSOAP
      );
      return p;
    };

  };

});
