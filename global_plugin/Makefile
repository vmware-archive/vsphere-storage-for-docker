# Copyright 2017 VMware, Inc. All Rights Reserved.
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


# Makefile for Docker data volume plugin

# Guest Package Version

# PKG_VERSION is either set externally as part of a build off a tag or
# suffixed with a sha1 of the most recent commit to the prefix of the
# most recent tag. Tagged builds use the externally defined version,
# developer builds use last tagged release and sha1 of the most recent commit.
# Format: <last tagged release>.<last commit hash>
PKG_VERSION ?= $(shell \
	       git describe --tags `git rev-list --tags --max-count=1` \
	       ).$(shell \
	       git log --pretty=format:'%h' -n 1)

export PKG_VERSION
export INCLUDE_UI

-include ../global_plugin_docker/Makefile

EPOCH := 0

# Place binaries here
BIN := ../build
MANAGED_PLUGIN := ../plugin

# Location for scripts
SCRIPTS     := ../misc/scripts

# Packaging variables
BASEPLUGNAME := docker-volume-vsphere
PLUGNAME  := docker-volume-global
GOPATH_PLUGNAME := $(BASEPLUGNAME)/global_plugin
GOPATH_ORG :=vmware
MAINTAINERS := cna-storage@vmware.com
REPO_URL    := https://github.com/$(GOPATH_ORG)/$(BASEPLUGNAME)
MIN_DOCKER_VERSION :=1.9
DOCKER_PACKAGE := docker-engine

#
# Scripts to deploy and control services - used from Makefile and from Drone CI
#
CHECK	    := $(SCRIPTS)/check.sh
BUILD       := $(SCRIPTS)/build.sh

#  binaries location
PLUGIN_BIN = $(BIN)/$(PLUGNAME)

# all binaries for VMs - plugin and tests
#VM_BINS = $(PLUGIN_BIN) $(BIN)/$(PLUGNAME).test
VM_BINS = $(PLUGIN_BIN)

# plugin name, for go build
PLUGIN := github.com/$(GOPATH_ORG)/$(GOPATH_PLUGNAME)

GO := GO15VENDOREXPERIMENT=1 go

# All sources. We rebuild if anything changes here
SRC = main.go ../vmdk_plugin/utils/refcount/refcnt.go \
	../vmdk_plugin/utils/config/config.go \
	../vmdk_plugin/utils/log_formatter/log_formatter.go \
	drivers/global/global_driver.go

TEST_SRC = ../tests/utils/inputparams/testparams.go

# Canned recipe
define log_target
@echo
@echo "=> Running target $@" `date`
@echo
endef

# The default build is using a prebuilt docker image that has all dependencies.
.PHONY: dockerbuild build-all
build-all: dockerbuild

dockerbuild:
	@$(CHECK) dockerbuild
	$(BUILD) global

# The non docker build.
.PHONY: build
build: prereqs .code_verify $(VM_BINS)

.PHONY: gvt
gvt:
	$(BUILD) gvt

.PHONY: documentation
documentation:
	$(BUILD) documentation

.PHONY: prereqs
prereqs:
	@$(CHECK) global

$(PLUGIN_BIN): $(SRC)
	@-mkdir -p $(BIN) && chmod a+w $(BIN)
	$(GO) build --ldflags '-extldflags "-static"' -o $(PLUGIN_BIN) $(PLUGIN)

(BIN)/$(PLUGNAME).test: $(SRC) $(SRC) $(TEST_SRC) *_test.go
	$(GO) test -c -o $@ $(PLUGIN) -cover

.PHONY: clean
clean:
	$(BUILD) clean-as-root

.PHONY: clean-as-root
clean-as-root: pkg-post
	rm -rf $(BIN) .code_verify

# GO Code quality checks.

DIRS_TO_VERIFY := . drivers/global

FILES_TO_VERIFY := $(foreach dir, $(DIRS_TO_VERIFY), $(dir)/*.go)
.code_verify: $(FILES_TO_VERIFY)
	@for i in $(DIRS_TO_VERIFY) ; do ${GOPATH}/bin/golint $$i ; done
	go tool vet .
	gofmt -s -l -w .
	@touch $@


# if we do not recognize the target, warn and keep going.
.DEFAULT:
	@echo "***" Warning: Skipping unsupported target \"$@\"
