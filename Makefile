# Makefile
#

# Makefile for Docker data volume VMDK plugin
#
# Builds client-side (docker engine) volume plug in code, and ESX-side VIB
#

# Place binaries here
BIN := ./bin

# esx service for docker volume ops
ESX_SRC     := vmdkops-esxsrv

#  binaries location
PLUGNAME  := docker-vmdk-plugin
PLUGIN_BIN = $(BIN)/$(PLUGNAME)

# all binaries for VMs - plugin and tests
VM_BINS = $(PLUGIN_BIN) $(BIN)/$(VMDKOPS_MODULE).test $(BIN)/$(PLUGNAME).test

VIBFILE := vmware-esx-vmdkops-1.0.0.vib
VIB_BIN := $(BIN)/$(VIBFILE)

# plugin name, for go build
PLUGIN := github.com/vmware/$(PLUGNAME)

GO := GO15VENDOREXPERIMENT=1 go

# make sure we rebuild of vmkdops or Dockerfile change (since we develop them together)
VMDKOPS_MODULE := vmdkops
VMDKOPS_MODULE_SRC = $(VMDKOPS_MODULE)/*.go $(VMDKOPS_MODULE)/vmci/*.[ch]

# All sources. We rebuild if anything changes here
SRC = plugin.go main.go log_formatter.go

#  TargetprintNQuit

# The default build is using a prebuilt docker image that has all dependencies.
.PHONY: dockerbuild build-all
dockerbuild:
	@$(SCRIPTS)/check.sh dockerbuild
	$(SCRIPTS)/build.sh build

build-all: dockerbuild

# The non docker build.
.PHONY: build
build: prereqs code_verify $(VM_BINS)
	@cd  $(ESX_SRC)  ; $(MAKE)  $@

.PHONY: prereqs
prereqs:
	@$(SCRIPTS)/check.sh

$(PLUGIN_BIN): $(SRC) $(VMDKOPS_MODULE_SRC)
	@-mkdir -p $(BIN) && chmod a+w $(BIN)
	$(GO) build --ldflags '-extldflags "-static"' -o $(PLUGIN_BIN) $(PLUGIN)

$(BIN)/$(VMDKOPS_MODULE).test: $(VMDKOPS_MODULE_SRC) $(VMDKOPS_MODULE)/*_test.go
	$(GO) test -c -o $@ $(PLUGIN)/$(VMDKOPS_MODULE)

$(BIN)/$(PLUGNAME).test: $(SRC) *_test.go
	$(GO) test -c -o $@ $(PLUGIN)

.PHONY: clean
clean:
	rm -f $(BIN)/* .build_*
	@cd  $(ESX_SRC)  ; $(MAKE)  $@


# GO Code quality checks.

.PHONY: code_verify
code_verify: lint vet fmt

.PHONY: lint
lint:
	@echo "Running $@"
	${GOPATH}/bin/golint
	${GOPATH}/bin/golint vmdkops
	${GOPATH}/bin/golint config
	${GOPATH}/bin/golint fs

.PHONY: vet
vet:
	@echo "Running $@"
	@go vet *.go
	@go vet vmdkops/*.go
	@go vet config/*.go
	@go vet fs/*.go

.PHONY: fmt
fmt:
	@echo "Running $@"
	gofmt -s -l *.go vmdkops/*.go config/*.go fs/*.go

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

TEST_VM = root@$(VM1)

VM1_DOCKER = tcp://$(VM1):2375
VM2_DOCKER = tcp://$(VM2):2375


SCP := $(DEBUG) scp -o StrictHostKeyChecking=no
SSH := $(DEBUG) ssh -kTax -o StrictHostKeyChecking=no

# bin locations on target guest
GLOC := /usr/local/bin

#
# Scripts to deploy and control services - used from Makefile and from Drone CI
#
# script sources live here. All scripts are copied to test VM during deployment
SCRIPTS     := ./scripts

# scripts started locally to deploy to and clean up test machines
DEPLOY_VM_SH  := $(SCRIPTS)/deploy-tools.sh deployvm
DEPLOY_ESX_SH := $(SCRIPTS)/deploy-tools.sh deployesx
CLEANVM_SH    := $(SCRIPTS)/deploy-tools.sh cleanvm
CLEANESX_SH   := $(SCRIPTS)/deploy-tools.sh cleanesx


#
# Deploy to existing testbed, Expects ESX VM1 and VM2 env vars
#
.PHONY: deploy-esx deploy-vm deploy deploy-all deploy
deploy-esx:
	$(DEPLOY_ESX_SH) "$(ESX)" "$(VIB_BIN)"

VMS= $(VM1) $(VM2)

# deploys to "GLOC" on vm1 and vm2
deploy-vm:
	$(DEPLOY_VM_SH) "$(VMS)" "$(VM_BINS)" $(GLOC) 

deploy-all: deploy-esx deploy-vm
deploy: deploy-all

#
# 'make test' or 'make testremote'
# CAUTION: FOR NOW, RUN testremote DIRECTLY VIA 'make'. DO NOT run via './build.sh'
#  reason: ssh keys for accessing remote hosts are not in container used by ./build.sh
# ----------


# this is a set of unit tests run on build machine
.PHONY: test
test:
	$(SCRIPTS)/build.sh testasroot

.PHONY: testasroot
testasroot:
	$(GO) test $(PLUGIN)/vmdkops
	$(GO) test $(PLUGIN)/config

# does sanity check of create/remove docker volume on the guest
TEST_VOL_NAME ?= TestVolume
TEST_VERBOSE   = -test.v

CONN_MSG := "Please make sure Docker is running and is configured to accept TCP connections"
.PHONY: checkremote
checkremote:
	$(SSH) $(TEST_VM) docker -H $(VM1_DOCKER) ps > /dev/null 2>/dev/null || \
		(echo VM1 $(VM1): $(CONN_MSG) ; exit 1)
	$(SSH) $(TEST_VM) docker -H $(VM2_DOCKER) ps > /dev/null 2>/dev/null || \
		(echo VM2 $(VM2): $(CONN_MSG); exit 1)

.PHONY: test-vm test-esx test-all testremote
# test-vm runs GO unit tests and plugin test suite on a guest VM
# expects binaries to be deployed ot the VM and ESX (see deploy-all target)
test-vm: checkremote
	$(SSH) $(TEST_VM) $(GLOC)/$(VMDKOPS_MODULE).test $(TEST_VERBOSE)
	$(SSH) $(TEST_VM) $(GLOC)/$(PLUGNAME).test $(TEST_VERBOSE) \
		-v $(TEST_VOL_NAME) \
		-H1 $(VM1_DOCKER) -H2 $(VM2_DOCKER)

# test-esx is a quick unittest for Python.
# Deploys, runs and clean unittests on ESX
TAR  = $(DEBUG) tar
ECHO = $(DEBUG) echo
test-esx:
	$(TAR) cz --no-recursion $(ESX_SRC)/*.py | $(SSH) root@$(ESX) "cd /tmp; $(TAR) xz"
	$(ECHO) Running unit tests for vmdk-opsd python code on $(ESX)...
	$(SSH) root@$(ESX) "python /tmp/$(ESX_SRC)/vmdk_ops_test.py"
	$(SSH) root@$(ESX) rm -rf /tmp/$(ESX_SRC)

testremote: test-esx test-vm 
test-all:  test testremote

.PHONY:clean-vm clean-esx clean-all
clean-vm:
	$(CLEANVM_SH) "$(VMS)" "$(VM_BINS)" "$(GLOC)"  "$(TEST_VOL_NAME)"

clean-esx:
	$(CLEANESX_SH) "$(ESX)" vmware-esx-vmdkops-service

clean-all: clean clean-vm clean-esx

# full circle
all: clean-all build-all deploy-all test-all clean-all

