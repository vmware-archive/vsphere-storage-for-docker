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

-include ../plugin/Makefile

EPOCH := 0

# Place binaries here
BIN := ../build
MANAGED_PLUGIN := ../plugin

# Location for scripts
SCRIPTS     := ../misc/scripts

# Packaging variables
PLUGNAME  := docker-volume-vsphere
GOPATH_PLUGNAME := $(PLUGNAME)/vmdk_plugin
GOPATH_ORG :=vmware
MAINTAINERS := cna-storage@vmware.com
REPO_URL    := https://github.com/$(GOPATH_ORG)/$(PLUGNAME)
MIN_DOCKER_VERSION :=1.9
DOCKER_PACKAGE := docker-engine
DOCKER_PACKAGE_PHOTON := docker
AFTER_INSTALL  := $(SCRIPTS)/install/after-install.sh
BEFORE_REMOVE  := $(SCRIPTS)/install/before-remove.sh
AFTER_REMOVE   := $(SCRIPTS)/install/after-remove.sh

#
# Scripts to deploy and control services - used from Makefile and from Drone CI
#
CHECK	    := $(SCRIPTS)/check.sh
BUILD       := $(SCRIPTS)/build.sh
DEPLOY_TOOLS := $(SCRIPTS)/deploy-tools.sh
SYSTEMD_UNIT := $(SCRIPTS)/install/$(PLUGNAME).service
UPSTART_CONF := $(SCRIPTS)/install/$(PLUGNAME).conf
PACKAGE      := package
SYSTEMD_LIB  := $(PACKAGE)/lib/systemd/system/
INSTALL_BIN  := $(PACKAGE)/usr/local/bin
UPSTART_INIT := $(PACKAGE)/etc/init/

# esx service for docker volume ops
ESX_SRC     := ../esx_service
VMCI_SRC    := $(ESX_SRC)/vmci/*.h $(ESX_SRC)/vmci/vmci_client.c

#  binaries location
PLUGIN_BIN = $(BIN)/$(PLUGNAME)

# all binaries for VMs - plugin and tests
VM_BINS = $(PLUGIN_BIN) $(BIN)/$(VMDKOPS_TEST_MODULE).test $(BIN)/$(PLUGNAME).test

VIBFILE := vmware-esx-vmdkops-$(PKG_VERSION).vib
VIB_BIN := $(BIN)/$(VIBFILE)

# plugin name, for go build
PLUGIN := github.com/$(GOPATH_ORG)/$(GOPATH_PLUGNAME)

# test location, picking up E2E tests
E2E_Tests := github.com/$(GOPATH_ORG)/$(PLUGNAME)/tests/e2e

GO := GO15VENDOREXPERIMENT=1 go
FPM := fpm

# make sure we rebuild of vmkdops or Dockerfile change (since we develop them together)
VMDKOPS_MODULE := drivers/vmdk/vmdkops
VMDKOPS_TEST_MODULE := vmdkops
VMDKOPS_MODULE_SRC = $(VMDKOPS_MODULE)/*.go $(VMCI_SRC)

# All sources. We rebuild if anything changes here
SRC = main.go log_formatter.go utils/refcount/refcnt.go \
	utils/fs/fs.go utils/config/config.go utils/log_formatter/log_formatter.go \
	utils/plugin_utils/plugin_utils.go \
	drivers/photon/photon_driver.go drivers/vmdk/vmdk_driver.go

TEST_SRC = ../tests/utils/inputparams/testparams.go

# Canned recipe
define log_target
@echo
@echo "=> Running target $@" `date`
@echo
endef

# The default build is using a prebuilt docker image that has all dependencies.
.PHONY: dockerbuild build-all
build-all: dockerbuild package

dockerbuild:
	@$(CHECK) dockerbuild
	$(BUILD) pylint
	$(BUILD) build


# The non docker build.
.PHONY: build pylint
build: prereqs .code_verify $(VM_BINS)
	$(MAKE)  --directory=$(ESX_SRC) $@
pylint:
	$(MAKE)  --directory=$(ESX_SRC) $@

.PHONY: gvt
gvt:
	$(BUILD) gvt

.PHONY: documentation
documentation:
	$(BUILD) documentation

.PHONY: prereqs
prereqs:
	@$(CHECK)

$(PLUGIN_BIN): $(SRC) $(VMDKOPS_MODULE_SRC)
	@-mkdir -p $(BIN) && chmod a+w $(BIN)
	$(GO) build --ldflags '-extldflags "-static"' -o $(PLUGIN_BIN) $(PLUGIN)

$(BIN)/$(VMDKOPS_TEST_MODULE).test: $(VMDKOPS_MODULE_SRC) $(TEST_SRC) $(VMDKOPS_MODULE)/*_test.go
	$(GO) test -c -o $@ $(PLUGIN)/$(VMDKOPS_MODULE) -cover

$(BIN)/$(PLUGNAME).test: $(SRC) $(TEST_SRC) *_test.go
	$(GO) test -c -o $@ $(PLUGIN) -cover

.PHONY: clean
clean:
	$(BUILD) clean-as-root

.PHONY: clean-as-root
clean-as-root: pkg-post
	rm -rf $(BIN) .code_verify
	$(MAKE) --directory=$(ESX_SRC) clean

# GO Code quality checks.

DIRS_TO_VERIFY := . utils/fs utils/config drivers/photon drivers/vmdk drivers/vmdk/vmdkops ../tests/e2e \
	../tests/utils/dockercli ../tests/utils/inputparams ../tests/utils/verification ../tests/constants/admincli \
	../tests/constants/dockercli ../tests/utils/ssh ../tests/utils/misc ../tests/constants/vm

FILES_TO_VERIFY := $(foreach dir, $(DIRS_TO_VERIFY), $(dir)/*.go)
.code_verify: $(FILES_TO_VERIFY)
	@for i in $(DIRS_TO_VERIFY) ; do ${GOPATH}/bin/golint $$i ; done
	go tool vet .
	gofmt -s -l -w .
	@touch $@

# Build the linux distro packages

DOCKER = $(DEBUG) docker

.PHONY: fpm-docker
fpm-docker:
	@echo "Building FPM container. It could take a while, please stand by..."
	$(DOCKER) build -t cnastorage/fpm -f misc/dockerfiles/Dockerfile.fpm . > /dev/null

.PHONY: pkg
pkg: dockerdeb dockerrpm

.PHONY: package
package: pkg

.PHONY: dockerdeb
dockerdeb: pkg-post
	$(BUILD) deb

.PHONY: dockerrpm pkg-post
dockerrpm:
	$(BUILD) rpm

DESCRIPTION := "Docker Volume Driver for vSphere"
FPM_COMMON := -p $(BIN) \
	-C $(PACKAGE) \
	-s dir \
	-n $(PLUGNAME) \
	-v $(PKG_VERSION) \
        --provides $(PLUGNAME) \
        -m $(MAINTAINERS) \
        --url $(REPO_URL) \
	--after-install $(AFTER_INSTALL) \
	--before-remove $(BEFORE_REMOVE) \
	--after-remove $(AFTER_REMOVE) \
	--description $(DESCRIPTION) \
	--architecture x86_64 \
	--force

# FPM should be installed for target deb
.PHONY: deb
deb: pkg-prep deb-pkg-prep
	@$(CHECK) pkg
	$(FPM) --deb-no-default-config-files $(FPM_COMMON) -t deb .

# Upstart needed for older versions of Ubuntu
.PHONY: deb-pkg-prep
deb-pkg-prep:
	@mkdir -p $(UPSTART_INIT)
	@cp $(UPSTART_CONF) $(UPSTART_INIT)

# Target to build for rpm based systems. Due to variations in how docker is packaged between RHEL and Photon,
# skip dependency check vs building 2 different rpm packages.
.PHONY: rpm
rpm: pkg-prep
	@$(CHECK) pkg
	$(FPM) --epoch $(EPOCH) $(FPM_COMMON) -t rpm .

.PHONY: pkg-prep
pkg-prep:
	@mkdir -p $(SYSTEMD_LIB)
	@cp $(SYSTEMD_UNIT) $(SYSTEMD_LIB)
	@mkdir -p $(INSTALL_BIN)
	@cp $(PLUGIN_BIN) $(INSTALL_BIN)
	@chmod a+w -R $(PACKAGE)

.PHONY: pkg-post
pkg-post:
	@rm -rf $(PACKAGE)
	@rm -rf DEBIAN

#
# 'make deploy'
# ----------
# temporary goal to simplify my deployments and sanity check test (create/delete)
#
# expectations:
#   Need target machines (ESX/Guest) to have proper ~/.ssh/authorized_keys
#
# You can
#   set ESX and VM (and VM1 / VM2) as env. vars,
# or pass on command line
# 	make deploy-esx ESX=10.20.105.54
# 	make deploy-vm  VM=10.20.105.121
# 	make testremote  ESX=10.20.105.54 VM1=10.20.105.121 VM2=10.20.105.122


VM1 ?= "$(VM)"
VM2 ?= "$(VM)"
TEST_VM_USER ?= root

TEST_VM = $(TEST_VM_USER)@$(VM1)

VM1_DOCKER = tcp://$(VM1):2375
VM2_DOCKER = tcp://$(VM2):2375

SSH := $(DEBUG) ssh $(SSH_KEY_OPT) -kTax -o StrictHostKeyChecking=no

# bin locations on target guest
TMP_LOC := /tmp/$(PLUGNAME)

# script sources live here. All scripts are copied to test VM during deployment
# scripts started locally to deploy to and clean up test machines
DEPLOY_VM_SH  := $(DEPLOY_TOOLS) deployvm
DEPLOY_VM_TEST_SH  := $(DEPLOY_TOOLS) deployvmtest
DEPLOY_VM_PLUGIN_SH := $(DEPLOY_TOOLS) deployplugin
DEPLOY_ESX_SH := $(DEPLOY_TOOLS) deployesx
CLEANVM_SH    := $(DEPLOY_TOOLS) cleanvm
CLEANESX_SH   := $(DEPLOY_TOOLS) cleanesx


#
# Deploy to existing testbed, Expects ESX VM1 and VM2 env vars
#
.PHONY: deploy-esx deploy-vm deploy-vm-test deploy deploy-all deploy
deploy-esx: clean-esx
	$(DEPLOY_ESX_SH) "$(ESX)" "$(VIB_BIN)"

VMS= $(VM1) $(VM2) $(WORKER2) $(CI_NODE4)

deploy-vm: clean-vm
	$(DEPLOY_VM_SH) "$(VMS)" $(PLUGIN_NAME):$(PLUGIN_TAG)
deploy-vm-test:
	$(DEPLOY_VM_TEST_SH) "$(VMS)" $(BIN) $(SCRIPTS)
deploy-vm-plugin:
	$(DEPLOY_VM_PLUGIN_SH) "$(VM2)" $(PLUGIN_BIN) $(MANAGED_PLUGIN) $(SCRIPTS) $(DOCKER_HUB_REPO) $(VERSION_TAG) $(EXTRA_TAG)

deploy-all: deploy-esx deploy-vm deploy-vm-test
deploy: deploy-all

#
# 'make test' or 'make testremote'
# CAUTION: FOR NOW, RUN testremote DIRECTLY VIA 'make'. DO NOT run via './build.sh'
#  reason: ssh keys for accessing remote hosts are not in container used by ./build.sh
# ----------

# this is a set of unit tests run on build machine
.PHONY: test
test:
	$(log_target)
	$(BUILD) testasroot

.PHONY: testasroot
testasroot:
	$(log_target)
	$(GO) test $(PLUGIN)/drivers/vmdk/vmdkops -cover -v
	$(GO) test $(PLUGIN)/utils/config -cover -v

# does sanity check of create/remove docker volume on the guest
TEST_VOL_NAME ?= DefaultTestVol
TEST_VERBOSE   = -test.v

CONN_MSG := "Please make sure Docker is running and is configured to accept TCP connections"
.PHONY: checkremote
checkremote:
	@$(SSH) $(TEST_VM) docker -H $(VM1_DOCKER) ps > /dev/null 2>/dev/null || \
		(echo VM1 $(VM1): $(CONN_MSG) ; exit 1)
	@$(SSH) $(TEST_VM) docker -H $(VM2_DOCKER) ps > /dev/null 2>/dev/null || \
		(echo VM2 $(VM2): $(CONN_MSG); exit 1)

.PHONY: test-vm test-esx test-all testremote test-e2e
# test-vm runs GO unit tests and plugin test suite on a guest VM
# expects binaries to be deployed ot the VM and ESX (see deploy-all target)
test-vm: checkremote deploy-vm-test
	$(log_target)
	$(SSH) $(TEST_VM) $(TMP_LOC)/$(VMDKOPS_TEST_MODULE).test $(TEST_VERBOSE)
	$(SSH) $(TEST_VM) $(TMP_LOC)/$(PLUGNAME).test $(TEST_VERBOSE) \
		-v $(TEST_VOL_NAME) \
		-H1 $(VM1_DOCKER) -H2 $(VM2_DOCKER)

# test-esx is a quick unittest for ESX Python code.
test-esx:
	$(log_target)
	$(MAKE)  --directory=$(ESX_SRC) $@

testremote: test-esx test-vm
test-all:  test test-e2e testremote

# capture coverage of vDVS driver code.
coverage:
	$(log_target)
	$(MAKE)  --directory=$(ESX_SRC) $@

# An end-to-end test target for vSphere Docker Volume Driver
test-e2e:
	$(log_target)
	$(BUILD) test-e2e-runalways
	$(BUILD) test-e2e-runonce

#
# E2E test target to control test run on CI.
# We have 2 ESXs on CI testbed and 2 datastore type each (VMFS/VSAN).
# test-e2e-runalways: makes sure to run test always
# test-e2e-runonce: makes sure to run test *once per ESX*
#
.PHONY: test-e2e-runalways test-e2e-runonce
test-e2e-runalways:
	$(log_target)
	$(GO) test -v -timeout 30m -tags runalways $(E2E_Tests)

test-e2e-runonce:
	$(log_target)
	$(GO) test -v -timeout 30m -tags runonce $(E2E_Tests)

MANAGED_PLUGIN_NAME="vsphere:latest"

.PHONY:clean-vm clean-esx clean-all clean-docker
clean-vm:
	$(CLEANVM_SH) "$(VMS)" "$(MANAGED_PLUGIN_NAME)"

clean-esx:
	$(CLEANESX_SH) "$(ESX)" esx-vmdkops-service

# rm ALL containers and volumes. Useful for post-failure force cleanup
clean-docker:
	-$(SSH) $(TEST_VM) 'docker kill `docker ps -q`'
	-$(SSH) $(TEST_VM) 'docker rm `docker ps -a -q`'
	-$(SSH) $(TEST_VM) 'docker volume rm `docker volume ls -q`'

AUTH_DB_PATH = '/etc/vmware/vmdkops/auth-db'
clean-auth-db:
	@echo "Removing auth DB file at $(ESX):$(AUTH_DB_PATH)"
	$(SSH) root@$(ESX) 'rm -f $(AUTH_DB_PATH)'

clean-all: clean clean-vm clean-esx

# full circle
all: clean-all build-all deploy-all test-all


# if we do not recognize the target, warn and keep going.
.DEFAULT:
	@echo "***" Warning: Skipping unsupported target \"$@\"
