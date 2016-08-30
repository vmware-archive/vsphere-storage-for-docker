"""
ESX UI component spec.
"""

VIBSUITE_BRANCH = 'esx50u1'
VIBSUITE_CLN = 1547037
VIBSUITE_BUILDTYPE = 'release'
VIBSUITE_HOSTTYPES = {
    'linux-centos64-64': 'linux64'
}

VUI_COMPONENTS_BRANCH = 'host-client'
VUI_COMPONENTS_CLN = '13834c17f8acb26c180fdb93fc96e9e307c9e2b8'
VUI_COMPONENTS_FILES = {
    'linux-centos64-64': [
       r'publish/vui-angular.zip',
       r'publish/vui-bootstrap.zip',
    ],
}
VUI_COMPONENTS_BUILDTYPE = 'release'

WMKSSDK_BRANCH = 'crt-main'
WMKSSDK_CLN = 181908
WMKSSDK_BUILDTYPE = 'release'
WMKSSDK_HOSTTYPES = {
    'linux-centos64-64': 'linux64',
    'windows-2008': 'linux64'
 }

CAYMAN_NODEJS_BRANCH = 'vmware-node-v4.x'
CAYMAN_NODEJS_CLN = 'b2c07af88d919353a55f1a314e0544c5b515541b'
CAYMAN_NODEJS_BUILDTYPE = 'release'
CAYMAN_NODEJS_HOSTTYPES = {
    'linux-centos64-64': 'linux64',
    'windows-2008': 'linux64'
}
