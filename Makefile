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
PNAME  := docker-vmdk-plugin
PLUGIN_BIN = $(BIN)/$(PNAME)

VIBFILE := vmware-esx-vmdkops-1.0.0.vib
VIB_BIN := $(BIN)/$(VIBFILE) 

# plugin name, for go build
PLUGIN := github.com/vmware/$(PNAME)

GO := GO15VENDOREXPERIMENT=1 go 

# make sure we rebuild of vmkdops or Dockerfile change (since we develop them together)
EXTRA_SRC = vmdkops/*.go 

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
	
#
# 'make deploy'
# ----------
# temporary goal to simplify my deployments and sanity check test (create/delete)
#
# expectations: 
#   Need target machines (ESX/Guest) to have proper ~/.ssh/authorized_keys


# msterin's ESX host and ubuntu guest current IPs
HOST  := root@10.20.104.35
GUEST := root@10.20.105.74

# bin locations on target guest
GLOC := /usr/local/bin


# vib install: we install by file name but remove by internal name
VIBNAME := vmware-esx-vmdkops-service
VIBCMD  := localcli software vib


.PHONY: deploy
# ignore failures in copy to guest (can be busy) and remove vib (can be not installed)
deploy: 
	-scp $(PLUGIN_BIN) $(GUEST):$(GLOC)/
	scp $(VIB_BIN) $(HOST):/tmp
	-ssh $(HOST) $(VIBCMD) remove --vibname $(VIBNAME)
	ssh  $(HOST) $(VIBCMD) install --no-sig-check  -v /tmp/$(VIBFILE)

.PHONY: cleanremote
cleanremote:
	-ssh $(GUEST) rm $(GLOC)/$(DVOLPLUG)
	-ssh $(HOST) $(VIBCMD) remove --vibname $(VIBNAME)

# "make simpletest" assumes all services are started manually, and simply 
# does sanity check of create/remove docker volume on the guest
TEST_VOL_NAME := MyVolume

.PHONY: testremote
testremote:  
	rsh $(GUEST)  docker volume create \
			--driver=vmdk --name=$(TEST_VOL_NAME) \
			-o size=1gb -o policy=good
	rsh $(GUEST)  docker volume ls
	rsh $(GUEST)  docker volume inspect $(TEST_VOL_NAME)
	rsh $(GUEST)  docker volume rm $(TEST_VOL_NAME)
	
