
.PHONY: build build-all clean

build-ui build build-all:
	@echo "running $@ for the ui"
	@echo "installing ui build dependencies"
	npm install
	@echo "building the ui"
	grunt

clean clean-ui:
	@echo "running $@ for the ui"
	@echo "removing ui build"
	rm -rf ./build
	@echo "removing ui build dependencies"
	rm -rf ./node_modules

default: build-all
