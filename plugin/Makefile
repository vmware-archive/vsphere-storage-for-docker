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

# Makefile for Docker data volume *managed* plugin construction
# This Makefile assumes it's in ./plugin in the tree and the binaries are already built
# Eventually we will throw away deb/rpm build from ../vmdk_plugin/Makefile and will use only
# managed plugin

# Allow user name for dockerhub (and first components of the plugin name) to be supplied in env
# Otherwise, grab it from current user (`whoami`)
DOCKER_HUB_USER ?= $(shell whoami)

# grab the latest commit SHA and related tag from git so we can construct plugin tag
GIT_SHA := $(shell git rev-parse --revs-only --short HEAD)
GIT_TAG := $(shell git describe --tags --abbrev=0 $(GIT_SHA))

# Allow PLUGIN_TAG to be supplied via env, or use the latest git tag.
# use EXTRA_TAG for adding suffixes like "-CI" or "-RELEASE" (otional)
PLUGIN_TAG ?= $(GIT_TAG)$(EXTRA_TAG)

# plugin name - used as a base for full plugin name and container for extracting rootfs
PLUGIN_NAME=$(DOCKER_HUB_USER)/docker-volume-vsphere


# Binaries we want to pick up from the actual build
BIN_LOC := ../build
BINARY := docker-volume-vsphere

# Tmp docker image used to construct rootfs + our binaries
TMP_IMAGE = $(PLUGIN_NAME):rootfs
# Tmp container used for exporting rootfs from it
TMP_CONTAINER := tmp

# default target
all: info clean plugin push

# unconditionally run those
.PHONY: all clean info plugin

clean:
	@echo "=== Cleaning work files, images and plugin(s)..."
	rm -rf ./rootfs
	rm -f $(BINARY)
	-docker plugin rm $(PLUGIN_NAME):$(PLUGIN_TAG)
	-docker rmi $(TMP_IMAGE)


plugin:
	@echo "== building Docker image, unpacking to ./rootfs and creating plugin..."
	cp $(BIN_LOC)/$(BINARY) .
	docker build -q -t $(TMP_IMAGE) .
	mkdir -p ./rootfs
	docker create --name $(TMP_CONTAINER) $(TMP_IMAGE)
	docker export  $(TMP_CONTAINER)  | tar -x -C ./rootfs
	docker rm -vf  $(TMP_CONTAINER)
	docker rmi $(TMP_IMAGE)
	@echo "-- Creating  plugin $(PLUGIN_NAME):$(PLUGIN_TAG) ..."
	docker plugin create $(PLUGIN_NAME):$(PLUGIN_TAG) .
	@touch $@


push:
	@echo Pushing $(PLUGIN_NAME):$(PLUGIN_TAG)  to dockerhub.io...
	@docker plugin push $(PLUGIN_NAME):$(PLUGIN_TAG) || \
			echo 'Please make sure the plugin is built ("make clean plugin") and you have logged in to docker hub first ("docker login -u $(DOCKER_HUB_USER)")'


info:
	@echo Using the following config:
	@echo DOCKER_HUB_USER $(DOCKER_HUB_USER), GIT_SHA $(GIT_SHA), GIT_TAG $(GIT_TAG)
	@echo PLUGIN_NAME $(PLUGIN_NAME):$(PLUGIN_TAG)
