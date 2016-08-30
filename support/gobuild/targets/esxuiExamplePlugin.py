# Copyright 2016 VMware, Inc.  All rights reserved. -- VMware Confidential
"""
ESXUI Example Plugin gobuild product module.
"""

import helpers.target
import helpers.env
import helpers.make
import specs.esxuiExamplePlugin as specs

# This flag will be controlled by gobuild command
ALLOW_OFFICIAL_KEY = 'allow.elfofficialkey'

class EsxUIExamplePlugin(helpers.target.Target, helpers.make.MakeHelper):
   """
   Embedded Host Client Example Plugin
   """

   def GetBuildProductNames(self):
      return {
          'name': 'esxui-example-plugin',
          'longname' : 'Embedded Host Client example plugin'
      }

   def GetClusterRequirements(self):
      return ['linux-centos64-64']

   def GetRepositories(self, hosttype):
      return [{
         'rcs': 'git',
         'src': 'esxui-example-plugin.git;%(branch);',
         'dst': 'esxui-example-plugin',
      }]

   def GetOptions(self):
      return [
         (ALLOW_OFFICIAL_KEY, False,
          ("Whether to allow official key to be turned on. This flag "
           "will be turned on in official build and off in sandbox and "
           "continuous build.")),
      ]

   def GetCommands(self, hosttype):
      flags = {
         'PRODUCT': 'esxui-example-plugin',
         'USE_OFFICIAL_KEY': self.options[ALLOW_OFFICIAL_KEY]
      }

      # Root directory, from where make command is to be executed.
      root = '%(buildroot)/esxui'

      # Environment settings under where make command is to be executed.
      env = helpers.env.SafeEnvironment(hosttype)

      return [{
          'desc'    : 'Building Embedded Host Client example plugin',
          'root'    : root,
          'log'     : 'build.log',
          'command' : self._Command(hosttype, 'all', **flags),
          'env'     : env,
      }, {
          'desc'    : 'Publishing Embedded Host Client example plugin',
          'root'    : root,
          'log'     : 'package.log',
          'command' : self._Command(hosttype, 'publish', **flags),
          'env'     : env,
      }]

   def GetComponentPath(self, hosttype):
      return 'esxui-example-plugin/build/'

   def GetStorageInfo(self, hosttype):
      storinfo = []
      storinfo += [{'type': 'build', 'src': 'esxui-example-plugin/build'}]
      storinfo += [{'type': 'source', 'src': 'esxui-example-plugin/'}]
      return storinfo

   def GetBuildProductVersion(self, hosttype):
      return "1.0.0"

   def GetComponentDependencies(self):
      comps = {}

      comps['cayman_nodejs'] = {
         'branch'    : specs.CAYMAN_NODEJS_BRANCH,
         'change'    : specs.CAYMAN_NODEJS_CLN,
         'buildtype' : specs.CAYMAN_NODEJS_BUILDTYPE,
         'hosttypes' : specs.CAYMAN_NODEJS_HOSTTYPES
      }

      comps['vibsuite'] = {
         'branch'    : specs.VIBSUITE_BRANCH,
         'change'    : specs.VIBSUITE_CLN,
         'buildtype' : specs.VIBSUITE_BUILDTYPE,
         'hosttypes' : specs.VIBSUITE_HOSTTYPES
      }

      return comps
