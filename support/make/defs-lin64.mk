NPM_EXE := npm
NODE_EXE := node

GRUNT := $(TCROOT)/noarch/grunt-cli-0.1.11/bin/grunt
RUBY := $(TCROOT)/lin32/ruby-1.9.2-p290
RUBY_BIN := $(RUBY)/bin

AWK := $(TCROOT)/lin64/gawk-3.1.5/bin/awk

MKISOFS := $(TCROOT)/lin64/cdrtools-2.01/bin/mkisofs

NASM_BIN := $(TCROOT)/lin64/nasm-2.09.10/bin
AUTOCONF_BIN := $(TCROOT)/lin64/autoconf-2.69/bin
AUTOMAKE_BIN := $(TCROOT)/lin64/automake-1.14.1/bin

FINDUTILS_BIN := $(TCROOT)/lin64/findutils-4.2.27/bin
FIND := $(FINDUTILS_BIN)/find
XARGS := $(FINDUTILS_BIN)/xargs

GCC_DIR := $(TCROOT)/lin32/gcc-4.8.0
GCC := $(GCC_DIR)/bin/x86_64-linux5.0-gcc
CXX := $(GCC_DIR)/bin/x86_64-linux5.0-g++
LD := $(GCC_DIR)/bin/x86_64-linux5.0-g++
CPP := $(GCC_DIR)/bin/x86_64-linux5.0-cpp
#LD := $(TCROOT)/lin32/binutils-2.22/x86_64-linux5.0/bin/ld
CC := $(GCC)

CURL_DIR := $(TCROOT)/lin64/curl-7.44.0-openssl-1.0.1p
CURL := $(CURL_DIR)/bin/curl

GIT_BIN := $(TCROOT)/lin64/git-1.8.5.6/bin
GIT := $(GIT_BIN)/git

PYTHON := $(TCROOT)/lin64/python-2.7.9-openssl1.0.1p/bin/python

RSYNC := $(TCROOT)/lin32/rsync-3.0.7/bin/rsync

MAKE := $(TCROOT)/lin32/make-3.81/bin/make

SED := $(TCROOT)/lin32/sed-4.1.5/bin/sed
ZIP := $(TCROOT)/lin32/zip-3.0/bin/zip
FIND := $(TCROOT)/lin32/findutils-4.4.0/bin/find

SH := $(TCROOT)/lin64/bash-4.1/bin/bash
COREUTILS := $(TCROOT)/lin64/coreutils-8.6
MKDIR := $(COREUTILS)/bin/mkdir
CP := $(COREUTILS)/bin/cp
ENV := $(COREUTILS)/bin/env
LN := $(COREUTILS)/bin/ln
MV := $(COREUTILS)/bin/mv
RM := $(COREUTILS)/bin/rm
PWD := $(COREUTILS)/bin/pwd
TOUCH := $(COREUTILS)/bin/touch

TAR := $(TCROOT)/lin64/tar-1.23/bin/tar
GZIP_BIN := $(TCROOT)/lin64/gzip-1.5/bin

UNZIP := $(TCROOT)/lin64/unzip-6.0/bin/unzip

MAINSRCROOT := $(shell $(PWD))
COMPASS_BIN := $(shell $(PWD))
