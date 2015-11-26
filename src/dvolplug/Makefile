#
# Makefile for Docker data volume VMDK plugin , client (docker-side) code 
#
# Builds client-side (docker engine) volume plug in code 
# docker-side plugin code
#
# Expectations (todo: check for them, or move all to containers) 
#   Need docker (for containerized go build)
	
.PHONY: first_target
first_target: build
	
# All sources. We rebuild if anything changes here
SRC = plugin.go main.go $(VMDKOPS_SRC)

# Place binaries here (on 'make stage')
BIN := ../../bin

# Binaries to build 
PLUGIN := dvolplug

# vmdkops module we depend on.
# it is handled here to make full build more reliable
VMDKOPS_MODULE = vmware/vmdkops
VMDKOPS_SRC = ../vmware/vmdkops/vmdkops.go


.build_module: .build_container $(VMDKOPS_SRC)
	$(GO) build $(VMDKOPS_MODULE)
	@touch $@

#
# Container and commands for 'go' builds.
#
GO_CONT       = gobld
GO_CONT_LOC   = ./containers/$(GO_CONT)

# locaitons with source code for volume mounts
CONT_SRC_LOC = /go/src
SRC_LOC = $(GOPATH)/src

GO = docker run --rm \
				-w $(CONT_SRC_LOC)/$(PLUGIN) \
				-v $(SRC_LOC):$(CONT_SRC_LOC) $(GO_CONT) go
				
				
.build_container:  $(GO_CONT_LOC)/Dockerfile
	@which docker > /dev/null ; \
	  if [ $$? -ne 0 ] ; then \
	      echo Failed to run: Docker installation required. ; exit 1 ; fi 
	cd $(GO_CONT_LOC) ; docker build -q -t $(GO_CONT):latest .
	@touch $@

#
#  Targets to build plugin
#
.PHONY: build test clean
build: .build_container .build_module $(BIN)/$(PLUGIN)

clean: .build_container
	$(GO) clean $(PLUGIN)  $(VMDKOPS_MODULE)
	rm -f $(DVOLPLUG) $(BIN)/$(PLUGIN) .build_*

#TBD: this is a good place to add unit tests...	
test: build
	$(GO) test $(VMDKOPS_MODULE)
	@echo "*** Info: No tests in plugin folder yet"
     
$(BIN)/$(PLUGIN): $(SRC)
	$(GO) build $(PLUGIN)
	@-mkdir -p $(BIN)
	mv $(PLUGIN) $(BIN)

 
#	
#lint: testdeps
#	$(GO_CONT_RUN) go get -v github.com/golang/lint/golint
#	for file in $$(find . -name '*.go' | grep -v '\.pb\.go' | grep -v '\.pb\.gw\.go' | grep -v '\.pb\.log\.go'); do \
#		$(GO_CONT_RUN) golint $$file; \
#		if [ -n "$$(golint $$file)" ]; then \
#			exit 1; \
#		fi; \
#	done
#	
#testdeps:
#	$(GO_CONT_RUN)  go get -d -v -t ./...


