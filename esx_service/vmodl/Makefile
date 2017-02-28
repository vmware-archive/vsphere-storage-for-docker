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
# Basic makefile to validate and package python stuff
#
# Example:
# ESX=10.132.15.22 make test # deploys to ESX, runs *_test.py files to validate the stuff
#

# SSH/SCP commands. Use 'DEBUG=echo make' to print info instead of copying files or running ssh
SSH := $(DEBUG) ssh -kTax -o StrictHostKeyChecking=no
SCP := $(DEBUG) scp -r -q -o StrictHostKeyChecking=no

# Site libraries for Python 2 and Python 3
PY3_COMMON = /lib64/python3.5
PY2_COMMON = /lib/python2.7
TARGET2 := $(PY2_COMMON)/site-packages/pyMo/vim/vsan
TARGET3 := $(PY3_COMMON)/site-packages/pyMo/vim/vsan

# Find the target location for the correct python version
TARGET_LOC = $(shell $(SSH) root@$(ESX) \
	"if [ -d $(TARGET2) ] ; then echo $(TARGET2) ; else echo $(TARGET3); fi")

# Note: in ESX source, here are the location:
#bora/vsan/<component>/esx/pyMo (Impl and __init)
#bora/vsan/pyvmodl (model)

NAME  := VsphereContainerService
MODEL := $(NAME).py
EXT   := $(NAME)__ext_init__.py
IMPL  := $(NAME)Impl.py

PYFILES   := $(MODEL) $(EXT) $(IMPL)
TESTFILES := $(shell find . -name '*_test.py')
TMP_LOC   := $(shell echo /tmp/dockvol-vmodl_unittest$$RANDOM)

ifdef INSTALL_VMODL

deploy:
	@echo Making test dir...
	$(SSH) root@$(ESX) 'mkdir -p $(TARGET2) $(TARGET3) $(TMP_LOC)'
	@echo Copying VMODL python files to root@$(ESX):$(TARGET2) ...
	@$(SCP) $(PYFILES) root@$(ESX):$(TARGET2)/
	@echo Copying VMODL python files to root@$(ESX):$(TARGET3) ...
	@$(SCP) $(PYFILES) root@$(ESX):$(TARGET3)/
	@echo Copying test file to root@$(ESX):$(TMP_LOC) ...
	@$(SCP) $(TESTFILES) $(TO_ESX_BIN) $(TO_ESX_PY) root@$(ESX):$(TMP_LOC)

# Testing. Note that vsanmgmtd neeeds a second or two between 'restart' returns
# and the daemon start to really listen
test: deploy
	@echo Restarting VSAN mgmt daemon and running tests...
	@$(SSH) root@$(ESX) \
        'set -x ; \
		/etc/init.d/vsanmgmtd restart ; sleep 3 ; \
		for i in $(TMP_LOC)/*_test.py ; \
			do python $$i ; \
            status=$$((status + $$?)) ;\
        done ; \
		rm -rf $(TMP_LOC) ;\
        set +x ; exit $$status'

copy_to_payload:
	echo GENERATING VMODL PAYLOAD...
	mkdir -p ../$(VMDKOPS_PAYLOAD)$(TARGET2) ../$(VMDKOPS_PAYLOAD)$(TARGET3)
	cp $(PYFILES) $(TESTFILES) ../$(VMDKOPS_PAYLOAD)$(TARGET2)
	cp $(PYFILES) $(TESTFILES) ../$(VMDKOPS_PAYLOAD)$(TARGET3)

else

deploy test copy_to_payload:
	@echo "Ignoring VMODL '$@' target."
	@echo See https://github.com/vmware/docker-volume-vsphere/pull/975 for details.

endif


