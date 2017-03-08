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

# User component of plugin name (and dockerhub group to push to)
USER := cnastorage

PLUGIN_NAME=$(USER)/docker-volume-vsphere
PLUGIN_TAG=0.12

# Binaries we want to pick up
BINARY := docker-volume-vsphere
BIN_LOC := ../build

# Tmp docker image used to constuct rootfs + our binaries
TMP_IMAGE = $(PLUGIN_NAME):rootfs
# temp container used for exporting rootfs from it
TMP_CONTAINER := tmp

all: clean plugin push

clean:
	@echo "=== Cleaning work files, images and plugin(s)..."
	rm -rf ./rootfs
	rm -f $(BINARY)
	-docker plugin rm $(PLUGIN_NAME):$(PLUGIN_TAG)
	-docker rmi $(TMP_IMAGE)

plugin: clean
	@echo "== building Docker image, unpacking to ./rootfs and creating plugin..."
	cp $(BIN_LOC)/$(BINARY) .
	docker build -q -t $(TMP_IMAGE) .
	mkdir -p ./rootfs
	docker create --name $(TMP_CONTAINER) $(TMP_IMAGE)
	docker export  $(TMP_CONTAINER)  | tar -x -C ./rootfs
	docker rm -vf  $(TMP_CONTAINER)
	@echo "-- Creating  plugin $(PLUGIN_NAME):$(PLUGIN_TAG) ..."
	docker plugin create $(PLUGIN_NAME):$(PLUGIN_TAG) .


push:
	docker plugin push $(PLUGIN_NAME):$(PLUGIN_TAG)
