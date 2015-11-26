# 
# Makefile for ESX "vmdkops" service - host part of "vmware/vmdkops" Go package
#
# builds the VIB and places it in ../bin
# 
# assumes all pre-reqs are checked by now
#
# Expectations:
#   Need 'apt-get libc6-dev-i386' for 32 bit C headers
#   TODO: check it is installed, or (better) do a container with cc and this

# code
PY_SRC := vmci_srv.py
C_SRC  := server.c

# Shared lib. Note that it is referred by name from vmci_srv.py
VMCI_SRV_LIB = libvmci_srv.so 

# all needed for pythoning , for now 
PYFILES = $(PY_SRC) $(VMCI_SRV_LIB)

# mkfs etc.. to copy to vib
TOOLS := tools/lin32/mkfs.ext4

# deliverables (placed by build)
BIN := ../bin
VIB := vmware-esx-vmdkops-1.0.0.vib

CP := cp -f 
RM := rm -f 

# location on ESX where the stuff needs to go (used in .py code)
ESX_BIN := /usr/lib/vmware/vmdkops/bin

# in-container Src locations for build steps
VIB_CONT_LOC    := /usr/src/vib
VIB_CONT        := lamw/vibauthor

# where the files should be placed for vibauthor
STAGE   = $(PWD)/vib

# payload location - see vib/descriptor.xml
PAYLOAD = $(STAGE)/payloads/payload1

PAYLOAD_BIN = $(PAYLOAD)$(ESX_BIN)

# containers to run commands for build steps
VIBAUTHOR := docker run --rm -v $(STAGE):$(VIB_CONT_LOC) $(VIB_CONT) vibauthor



# C stuff
# Note : vSockets return cartelId to ID connecting VM. We use vsi to convert 
#       it to VM name/ID. ESX is shipped with 32bit VSI python module only, so 
#       we use 32bit python and 32 bit shared libs for VMCI
CC      := gcc
CFLAGS  := -fPIC -m32 -shared

# this is used when manually invoking make with no options
.PHONY: first_target
first_target: build

# this is used from a Makefile upstairs, and could be used manually 
.PHONY: build test clean
build: .build_container $(BIN)/$(VIB)

.build_container:
	@which docker > /dev/null ; \
	  if [ $$? -ne 0 ] ; then \
	      echo Failed to run: Docker installation required. ; exit 1 ; fi 
	docker pull $(VIB_CONT)
	@touch $@

# puts stuff in staging, and also to $(BIN) for manual test
$(BIN)/$(VIB): $(STAGE)/descriptor.xml $(PYFILES) $(TOOLS)
	@mkdir -p $(PAYLOAD_BIN)
	$(CP) $(PYFILES) $(TOOLS) $(PAYLOAD_BIN)
	$(VIBAUTHOR) --debug --compose \
				 --vib=$(VIB_CONT_LOC)/$(VIB) --stage-dir $(VIB_CONT_LOC) \
		  		 --force
	$(CP) vib/$(VIB) $(BIN)
	$(CP) $(PYFILES) $(TOOLS) $(BIN)

$(VMCI_SRV_LIB): $(C_SRC)
	$(CC) $(CFLAGS) -o $@ $(C_SRC)
	
	
clean:
	$(RM) $(BIN)/$(VIB) $(STAGE)/$(VIB) 
	$(RM) $(VMCI_SRV_LIB)
	$(RM) $(PAYLOAD_BIN)/*
	$(RM) $(BIN)/$(PY_SRC) $(BIN)/$(VMCI_SRV_LIB) $(BIN)/mkfs.ext4


test: build
	@echo "*** Info: No unit test here  yet ($(PWD))"	


	