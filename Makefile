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

COPY  := scp
MKDIR := mkdir -p

# bin locations on target machines (guest and host)
GLOC := /usr/local/bin
HLOC := /usr/lib/vmware/vmdkops/bin

# files to copy to guest
GFILES := dvolplug
# files to copy to host
HFILES   := vmci_srv.py libvmci_srv.so mkfs.ext4  

SRV_BIN := vmdkops-esxsrv/payloads/payload1/usr/lib/vmware/vmdkops/bin

.PHONY: deploy
deploy: build
	@echo Copying $(GFILES) to $(GUEST):$(GLOC) ...
	@-$(foreach f,$(GFILES), $(COPY) $(BIN)/$f $(GUEST):$(GLOC)/$f; )
	@rsh $(HOST) $(MKDIR) $(HLOC)
	@echo Copying $(HFILES) to $(HOST):$(HLOC) ...
	@-$(foreach f,$(HFILES), $(COPY) $(SRV_BIN)/$f $(HOST):$(HLOC)/$f; )

cleanremote:
	-ssh $(GUEST) rm $(GLOC)/$(DVOLPLUG)
	-ssh $(ESX)   rm $(HLOC)/$(VMCI_SRV)

# "make simpletest" assumes all services are started manually, and simply 
# does sanity check of create/remove docker volume on the guest
TEST_VOL_NAME := MyVolume

.PHONY: simpletest
simpletest:  
	rsh $(GUEST)  docker volume create \
			--driver=vmdk --name=$(TEST_VOL_NAME) \
			-o size=1tb -o policy=good
	rsh $(GUEST)  docker volume ls
	rsh $(GUEST)  docker volume inspect $(TEST_VOL_NAME)
	rsh $(GUEST)  docker volume rm $(TEST_VOL_NAME)
	
