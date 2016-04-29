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

# deliverables (placed by build)
BIN := ../bin
PKG_VERSION ?= 0.1.TP-pre.$(shell git log --pretty=format:'%h' -n 1)
VIB := vmware-esx-vmdkops-$(PKG_VERSION).vib
OFFLINE_DEPOT := vmware-esx-vmdkops-$(PKG_VERSION).zip
VIB_BIN  = $(BIN)/$(VIB)

# code
PY_SRC := vmdk_ops.py log_config.py\
	volumeKVStore.py kvESX.py \
	vmdkops_admin.py cli_table.py
C_SRC  := vmci/vmci_server.c
INIT_SCRIPT := vmdk-opsd
CONF_FILES  := log_config.json

# extra stuff to copy to vib
TOOLS := tools/lin32/mkfs.ext4


# Shared lib. Note that it is referred by name from vmci_srv.py
VMCI_SRV_LIB := ../bin/libvmci_srv.so

PY_FILES     = $(PY_SRC) $(VMCI_SRV_LIB)

# location on ESX where the stuff needs to go in payload
ESX_BIN := /usr/lib/vmware/vmdkops/bin
INITD_PATH := /etc/init.d
ETC_CONF_PATH := /etc/vmware/vmdkops

# payload location - see descriptor.xml
PAYLOAD=payloads
VMDKOPS_PAYLOAD=$(PAYLOAD)/vmdkops
VMDKOPS_BIN = $(VMDKOPS_PAYLOAD)$(ESX_BIN)
VMDKOPS_INITD = $(VMDKOPS_PAYLOAD)$(INITD_PATH)
VMDKOPS_ETC_CONF = $(VMDKOPS_PAYLOAD)$(ETC_CONF_PATH)

# We need to build C code as 32 bits.
# Reason: vSockets return cartelId to ID connecting VM. We use VSI to convert
#       it to VM name/ID. ESX is shipped with 32bit VSI python module only, so
#       we use 32bit python and 32 bit shared libs for VMCI/vSockets
CFLAGS  := -fPIC -m32 -shared
cc := gcc
VIBAUTHOR := vibauthor
WDIR := .

VIBAUTHOR_PARAMS := --debug --compose \
	--vib=$(BIN)/$(VIB) \
	--stage-dir $(WDIR) \
	--offline-depot $(BIN)/$(OFFLINE_DEPOT) \
	--force

# this is used from a Makefile upstairs, and could be used manually
.PHONY: build
build: $(VIB_BIN)

$(VIB_BIN): descriptor.xml $(PY_FILES) $(TOOLS) $(INIT_SCRIPT) $(CONF_FILES)
	@echo === Staging in $(VMDKOPS_BIN), $(VMDKOPS_INITD), $(VMDKOPS_ETC_CONF)
	@mkdir -p $(VMDKOPS_BIN)
	@mkdir -p $(VMDKOPS_INITD)
	@mkdir -p $(VMDKOPS_ETC_CONF)
	@chmod -R a+w $(PAYLOAD)
	@cp -f $(PY_FILES) $(TOOLS) $(VMDKOPS_BIN)
	@cp -f $(INIT_SCRIPT) $(VMDKOPS_INITD)
	@cp -f $(CONF_FILES) $(VMDKOPS_ETC_CONF)
	$(VIBAUTHOR) $(VIBAUTHOR_PARAMS)

$(VMCI_SRV_LIB): $(C_SRC)
	$(CC) $(CFLAGS) -o $@ $(C_SRC)

.PHONY:	clean
clean:
	rm -rf $(VIB) $(VMCI_SRV_LIB) $(PAYLOAD)

.PHONY: test
test: build
	@echo "? Info: No unit test here yet."
