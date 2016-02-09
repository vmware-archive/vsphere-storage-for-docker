# 
# Makefile for ESX "vmdkops" service - host part of "vmware/vmdkops" Go package
#
# Prepares payload folder for vibauthor and builds/places neede stuff there
# We expect the actual vibauthor to be run via dedicated container upstair
# 


# code
PY_SRC := vmci_srv.py
C_SRC  := server.c

# extra stuff to copy to vib
TOOLS := tools/lin32/mkfs.ext4

# Shared lib. Note that it is referred by name from vmci_srv.py
VMCI_SRV_LIB := libvmci_srv.so 

PY_FILES     = $(PY_SRC) $(VMCI_SRV_LIB)
SRC_FILES    = descriptor.xml $(PY_FILES) $(TOOLS)

# location on ESX where the stuff needs to go (used in .py code)
ESX_BIN := /usr/lib/vmware/vmdkops/bin

# payload location - see descriptor.xml
PAYLOAD = payloads/payload1
PAYLOAD_BIN = $(PAYLOAD)$(ESX_BIN)

# C stuff
# Note : vSockets return cartelId to ID connecting VM. We use vsi to convert 
#       it to VM name/ID. ESX is shipped with 32bit VSI python module only, so 
#       we use 32bit python and 32 bit shared libs for VMCI
CC      := gcc
CFLAGS  := -fPIC -m32 -shared


# this is used from a Makefile upstairs, and could be used manually 
.PHONY: build test clean
build: .copy_files


.copy_files: $(SRC_FILES)
	mkdir -p $(PAYLOAD_BIN)
	cp -f  $(PY_FILES) $(TOOLS) $(PAYLOAD_BIN)
	@touch $@


$(VMCI_SRV_LIB): $(C_SRC)
	$(CC) $(CFLAGS) -o $@ $(C_SRC)
	
	
clean:
	rm -fr $(VIB) $(VMCI_SRV_LIB) $(PAYLOAD)

test: build
	@echo "*** Info: No unit test here  yet ($(PWD))"	


	