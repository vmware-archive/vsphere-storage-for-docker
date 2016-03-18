# Makefile
#

# Makefile for Docker data volume VMDK plugin
#
# Builds client-side (docker engine) volume plug in code, and ESX-side VIB
#

# Place binaries here
BIN := ./bin

# source locations for Make
ESX_SRC     := vmdkops-esxsrv # esx service for docker volume ops

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
.PHONY: dockerbuild
dockerbuild:
	@$(SCRIPTS)/check.sh dockerbuild
	$(SCRIPTS)/build.sh build

# The non docker build.
.PHONY: build
build: prereqs code_verify $(VM_BINS)
	@cd  $(ESX_SRC)  ; $(MAKE)  $@

.PHONY: prereqs
prereqs:
	@$(SCRIPTS)/check.sh

$(PLUGIN_BIN): $(SRC) $(VMDKOPS_MODULE_SRC)
	@-mkdir -p $(BIN)
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
#   set ESX_IP and VM_IP (and VM1_IP / VM2_IP) as env. vars,
# or pass on command line
# 	make deploy-esx ESX_IP=10.20.105.54
# 	make deploy-vm  VM_IP=10.20.105.121
# 	make testremote  ESX_IP=10.20.105.54 VM1_IP=10.20.105.121 VM2_IP=10.20.105.122

ESX_IP ?= 10.20.105.54
VM1_IP ?= 10.20.105.121
VM2_IP ?= $(VM1_IP)

ESX ?= root@$(ESX_IP)
VM1 ?= root@$(VM1_IP)
VM2 ?= root@$(VM2_IP)
VM  ?= $(VM1)

VM1_DOCKER = tcp://$(VM1_IP):2375
VM2_DOCKER = tcp://$(VM2_IP):2375


SCP := scp -o StrictHostKeyChecking=no
SSH := ssh -kTax -o StrictHostKeyChecking=no

# bin locations on target guest
GLOC := /usr/local/bin

#
# Scripts to deploy and control services - used from Makefile and from Drone CI
#
# script sources live here. All scripts are copied to test VM during deployment
SCRIPTS     := ./scripts

# scripts started locally to deploy to and clean up test machines
DEPLOY_VM_SH  := $(SCRIPTS)/deployvm.sh
DEPLOY_ESX_SH := $(SCRIPTS)/deployesx.sh
CLEANVM_SH    := $(SCRIPTS)/cleanvm.sh
CLEANESX_SH   := $(SCRIPTS)/cleanesx.sh


# A helper to validate environment variables settings.
.PHONY: check-test-ip
check-test-ip:
ifeq "$(origin ESX_IP)" "undefined"
	$(error ESX_IP environment needs to be set to point to test ESX IP address)
endif 
ifndef VM1_IP
	$(error VM1_IP environment needs to be set to point to test VM IP address)
endif
ifndef VM2_IP
	$(error VM2_IP environment needs to be set to point to test VM IP address)
endif

#
# Deploy to existing testbed, Expects ESX_IP VM1_IP and VM2_IP env vars
#
.PHONY: deploy deploy-esx deploy-vm
deploy-esx: check-test-ip
	$(DEPLOY_ESX_SH) $(VIB_BIN) $(ESX_IP)

VM_IPS= $(VM1_IP) $(VM2_IP)

# deploys to "GLOC" on vm1 and vm2
deploy-vm: check-test-ip
	$(DEPLOY_VM_SH) "$(VM_BINS)" $(GLOC) "$(VM_IPS)"

deploy: deploy-esx deploy-vm


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
	@$(SSH) $(VM1) docker -H $(VM1_DOCKER) ps > /dev/null 2>/dev/null || \
		(echo VM1 $(VM1_IP): $(CONN_MSG) ; exit 1)
	@$(SSH) $(VM1) docker -H $(VM2_DOCKER) ps > /dev/null 2>/dev/null || \
		(echo VM2 $(VM2_IP): $(CONN_MSG); exit 1)

.PHONY: testremote
testremote: checkremote
	$(SSH) $(VM1) $(GLOC)/$(VMDKOPS_MODULE).test $(TEST_VERBOSE)
	$(SSH) $(VM1) $(GLOC)/$(PLUGNAME).test $(TEST_VERBOSE) \
		-v $(TEST_VOL_NAME) \
		-H1 $(VM1_DOCKER) -H2 $(VM2_DOCKER)

.PHONY:clean-vm clean-esx
clean-vm: check-test-ip
	$(CLEANVM_SH) "$(VM_BINS)" $(GLOC)  $(TEST_VOL_NAME) "$(VM_IPS)"

clean-esx: check-test-ip
	$(CLEANESX_SH) "$(ESX_IP)"


# helper goals - save typing in manual passes
clean-all: clean clean-vm clean-esx
all: dockerbuild deploy testremote
all-vm: dockerbuild deploy-vm testremote

