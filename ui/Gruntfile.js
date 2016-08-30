'use strict';

module.exports = function(grunt) {
  // Load grunt tasks automatically
  require('load-grunt-tasks')(grunt);

  // Time how long tasks take. Can help when optimizing build times
  require('time-grunt')(grunt);

  // prepare copyfiles and uglyfiles conditioned on env option

  var env = grunt.option('env');

  var copyfiles = [
    'views/**/{,*/}*.html',
    'templates/**/{,*/}*.html',
    'images/**/{,*/}*.*',
    'styles/fonts/**/{,*/}*.*',
    'i18n/**/{,*/}*.*'
  ];

  if (env === 'dev') {
    copyfiles = copyfiles.concat([
      'scripts/**/{,*/}*.*',
      'plugin.js'
    ]);
  }

  var uglyfiles = (env === 'dev') ? [] : [{
    //
    // these services should not have interdependencies
    //
    'build/dist/scripts/services/dvol-context-menu.js': 'scripts/services/dvol-context-menu.js',
    'build/dist/scripts/services/dvol-dialog-service.js': 'scripts/services/dvol-dialog-service.js',
    'build/dist/scripts/services/dvol-datacenter-vm-service.js': 'scripts/services/dvol-datacenter-vm-service.js',
    'build/dist/scripts/services/dvol-tenant-service.js': 'scripts/services/dvol-tenant-service.js',
    'build/dist/scripts/services/dvol-datastore-service.js': 'scripts/services/dvol-datastore-service.js',
    'build/dist/scripts/services/dvol-vsan-service.js': 'scripts/services/dvol-vsan-service.js',
    //
    // Grid services depend on other services
    //
    'build/dist/scripts/services/dvol-vms-grid-service.js': 'scripts/services/dvol-vms-grid-service.js',
    'build/dist/scripts/services/dvol-datastore-grid-service.js': 'scripts/services/dvol-datastore-grid-service.js',
    'build/dist/scripts/services/dvol-tenant-grid-service.js': 'scripts/services/dvol-tenant-grid-service.js'
  }, {
    expand: true,
    src: 'scripts/controllers/*.js',
    dest: 'build/dist/'
  }, {
    'build/dist/plugin.js': 'plugin.js'
  }];

  // Define the configuration for all the tasks
  grunt.initConfig({
    // Compiles Sass to CSS and generates necessary files if requested
    compass: {
      dist: {
        options: {
          sassDir: 'styles',
          cssDir: 'build/dist/styles',
          raw: 'Sass::Script::Number.precision = 10\n'
        }
      }
    },
    uglify: {
      //
      // mangle obfuscation is disabled for now
      // it causes the app to be non-functional,
      // likely causing problems for overall esxui angular dependency mgmt
      //
      options: {
        mangle: false
      },
      //
      //
      //
      dist: {
        files: uglyfiles
      }
    },
    copy: {
      dist: {
        files: [{
          expand: true,
          dest: 'build/dist',
          src: copyfiles
        }]
      }
    }
  });

  grunt.registerTask('default', [
    'compass',
    'uglify',
    'copy'
  ]);

};
