#
# Makefile 
#
# Top-level makefile for ESX vmkdops service and Docker Volume VMDK plugin 
#
# Traverses esx and guest-specific folders and invokes make there. 
# 

# source locations for Make
ESX_SRC     := vmdkops-esxsrv # esx service for docker volume ops
CLIENT_SRC  := src/dvolplug   # Golang code for  docker volume vmdk plugin

.PHONY: build test clean
build test clean: 
	@-mkdir -p $(BIN)
	cd  $(ESX_SRC)   ; $(MAKE) -$(MAKEFLAGS) $@ 
	cd  $(CLIENT_SRC); $(MAKE) -$(MAKEFLAGS) $@

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

# nested makefiles put stuff here. We need to know to use ...
BIN = ./bin

# bin locations on target guest
GLOC := /usr/local/bin


# vib install: we install by file name but remove by internal name
VIBFILE := vmware-esx-vmdkops-1.0.0.vib
VIBNAME := vmware-esx-vmdkops-service
VIBCMD  := localcli software vib


.PHONY: deploy
# ignore failures in copy to guest (can be busy) and remove vib (can be not installed)
deploy: 
	./build.sh
	-cd $(BIN); scp dvolplug $(GUEST):$(GLOC)/
	cd $(BIN); scp $(VIBFILE) $(HOST):/tmp
	-ssh $(HOST) $(VIBCMD) remove --vibname $(VIBNAME)
	ssh  $(HOST) $(VIBCMD) install --no-sig-check  -v /tmp/$(VIBFILE)

cleanremote:
	-ssh $(GUEST) rm $(GLOC)/$(DVOLPLUG)
	-ssh $(HOST) $(VIBCMD) remove --vibname $(VIBNAME)

# "make simpletest" assumes all services are started manually, and simply 
# does sanity check of create/remove docker volume on the guest
TEST_VOL_NAME := MyVolume

.PHONY: simpletest
simpletest:  
	rsh $(GUEST)  docker volume create \
			--driver=vmdk --name=$(TEST_VOL_NAME) \
			-o size=1gb -o policy=good
	rsh $(GUEST)  docker volume ls
	rsh $(GUEST)  docker volume inspect $(TEST_VOL_NAME)
	rsh $(GUEST)  docker volume rm $(TEST_VOL_NAME)
	
