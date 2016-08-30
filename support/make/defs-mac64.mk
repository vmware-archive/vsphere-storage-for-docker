NPM_EXE := npm
NODE_EXE := node

GRUNT := $(TCROOT)/noarch/grunt-cli-0.1.11/bin/grunt

GCC_DIR := $(TCROOT)/mac32/gcc-4.4.3
GCC := $(GCC_DIR)/bin/i686-linux-gcc
CPP := $(GCC_DIR)/bin/i686-linux-cpp
CC := $(GCC)

MKISOFS := $(TCROOT)/mac32/cdrtools-2.01/bin/mkisofs

CURL_DIR := $(TCROOT)/mac32/curl-7.44.0-openssl-1.0.1p
CURL := $(CURL_DIR)/bin/curl

PYTHON := $(TCROOT)/mac32/python-2.6.1/bin/python

BINUTILS_DIR := $(TCROOT)/mac32/binutils-2.23.52
LD := $(BINUTILS_DIR)/bin/i686-linux-ld

MAKE := $(TCROOT)/mac32/make-3.81/bin/make

GIT_BIN := $(TCROOT)/mac32/git-1.8.5.6/bin
GIT := $(GIT_BIN)/git

RSYNC := $(TCROOT)/mac32/rsync-3.0.7/bin/rsync

SED := $(TCROOT)/mac32/sed-4.1.5/bin/sed
ZIP := $(TCROOT)/mac32/zip-2.32/bin/zip
FIND := $(TCROOT)/mac32/findutils-4.4.0/bin/find

SH := $(TCROOT)/mac32/bash-3.2/bin/bash
COREUTILS := $(TCROOT)/mac32/coreutils-8.6
MKDIR := $(COREUTILS)/bin/mkdir
CP := $(COREUTILS)/bin/cp
ENV := $(COREUTILS)/bin/env
LN := $(COREUTILS)/bin/ln
MV := $(COREUTILS)/bin/mv
RM := $(COREUTILS)/bin/rm
PWD := $(COREUTILS)/bin/pwd
TOUCH:= $(COREUTILS)/bin/touch

TAR := /usr/bin/tar

MAINSRCROOT := $(shell $(PWD))
