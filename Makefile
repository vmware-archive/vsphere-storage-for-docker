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
PLUGIN_NAME  := docker-vmdk-plugin
PLUGIN_BIN = $(BIN)/$(PLUGIN_NAME)

VIBFILE := vmware-esx-vmdkops-1.0.0.vib
VIB_BIN := $(BIN)/$(VIBFILE)

# plugin name, for go build
PLUGIN := github.com/vmware/$(PLUGIN_NAME)

GO := GO15VENDOREXPERIMENT=1 go

# make sure we rebuild of vmkdops or Dockerfile change (since we develop them together)
VMDKOPS_MODULE := vmdkops
EXTRA_SRC      = $(VMDKOPS_MODULE)/*.go $(VMDKOPS_MODULE)/vmci/*.[ch]

# All sources. We rebuild if anything changes here
SRC = plugin.go main.go

#  Targets
#
.PHONY: build
build: prereqs $(PLUGIN_BIN)
	@cd  $(ESX_SRC)  ; $(MAKE)  $@

.PHONY: prereqs
prereqs:
	@./check.sh

$(PLUGIN_BIN): $(SRC) $(EXTRA_SRC)
	@-mkdir -p $(BIN)
	$(GO) build --ldflags '-extldflags "-static"' -o $(PLUGIN_BIN) $(PLUGIN)

.PHONY: clean
clean:
	rm -f $(BIN)/* .build_*
	@cd  $(ESX_SRC)  ; $(MAKE)  $@


#TBD: this is a good place to add unit tests...
.PHONY: test
test: build
	$(GO) test $(PLUGIN)/vmdkops $(PLUGIN)
	@echo "*** Info: No tests in plugin folder yet"
	@cd  $(ESX_SRC)  ; $(MAKE)  $@


# either pass the ESX and VM
# make deploy-esx root@10.20.105.54
# make deploy-vm root@10.20.105.121
# or edit the entries below

ESX?=root@10.20.105.54
VM?=root@10.20.105.121
VM2?=$(VM1) # run tests only on one VM
SCP := scp -o StrictHostKeyChecking=no
SSH := ssh -o StrictHostKeyChecking=no

# bin locations on target guest
GLOC := /usr/local/bin


# vib install: we install by file name but remove by internal name
VIBNAME := vmware-esx-vmdkops-service
VIBCMD  := localcli software vib
STARTESX := startesx.sh
STOPESX  := stopesx.sh
STARTVM := startvm.sh
STOPVM  := stopvm.sh
STARTVM_LOC := ./hack/$(STARTVM)
STOPVM_LOC  := ./hack/$(STOPVM)
STARTESX_LOC := ./hack/$(STARTESX)
STOPESX_LOC  := ./hack/$(STOPESX)

.PHONY: deploy-esx
# ignore failures in copy to guest (can be busy) and remove vib (can be not installed)
deploy-esx:
	$(SCP) $(VIB_BIN) $(ESX):/tmp
	$(SCP) $(STARTESX_LOC) $(STOPESX_LOC) $(ESX):/tmp
	-$(SSH) $(ESX) $(VIBCMD) remove --vibname $(VIBNAME) < /dev/null
	$(SSH) $(ESX) $(VIBCMD) install --no-sig-check  -v /tmp/$(VIBFILE) < /dev/null
	$(SSH) $(ESX) "sh /tmp/$(STARTESX)" < /dev/null

.PHONY: deploy-vm
deploy-vm:
	$(SCP) $(PLUGIN_BIN) $(VM):$(GLOC)
	$(SCP) $(STARTVM_LOC) $(STOPVM_LOC) $(VM):/tmp/
	$(SSH) $(VM) "sh /tmp/$(STARTVM) &" < /dev/null

# "make simpletest" assumes all services are started manually, and simply
# does sanity check of create/remove docker volume on the guest
TEST_VOL_NAME?=MyVolume

TEST_DATA?=$(shell date -u)

.PHONY: testremote
testremote:
	$(SSH) $(VM) docker volume create \
			--driver=vmdk --name=$(TEST_VOL_NAME) \
			-o size=1gb -o policy=good < /dev/null
	$(SSH) $(VM) docker run --rm -v $(TEST_VOL_NAME):/$(TEST_VOL_NAME) busybox touch /$(TEST_VOL_NAME)/file </dev/null
	$(SSH) $(VM2) docker run --rm -v $(TEST_VOL_NAME):/$(TEST_VOL_NAME) --volume-driver=vmdk busybox stat /$(TEST_VOL_NAME)/file </dev/null
	$(SSH) $(VM) docker volume ls </dev/null
	$(SSH) $(VM) docker volume inspect $(TEST_VOL_NAME) </dev/null
	$(SSH) $(VM) docker volume rm $(TEST_VOL_NAME) </dev/null

.PHONY: clean-vm
clean-vm:
	-$(SSH) $(VM) docker volume rm $(TEST_VOL_NAME) < /dev/null
	-$(SSH) $(VM) rm -rvf /mnt/vmdk/$(TEST_VOL_NAME) < /dev/null
	-$(SSH) $(VM) docker volume rm $(TEST_VOL_NAME) < /dev/null
	-$(SCP) $(STOPVM_LOC) $(VM):/tmp/
	-$(SSH) $(VM) "sh /tmp/$(STOPVM)" < /dev/null
	-$(SSH) $(VM) rm $(GLOC)/$(PLUGIN_NAME) < /dev/null
	-$(SSH) $(VM) rm /tmp/$(STARTVM) /tmp/$(STOPVM) < /dev/null
	-$(SSH) $(VM) rm -rvf /mnt/vmdk/$(TEST_VOL_NAME) < /dev/null
	-$(SSH) $(VM) docker volume rm $(TEST_VOL_NAME) < /dev/null # delete any local datavolumes 
	-$(SSH) $(VM) service docker restart < /dev/null

.PHONY: clean-esx
clean-esx:
	-$(SCP) $(STOPESX_LOC) $(ESX):/tmp
	-$(SSH) $(ESX) "sh /tmp/$(STOPESX)" < /dev/null
	-$(SSH) $(ESX) "rm -v /tmp/$(STARTESX) /tmp/$(STOPESX)" < /dev/null
	-$(SSH) $(ESX) $(VIBCMD) remove --vibname $(VIBNAME) < /dev/null
	-$(SSH) $(ESX) "rm -v /tmp/$(VIBFILE)" < /dev/null
