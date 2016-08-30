<<<<<<< 443efb213f303c7760699f57da3648f6535fc715
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
	clean-all clean clean-vm clean-esx \
	build-all build dockerbuild \
	deb rpm package gvt documentation

# default target
default: build-all

# redirect all
$(TARGETS):
	$(MAKE) --directory=vmdk_plugin $@
=======
export PROJECT := esxui-example-plugin
PROJECT_ROOT = $(MAINSRCROOT)
GOBUILD_TARGET ?= $(PROJECT)

include support/make/typical.make

BUILDTYPE := $(OBJDIR)
VERSION := 1.0.0

.PHONY: all clean npm

all: vib

# We might be invoked from the main ESXUI build, in which case we
# don't need to re-download deps.
ifeq ($(GOBUILD_TARGET),$(PROJECT))
deps: $(GOBUILD_TARGET)-deps
include $(MAINSRCROOT)/support/gobuild/make/auto-components.mk
else
deps:
$(GOBUILD_TARGET)-deps:
	$(TOUCH) $(BUILDROOT)/deps.stamp
endif

GRUNT_STAMP = $(BUILDROOT)/grunt.stamp
NPM_STAMP = $(BUILDROOT)/npm.stamp

# Tools
NODE_BIN := $(GOBUILD_CAYMAN_NODEJS_ROOT)/$(ARCH)/bin
NODE := $(NODE_BIN)/$(NODE_EXE)
NPM := npm_config_cache=$(NPMCACHE) \
	npm_config_registry=$(VMW_NPM_REGISTRY) \
	$(NODE_BIN)/$(NPM_EXE)
VIBAUTHOR := $(GOBUILD_VIBSUITE_ROOT)/bora/apps/vibtool/vibauthor
VIBPUBLISH := $(GOBUILD_VIBSUITE_ROOT)/bora/apps/vibtool/vibpublish
VIBSIGN = --force --sign -r $(TEST_CERT) -k $(TEST_KEY)
SIGNC := /build/toolchain/noarch/vmware/signserver/signc

# Certificates
TEST_CERT := $(GOBUILD_VIBSUITE_ROOT)/bora/testcerts/vmware.cert
TEST_KEY := $(GOBUILD_VIBSUITE_ROOT)/bora/testcerts/vmware.key

# VIB construction
VIB = $(PROJECT)-$(BUILD_NUMBER).vib
VIB_SIGNED = $(PROJECT)-signed-$(BUILD_NUMBER).vib
VIB_TEST_SIGNED = $(PROJECT)-signed-test-cert-$(BUILD_NUMBER).vib
VIBFILE = $(BUILDROOT)/$(VIB)
VIBFILE_SIGNED = $(BUILDROOT)/$(VIB_SIGNED)
VIBFILE_TEST_SIGNED = $(BUILDROOT)/$(VIB_TEST_SIGNED)
VIBSTAGE = $(BUILDROOT)/vib-stage
VIB_SIGNATURE_FILE := $(BUILDROOT)/sig.pkcs7
VIBDESC = $(MAINSRCROOT)/vib-descriptor.template.xml
VIBDESC_FILLED = $(BUILDROOT)/vib-descriptor.template.filled.xml
VIB_BULLETIN = $(MAINSRCROOT)/vib-bulletin.template.xml
VIB_BULLETIN_FILLED=$(BUILDROOT)/vib-bulletin.template.filled.xml

ESX_DOC_ROOT = $(VIBSTAGE)/usr/lib/vmware/hostd/docroot/ui
PLUGIN_ROOT = $(ESX_DOC_ROOT)/plugins

# Offline bundle construction
OFFLINE_BUNDLE_STAGE = $(BUILDROOT)/offline-bundle-stage
OFFLINE_BUNDLE_5x = $(BUILDROOT)/$(PROJECT)-offline-bundle-5.x-$(BUILD_NUMBER).zip
OFFLINE_BUNDLE_6x = $(BUILDROOT)/$(PROJECT)-offline-bundle-6.x-$(BUILD_NUMBER).zip

DEPS = $(filter %.js %.html %.scss,$(shell $(FIND) $(PROJECT_ROOT) -type f))
DEPS += $(PROJECT_ROOT)/Gruntfile.js

NPM_VERBOSE :=
ifeq ($(VERBOSE),3)
NPM_VERBOSE = "--verbose"
endif


npm: $(NPM_STAMP)
$(NPM_STAMP): $(PROJECT_ROOT)/package.json
	@echo "*** Running NPM install"
	@$(MKDIR) -p $(BUILDROOT)
	cd $(PROJECT_ROOT); \
		PATH=/bin:/usr/bin:$(GIT_BIN):$(NODE_BIN):$(AUTOCONF_BIN):$(NASM_BIN):$(AUTOMAKE_BIN) \
		PYTHON=$(PYTHON) \
		CXX="$(CXX)" \
		CPP="$(CPP)" \
		LD="$(LD)" \
		CC="$(GCC)" \
		GIT="$(GIT)" \
		$(NPM) $(NPM_VERBOSE) install && \
		$(TOUCH) $(NPM_STAMP)


$(GRUNT_STAMP): $(NPM_STAMP) $(DEPS)
	@echo "*** Running grunt --force --debug for obj build"
	@cd $(PROJECT_ROOT); \
		PATH=$(GIT_BIN):$(RUBY_BIN):$(COMPASS_BIN):$(PATH) \
		BUILDTYPE=$(BUILDTYPE) \
		CHANGE_NUMBER=$(CHANGE_NUMBER) \
		BUILD_NUMBER=$(BUILD_NUMBER) \
		PRODUCT_BUILD_NUMBER=$(PRODUCT_BUILD_NUMBER) \
		VERSION=$(VERSION) \
		BRANCH_NAME=$(BRANCH_NAME) \
		$(NODE) $(GRUNT) --debug --force --stack --verbose && \
		$(TOUCH) $(GRUNT_STAMP)


$(VIBAUTHOR): deps

vib: $(VIBFILE)
$(VIBFILE): $(GRUNT_STAMP) $(VIBAUTHOR) $(VIBDESC)
	@if [ "$$OSTYPE" = "linux" ]; then \
		echo "*** Staging VIB dependencies"; \
		$(RM) -Rf $(PLUGIN_ROOT); \
		$(MKDIR) -p $(PLUGIN_ROOT); \
		$(RSYNC) -rvu $(BUILDROOT)/dist/* $(PLUGIN_ROOT)/$(PROJECT); \
		echo "*** Producing VIB"; \
		$(SED) 's/%%BUILDNUMBER%%/$(BUILD_NUMBER)/' $(VIBDESC) > $(VIBDESC_FILLED); \
		$(SED) -i 's/%%PROJECT%%/$(PROJECT)/' $(VIBDESC_FILLED); \
		$(SED) -i 's/%%VERSION%%/$(VERSION)/' $(VIBDESC_FILLED); \
		$(RM) -f $(@); \
		CMD=`echo "$(VIBAUTHOR) -c -v $(@) -t $(VIBSTAGE) -d $(VIBDESC_FILLED) $(VIBSIGN)"`; \
		echo "Running vibauthor --create: $$CMD"; \
		$$CMD; \
	else \
		echo "*** Not producing VIB on non-linux platform"; \
	fi;


PAYLOAD_NAME = $(shell echo $(PROJECT) | $(AWK) '{print substr($$0,1,15)}')

vib-signed: $(VIBFILE_SIGNED)
$(VIBFILE_SIGNED): $(VIBFILE)
ifdef USE_OFFICIAL_KEY
	@echo "*** Generating officially signed VIB"
	cd $(BUILDROOT); \
	PATH=$(PATH):/bin:/usr/bin cp $(<) $(@); \
	PATH=$(PATH):/bin:/usr/bin ar -x $(@); \
	PATH=$(PATH):/bin:/usr/bin chmod 777 descriptor.xml sig.pkcs7 $(PAYLOAD_NAME); \
	PATH=$(PATH):/bin:/usr/bin rm sig.pkcs7 $(PAYLOAD_NAME); \
	PATH=$(PATH):/bin:/usr/bin $(SIGNC) \
		--input=descriptor.xml \
		--keyid=vmware_esx40 \
		--signmethod=vibsign-1.0 \
		--output=$(VIB_SIGNATURE_FILE); \
	PATH=$(PATH):/bin:/usr/bin ar -r $(@) descriptor.xml; \
	PATH=$(PATH):/bin:/usr/bin ar -r $(@) $(VIB_SIGNATURE_FILE)
else
	 @echo "*** Not generating officially signed VIB"
endif


vib-test-signed: $(VIBFILE_TEST_SIGNED)
$(VIBFILE_TEST_SIGNED): $(VIBFILE)
	@echo "*** Generating test signed VIB $(PAYLOAD_NAME)"
	cd $(BUILDROOT); \
	PATH=$(PATH):/bin:/usr/bin cp $(<) $(@); \
	PATH=$(PATH):/bin:/usr/bin ar -x $(@); \
	PATH=$(PATH):/bin:/usr/bin ls .; \
	PATH=$(PATH):/bin:/usr/bin chmod 777 descriptor.xml sig.pkcs7 $(PAYLOAD_NAME); \
	PATH=$(PATH):/bin:/usr/bin rm sig.pkcs7 $(PAYLOAD_NAME); \
	PATH=$(PATH):/bin:/usr/bin $(SIGNC) \
		--input=descriptor.xml \
		--keyid=elfsign_test \
		--signmethod=vibsign-1.0 \
		--output=$(VIB_SIGNATURE_FILE); \
	PATH=$(PATH):/bin:/usr/bin ar -r $(@) descriptor.xml; \
	PATH=$(PATH):/bin:/usr/bin ar -r $(@) $(VIB_SIGNATURE_FILE)


offline-bundle: $(OFFLINE_BUNDLE_6x)
$(OFFLINE_BUNDLE_6x): $(VIBFILE_TEST_SIGNED) $(VIBFILE_SIGNED) $(VIB_BULLETIN)
	$(SED) 's/%%BUILDNUMBER%%/$(BUILD_NUMBER)/' $(VIB_BULLETIN) > $(VIB_BULLETIN_FILLED)
	$(SED) -i 's/%%VERSION%%/$(VERSION)/' $(VIB_BULLETIN_FILLED)
	$(SED) -i 's/%%PROJECT%%/$(PROJECT)/' $(VIB_BULLETIN_FILLED)
	$(SED) -i 's/%%ESX_TARGET_VERSION%%/6.*/' $(VIB_BULLETIN_FILLED)
ifdef USE_OFFICIAL_KEY
	@echo "*** Generating officially signed offline bundle for ESX 6.x"
	@$(RM) -Rf $(OFFLINE_BUNDLE_STAGE)
	@$(MKDIR) -p $(OFFLINE_BUNDLE_STAGE)
	$(VIBPUBLISH) -g \
		--create-offline-bundle=$(@) \
		--force \
		--bulletin $(VIB_BULLETIN_FILLED) \
		-s $(VIBFILE_SIGNED) \
		--target ESXi,6.0.0 \
		--vendor VMware \
		--vendor-code vmw \
		-o $(OFFLINE_BUNDLE_STAGE)
else
	@echo "*** Generating test signed offline bundle for ESX 6.x"
	@$(RM) -Rf $(OFFLINE_BUNDLE_STAGE)
	@$(MKDIR) -p $(OFFLINE_BUNDLE_STAGE)
	$(VIBPUBLISH) -g \
		--create-offline-bundle=$(@) \
		--force \
		--bulletin $(VIB_BULLETIN_FILLED) \
		-s $(VIBFILE_TEST_SIGNED) \
		--target ESXi,6.0.0 \
		--vendor VMware \
		--vendor-code vmw \
		-o $(OFFLINE_BUNDLE_STAGE)
endif


publish: $(VIBFILE) $(VIBFILE_SIGNED) $(VIBFILE_TEST_SIGNED) $(OFFLINE_BUNDLE_6x)
	@if [ "$(PROJECT_PUBLISH_DIR)x" == "x" ]; then \
		echo "*** No PROJECT_PUBLISH_DIR set, we're done here"; \
		exit 1; \
	fi;
	@echo "*** Publishing Gobuild component"
	$(MKDIR) -p $(PROJECT_PUBLISH_DIR)
	$(CP) -a \
		$(VIBFILE) \
		$(VIBFILE_SIGNED) \
		$(VIBFILE_TEST_SIGNED) \
		$(OFFLINE_BUNDLE_6x) \
		$(PROJECT_PUBLISH_DIR) > /dev/null 2>&1
	@if [ "$?" = "0" ]; then \
		echo "***  done!"; \
	else \
		echo "***  some bits failed to copy, maybe we didn't end up generating an officially signed VIB?"; \
	fi; \
	/bin/ls -l $(PROJECT_PUBLISH_DIR); \
	exit 0;


clean:
	@echo "*** Cleaning"
	@-cd $(PROJECT_ROOT) && $(NODE) $(GRUNT) clean --force
	@rm -f $(PROJECT_ROOT)/app/config.js
	@rm -Rf $(PROJECT_ROOT)/node_modules
	@rm -f $(PROJECT_ROOT)/bower.json
	@rm -Rf $(BUILDROOT)
	@-find $(CLIENT_APP_ROOT)/bower_components ! -iregex '.git' | xargs rm -Rf
>>>>>>> first commit
