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
	clean-all clean-vm clean-esx \
	dockerbuild \
	deb rpm package gvt documentation

BUILD := ./misc/scripts/build.sh

#
# kill switch for UI
# To override default here export env variable before make
# export INCLUDE_UI=true
INCLUDE_UI ?= false

export INCLUDE_UI

# default target, build ui then build vib, rpm, deb
default: build-all

build-all: dockerized-build-ui
	$(MAKE) --directory=vmdk_plugin $@
	$(MAKE) --directory=plugin all

build-global: dockerized-build-ui
	$(MAKE) --directory=global_plugin build-all
	$(MAKE) --directory=global_plugin_docker all

# clean inside docker run to avoid sudo make clean
# dev builds are inside docker which creates folders
# as root.
clean: dockerized-clean-ui
	$(MAKE) --directory=vmdk_plugin $@
	$(MAKE) --directory=global_plugin $@

# Non dockerized build, used by CI
build:
ifeq ($(INCLUDE_UI), true)
	$(MAKE) --directory=ui $@
endif
	$(MAKE) --directory=vmdk_plugin $@
	$(MAKE) --directory=global_plugin $@

# Forward to UI inside docker run
dockerized-build-ui:
ifeq ($(INCLUDE_UI), true)
	$(BUILD) ui build
endif

dockerized-clean-ui:
ifeq ($(INCLUDE_UI), true)
	$(BUILD) ui clean
endif

# redirect all
$(TARGETS):
	$(MAKE) --directory=vmdk_plugin $@

# if we do not recognize the target, just pass it on to vmdk_plugin Makefile
.DEFAULT:
	$(MAKE) --directory=vmdk_plugin $@
