'use strict';

module.exports = function (grunt) {
   // Load grunt tasks automatically
   require('load-grunt-tasks')(grunt);

   // Time how long tasks take. Can help when optimizing build times
   require('time-grunt')(grunt);

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
         dist: {
           files: {
             'build/dist/scripts/services/example.js': ['scripts/services/example.js'],
             'build/dist/scripts/services/example-dialog.js': ['scripts/services/example-dialog.js'],
             'build/dist/scripts/services/example-wizard.js': ['scripts/services/example-wizard.js'],
             'build/dist/scripts/services/example-context-menu.js': ['scripts/services/example-context-menu.js'],
             'build/dist/scripts/controllers/example-main.js': ['scripts/controllers/example-main.js'],
             'build/dist/scripts/controllers/example-tab-one.js': ['scripts/controllers/example-tab-one.js'],
             'build/dist/scripts/controllers/example-tab-two.js': ['scripts/controllers/example-tab-two.js'],
             'build/dist/scripts/controllers/wizard/test-page-one.js': ['scripts/controllers/wizard/test-page-one.js'],
             'build/dist/scripts/controllers/wizard/test-page-two.js': ['scripts/controllers/wizard/test-page-two.js'],
             'build/dist/plugin.js': ['plugin.js']
           }
         }
      },
      copy: {
         dist: {
            files: [{
               expand: true,
               dest: 'build/dist',
               src: [
                  'views/**/{,*/}*.html',
                  'templates/**/{,*/}*.html',
                  'images/**/{,*/}*.*',
                  'styles/fonts/**/{,*/}*.*',
                  'i18n/**/{,*/}*.*'
               ]
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
