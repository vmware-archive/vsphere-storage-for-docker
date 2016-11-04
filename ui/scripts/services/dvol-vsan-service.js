/* global define */

define(['angular'], function(angular) {
  'use strict';

  return function(
      $rootScope, $q, $log, $location, $interval, $filter, $timeout, $sce, $window,
      VIMService, TaskService, StorageService, NotificationService, AuthService, StorageManager
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
      xhr.send(soapReq);

      xhr.onreadystatechange = function() {
        if (xhr.readyState === 4) {
          if (xhr.status === 200) {
            deferred.resolve(xhr.response);
          } else {
            deferred.reject();
          }
        }
      };

      return deferred.promise;

    };

    this.getTenants = function() {

      var p = performRawSOAPRequest(
        'VimHostVsanDockerPersistentVolumeSystem',
        'vsan-docker-persistent-volumes',
        'GetTenantList',
        '6.0',
        ''
      );

      return p;

    };

  };

});
