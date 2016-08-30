NPM_EXE := npm.cmd
NODE_EXE := node.exe

AUTOCONF_BIN := /bin
AUTOMAKE_BIN := /bin
NASM_BIN := /bin

GCC := gcc

GRUNT := $(TCROOT)/noarch/grunt-cli-0.1.11/bin/grunt

GIT_BIN := $(TCROOT)/win32/git-1.7.6/bin
GIT := $(GIT_BIN)/git

SED := sed
FIND := $(TCROOT)/win32/findutils-4.2.20-2/bin/find.exe

COREUTILS_BIN := $(TCROOT)/win32/coreutils-5.3.0/bin
MKDIR := $(COREUTILS_BIN)/mkdir.exe
CP := $(COREUTILS_BIN)/cp.exe
ENV := $(COREUTILS_BIN)/env.exe
LN := $(COREUTILS_BIN)/ln.exe
MV := $(COREUTILS_BIN)/mv.exe
RM := $(COREUTILS_BIN)/rm.exe
PWD := $(COREUTILS_BIN)/pwd.exe

MAINSRCROOT := $(shell $(PWD))
