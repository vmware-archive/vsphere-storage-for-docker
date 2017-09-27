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

#
# Makefile for Docker data volume *managed* plugin.
#
# This Makefile assumes it's in ./plugin in the tree and the binaries are already built
# Eventually we will throw away deb/rpm build from ../client_plugin/Makefile and will use only
# managed plugin

#
# Makefile names the plugin simlarly to this (for example) 'msterin/docker-volume-vsphere:0.13-dev'
#
# Make targets:
#    info (helpful info), clean (remove build artefacts), plugin, (build plugin), push (push to dockerhub)
#

# Exporting common constants
-include ../Commonvars.mk

# Binaries we want to pick up from the actual build
BINARY ?= docker-volume-vsphere
VFILE_BINARY := vfile

# Dockerfile templates
DOCKERFILE ?= Dockerfile.vmdk
VFILE_DOCKERFILE := Dockerfile.vfile

# Network Type
NETWORK ?=
VFILE_NETWORK := host

# Sock Type
SOCKTYPE ?= vsphere
VFILE_SOCKTYPE := vfile

# Mount Point
MOUNTPOINT ?= vmdk
VFILE_MOUNTPOINT := vfile

# Name
NAME ?= vSphere
VFILE_NAME := vFile

# Version
VDVS_VERSION = $(PLUGIN_TAG)

# Tmp docker image used to construct rootfs + our binaries
TMP_IMAGE = $(PLUGIN_NAME):rootfs
# Tmp container used for exporting rootfs from it
TMP_CONTAINER := tempContainer
# Tmp location on local FS to manipulate plugin files
TMP_LOC := /tmp/plugin

# default target
all: info plugin push
vfile-all: vfile-info vfile-plugin vfile-push

# unconditionally run those
.PHONY: all clean info plugin vfile-all vfile-info vfile-plugin vfile-push

info:
	@echo Using the following config:
	@echo DOCKER_HUB_REPO $(DOCKER_HUB_REPO) EXTRA_TAG $(EXTRA_TAG) VERSION_TAG $(VERSION_TAG)
	@echo PLUGIN_NAME $(PLUGIN_NAME):$(PLUGIN_TAG)
vfile-info:
	PLUGIN_NAME=$(VFILE_PLUGIN_NAME) $(MAKE) info

clean:
	@echo "=== Cleaning work files, images and plugin(s)..."
	rm -rf $(TMP_LOC)
	rm -f $(BINARY)
	rm -f Dockerfile
	rm -f config.json
	-docker plugin rm $(PLUGIN_NAME):$(PLUGIN_TAG) -f
	-docker rm -vf $(TMP_CONTAINER)
	-docker rmi $(TMP_IMAGE)
vfile-clean:
	PLUGIN_NAME=$(VFILE_PLUGIN_NAME) BINARY=$(VFILE_BINARY) $(MAKE) clean

plugin: clean
	@echo "== building Docker image, unpacking to ./rootfs and creating plugin..."
	cp $(DOCKERFILE) Dockerfile
	sed 's/\%NAME\%/$(NAME)/g;s/\%BINARY\%/$(BINARY)/g;s/\%NETWORKTYPE\%/$(NETWORK)/g;s/\%SOCKTYPE\%/$(SOCKTYPE)/g;s/\%MOUNTPOINT\%/$(MOUNTPOINT)/g;s/\%VDVS_VERSION\%/$(VDVS_VERSION)/g' config.json-template > config.json
	cp $(BIN)/$(BINARY) .
	docker build -q -t $(TMP_IMAGE) .
	docker create --name $(TMP_CONTAINER) $(TMP_IMAGE)
	mkdir -p $(TMP_LOC)/rootfs
	docker export  $(TMP_CONTAINER)  | tar -xm -C $(TMP_LOC)/rootfs
	cp config.json $(BINARY) $(TMP_LOC)
	@echo "-- Creating  plugin $(PLUGIN_NAME):$(PLUGIN_TAG) ..."
	docker plugin create $(PLUGIN_NAME):$(PLUGIN_TAG) $(TMP_LOC)
vfile-plugin: vfile-clean
	NAME=$(VFILE_NAME) PLUGIN_NAME=$(VFILE_PLUGIN_NAME) BINARY=$(VFILE_BINARY) DOCKERFILE=$(VFILE_DOCKERFILE) NETWORK=$(VFILE_NETWORK) SOCKTYPE=$(VFILE_SOCKTYPE) MOUNTPOINT=$(VFILE_MOUNTPOINT) $(MAKE) plugin

push:
	@echo Pushing $(PLUGIN_NAME):$(PLUGIN_TAG)  to dockerhub.io...
	@docker plugin push $(PLUGIN_NAME):$(PLUGIN_TAG) || \
			echo 'Please make sure the plugin is built ("make clean plugin") and you have logged in to docker hub first ("docker login -u $(DOCKER_HUB_REPO)")'
vfile-push:
	PLUGIN_NAME=$(VFILE_PLUGIN_NAME) $(MAKE) push
