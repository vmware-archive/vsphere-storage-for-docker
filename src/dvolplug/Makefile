#
# Makefile for Docker data volume VMDK plugin , client (docker-side) code 
#
# Builds client-side (docker engine) volume plug in code 
# docker-side plugin code
#
# Expectations (todo: check for them, or move all to containers) 
#   Need docker (for containerized go build)

# Binaries to build 
PLUGIN := dvolplug
# Place binaries here
BIN := ../../bin

# make sure we rebuild of vmkdops change (since we develop them together)
VMDKOPS_SRC = ../../vmdkops/*.go

# All sources. We rebuild if anything changes here
SRC = plugin.go main.go $(VMDKOPS_SRC)

#  Targets to build plugin
#
.PHONY: build test clean
build: $(BIN)/$(PLUGIN)
     
$(BIN)/$(PLUGIN): $(SRC) 
	go build -o $@ .
	@-mkdir -p $(BIN)

clean: 
	go clean $(PLUGIN)
	rm -f $(BIN)/$(PLUGIN) .build_*

#TBD: this is a good place to add unit tests...	
test: build
	@echo "*** Info: No tests in plugin folder yet"

