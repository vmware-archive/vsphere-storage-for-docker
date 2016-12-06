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
# Makefile for ESX "vmdkops" service - host part of "vmware/vmdkops" Go package
#
# Prepares payload folder for vibauthor and builds/places neede stuff there
# We expect the actual vibauthor to be run via dedicated container upstair
#

# build places binaries here:
BIN := ../build

# PKG_VERSION is either set externally as part of a build off a tag or
# suffixed with a sha1 of the most recent commit to the prefix of the
# most recent tag. Tagged builds use the externally defined version,
# developer builds use last tagged release and sha1 of the most recent commit.
# Format: <last tagged release>.<last commit hash>
PKG_VERSION ?= $(shell \
	       git describe --tags `git rev-list --tags --max-count=1` \
	       ).$(shell \
	       git log --pretty=format:'%h' -n 1)

# Release version needed for vib
RELEASE_VERSION ?= 0.0.1

VIB := vmware-esx-vmdkops-$(PKG_VERSION).vib
OFFLINE_DEPOT := vmware-esx-vmdkops-$(PKG_VERSION).zip
VIB_BIN  = $(BIN)/$(VIB)

# Shared lib. Note that it is referred by name from vmci_srv.py
VMCI_SRV_LIB_NAME := libvmci_srv.so
VMCI_SRV_LIB   := $(BIN)/$(VMCI_SRV_LIB_NAME)
VMCI_SRV_LIB64 := $(BIN)/$(VMCI_SRV_LIB_NAME).64
VMCI_SRV_LIBS  := $(VMCI_SRV_LIB) $(VMCI_SRV_LIB64)

# Source code - all .py stuff which is not tests
PY_SRC   := $(filter-out %_test.py, $(wildcard *.py))
# exclude __init__.py to avoid clash with __init__.py from parent
PY_CLI   := $(filter-out %_test.py, $(wildcard cli/[a-z]*.py))
PY_UTILS := $(filter-out %_test.py, $(wildcard utils/*.py))

C_SRC  := vmci/vmci_server.c

# Build both a 32-bit and a 64-bit library for vmci.
CFLAGS  := -fPIC -m32 -shared
CFLAGS64  := -fPIC -shared

cc := gcc

INIT_SCRIPT := vmdk-opsd

# Vib layout descriptor
DESCRIPTOR := descriptor.xml

# vibauthor definitions
VIBAUTHOR := vibauthor
WDIR := $(BIN)

VIBAUTHOR_PARAMS := --debug --compose \
	--vib=$(BIN)/$(VIB) \
	--stage-dir $(WDIR) \
	--offline-depot $(BIN)/$(OFFLINE_DEPOT) \
	--force


# Payload root
PAYLOAD=$(WDIR)/payloads

# Location on ESX where the stuff needs to go
ESX_LOC := /usr/lib/vmware/vmdkops

# Location in VIB staging ("payload")
# 'vmdkops' name under payloads should match payloads.name in descriptor.xml
VMDKOPS_PAYLOAD := $(PAYLOAD)/vmdkops
VMDKOPS_BIN     := $(VMDKOPS_PAYLOAD)$(ESX_LOC)/bin
VMDKOPS_PY      := $(VMDKOPS_PAYLOAD)$(ESX_LOC)/Python
VMDKOPS_PY2     := $(VMDKOPS_PAYLOAD)$(ESX_LOC)/Python/2
VMDKOPS_LIB     := $(VMDKOPS_PAYLOAD)$(ESX_LOC)/lib
VMDKOPS_LIB64   := $(VMDKOPS_PAYLOAD)$(ESX_LOC)/lib64
VMDKOPS_INITD   := $(VMDKOPS_PAYLOAD)/etc/init.d

# build the docker-volume-plugin

UI_FROM := ../ui/build/dist/*
UI_TO_BASE := $(VMDKOPS_PAYLOAD)/usr/lib/vmware
UI_TO_HOSTD := /hostd/docroot/ui/plugins/docker-volume-plugin/
UI_TO := $(UI_TO_BASE)$(UI_TO_HOSTD)

# this is used from a Makefile upstairs, and could be used manually
.PHONY: build
build: $(VIB_BIN)

# What needs to be placed in payloads:
TO_WDIR := $(DESCRIPTOR)
TO_ESX_BIN := $(PY_SRC) $(PY_CLI)
TO_ESX_PY  := $(PY_UTILS)
TO_ESX_LIB := $(VMCI_SRV_LIBS)
TO_ESX_INITD := $(INIT_SCRIPT)


# SQLITE python libs. We need that on ESX before 6.0 U2
PY_SQLITE := tools/sqlite/sqlite3
SO_SQLITE := tools/sqlite/*.so

# copy files to staging area and then build the VIB
$(VIB_BIN): $(TO_WDIR) $(TO_ESX_BIN) $(TO_ESX_PY) $(TO_ESX_LIB) $(TO_ESX_INITD) $(PY_SQLITE) $(SO_SQLITE)
	@echo Staging in $(PAYLOAD)
ifeq ($(INCLUDE_UI), true)
	@mkdir -p $(VMDKOPS_BIN) $(VMDKOPS_PY) $(VMDKOPS_PY2) $(VMDKOPS_LIB) $(VMDKOPS_LIB64) $(VMDKOPS_INITD) $(UI_TO)
else
	@mkdir -p $(VMDKOPS_BIN) $(VMDKOPS_PY) $(VMDKOPS_PY2) $(VMDKOPS_LIB) $(VMDKOPS_LIB64) $(VMDKOPS_INITD)
endif
	@chmod -R u+w $(PAYLOAD)
	cp $(TO_ESX_BIN) $(VMDKOPS_BIN)
	cp $(TO_ESX_PY)  $(VMDKOPS_PY)
	cp -R $(PY_SQLITE) $(VMDKOPS_PY2)
ifeq ($(INCLUDE_UI), true)
	cp -R $(UI_FROM) $(UI_TO)
endif
	cp $(SO_SQLITE) $(VMDKOPS_PY2)
	cp $(VMCI_SRV_LIB) $(VMDKOPS_LIB)/$(VMCI_SRV_LIB_NAME)
	cp $(VMCI_SRV_LIB64) $(VMDKOPS_LIB64)/$(VMCI_SRV_LIB_NAME)
	cp $(TO_ESX_INITD) $(VMDKOPS_INITD)
	@chmod -R u+x $(VMDKOPS_BIN)
	@sed "s/   <version>1.0.0-0.0.1<\/version>/   <version>$(PKG_VERSION)-$(RELEASE_VERSION)<\/version>/g" $(TO_WDIR) > $(WDIR)/$(DESCRIPTOR)
	$(VIBAUTHOR) $(VIBAUTHOR_PARAMS)

$(VMCI_SRV_LIB): $(C_SRC)
	$(CC) $(CFLAGS) -o $@ $(C_SRC)

$(VMCI_SRV_LIB64): $(C_SRC)
	$(CC) $(CFLAGS64) -o $@ $(C_SRC)

# Run pylint on python source. Currently this is run manually and
# needs pylint and dependent packages to be installed.
# TODO: Automate.

# we will incrementally start failing on issues issue. For now, ignore ALL but errors.
PYLINT_IGNORE := --errors-only -d import-error,no-name-in-module
PYLINT_COMMAND := pylint $(PYLINT_IGNORE) --rcfile pylint_cfg.ini

.PHONY: pylint
pylint: .lint_src .lint_cli .lint_util

.lint_src: $(PY_SRC)
	$(PYLINT_COMMAND) ../esx_service
	@touch $@

.lint_cli: $(PY_CLI)
	$(PYLINT_COMMAND) cli
	@touch $@

.lint_util: $(PY_UTIL)
	$(PYLINT_COMMAND) utils
	@touch $@


.PHONY:	clean
clean:
	rm -rf $(VIB) $(VMCI_SRV_LIBS) $(WDIR)/$(DESCRIPTOR) $(PAYLOAD) .lint_*

.PHONY: test
test: test-esx build

# test-esx is a quick unittest for ESX Python code.
# Deploys, runs and clean unittests (i.e. all files *_test.py) on ESX
SSH := $(DEBUG) ssh $(SSH_KEY_OPT) -kTax -o StrictHostKeyChecking=no
SCP := $(DEBUG) scp $(SSH_KEY_OPT) -r -q -o StrictHostKeyChecking=no

TMP_LOC    := $(shell echo /tmp/vmdk_ops_unittest$$RANDOM)
TEST_FILES := $(shell find . -name '*_test.py')

# By default, we run all *_test.py. Use TEST_PATTERN to narrow the test set, e.g.
# e.g. make test-esx TEST_PATTERN='*admin*'
TEST_PATTERN := '*'
AUTH_DB_PATH = '/etc/vmware/vmdkops/auth-db'

test-esx:
	@echo "Removing auth DB file at $(ESX):$(AUTH_DB_PATH)"
	$(SSH) root@$(ESX) 'rm -f $(AUTH_DB_PATH)'
	@echo "Copying files to $(ESX):$(TMP_LOC) and running .py unittests ..."
	$(SSH) root@$(ESX) 'mkdir -p $(TMP_LOC)'
	$(SCP) $(TEST_FILES) $(TO_ESX_BIN) $(TO_ESX_PY) root@$(ESX):$(TMP_LOC)
	$(SSH) root@$(ESX) \
		'for i in $(TMP_LOC)/$(TEST_PATTERN)_test.py ; \
				do echo Running unit tests in $$i... ; python $$i ; \
				status=$$((status + $$?)) ;\
		done; \
		exit $$status'
	#$(SSH) root@$(ESX) rm -rf $(TMP_LOC)
