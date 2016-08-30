# Copyright 2016 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This Makefile has 2 purposes:
#   - convenience: redirects targets to ./vmdk_plugin so 'make'
#                  can be run from the top
#   - drone invariant: defines deb/rpm/build/testasroot targets so drone.yml
#                   can be kept as-is

TARGETS := all \
	deploy-all deploy-esx deploy-vm deploy-vm-test deploy\
	test-all test-vm test-esx testremote testasroot\
	clean-all clean clean-vm clean-esx clean-ui \
	build-all build dockerbuild build-ui \
	deb rpm package gvt documentation

# default target
default: build-all

# redirect all
$(TARGETS):
	$(MAKE) --directory=ui $@
	$(MAKE) --directory=vmdk_plugin $@
