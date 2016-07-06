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

# Python lint
PYLINT := /usr/bin/pylint

# build places binaries here:
BIN := ../build

# VIB version is either set from outside, or includes git commit SHA
PKG_VERSION ?= 0.1.TP-pre.$(shell git log --pretty=format:'%h' -n 1)
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
PY_CLI   := $(filter-out %_test.py, $(wildcard cli/*.py))
PY_UTILS := $(filter-out %_test.py, $(wildcard utils/*.py))

C_SRC  := vmci/vmci_server.c

# Build both a 32-bit and a 64-bit library for vmci.
CFLAGS  := -fPIC -m32 -shared
CFLAGS64  := -fPIC -shared

cc := gcc

INIT_SCRIPT := vmdk-opsd

# Vib layout descriptor
DESCRIPTOR := descriptor.xml

# extra stuff to copy to vib
TOOLS := tools/lin32/mkfs.ext4

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
VMDKOPS_LIB     := $(VMDKOPS_PAYLOAD)$(ESX_LOC)/lib
VMDKOPS_LIB64   := $(VMDKOPS_PAYLOAD)$(ESX_LOC)/lib64
VMDKOPS_INITD   := $(VMDKOPS_PAYLOAD)/etc/init.d

# this is used from a Makefile upstairs, and could be used manually
.PHONY: build
build: $(VIB_BIN)

# What goes where in payloads:
TO_WDIR := $(DESCRIPTOR)
TO_ESX_BIN := $(PY_SRC) $(PY_CLI) $(TOOLS)
TO_ESX_PY  := $(PY_UTILS)
TO_ESX_LIB := $(VMCI_SRV_LIBS)
TO_ESX_INITD := $(INIT_SCRIPT)

# copy files to staging area and then build the VIB
$(VIB_BIN): $(TO_WDIR) $(TO_ESX_BIN) $(TO_ESX_PY) $(TO_ESX_LIB) $(TO_ESX_INITD)
	@echo === Staging in $(PAYLOAD) ===
	@mkdir -p $(VMDKOPS_BIN) $(VMDKOPS_PY) $(VMDKOPS_LIB) $(VMDKOPS_LIB64) $(VMDKOPS_INITD)
	@chmod -R a+w $(PAYLOAD)
	@cp $(TO_WDIR) $(WDIR)
	@cp $(TO_ESX_BIN) $(VMDKOPS_BIN)
	@cp $(TO_ESX_PY)  $(VMDKOPS_PY)
	@cp $(VMCI_SRV_LIB) $(VMDKOPS_LIB)/$(VMCI_SRV_LIB_NAME)
	@cp $(VMCI_SRV_LIB64) $(VMDKOPS_LIB64)/$(VMCI_SRV_LIB_NAME)
	@cp $(TO_ESX_INITD) $(VMDKOPS_INITD)
	$(VIBAUTHOR) $(VIBAUTHOR_PARAMS)


$(VMCI_SRV_LIB): $(C_SRC)
	$(CC) $(CFLAGS) -o $@ $(C_SRC)

$(VMCI_SRV_LIB64): $(C_SRC)
	$(CC) $(CFLAGS64) -o $@ $(C_SRC)

# Run lint on python source. Currently this is run manually and
# needs pylint and dependent packages to be installed.
# TODO: Automate.
.PHONY: lint
lint:
	@echo "Running $@"
	@rm -f ./.pylint
	@for file in $(PY_SRC) $(PY_UTILS) $(PY_CLI); do \
		$(PYLINT) -f text -r y $$file >> .pylint.out 2>&1; done

.PHONY:	clean
clean:
	rm -rf $(VIB) $(VMCI_SRV_LIBS) $(WDIR)/$(DESCRIPTOR) $(PAYLOAD)

.PHONY: test
test: test-esx build

# test-esx is a quick unittest for ESX Python code.
# Deploys, runs and clean unittests (i.e. all files *_test.py) on ESX
SSH := $(DEBUG) ssh -kTax -o StrictHostKeyChecking=no
SCP := $(DEBUG) scp -r -q -o StrictHostKeyChecking=no

TMP_LOC    := $(shell echo /tmp/vmdk_ops_unittest$$RANDOM)
TEST_FILES := $(shell find . -name '*_test.py')

test-esx:
	@echo Copying files to $(ESX):$(TMP_LOC) and running .py unittests ...
	$(SSH) root@$(ESX) 'mkdir -p $(TMP_LOC)'
	$(SCP) $(TEST_FILES) $(TO_ESX_BIN) $(TO_ESX_PY) root@$(ESX):$(TMP_LOC)
	$(SSH) root@$(ESX) \
		'for i in $(TMP_LOC)/*_test.py ; \
				do echo Running unit tests in $$i... ; python $$i ; \
				status=$$((status + $$?)) ;\
		done; \
		exit $$status'
	#$(SSH) root@$(ESX) rm -rf $(TMP_LOC)
