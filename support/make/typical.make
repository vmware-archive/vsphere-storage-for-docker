ifeq ($(OS), Windows_NT)
   OSTYPE=windows
else
   UNAME=$(shell test -e /bin/uname && echo "/bin/uname" || echo "/usr/bin/uname")
   OS=$(shell $(UNAME))
   ifeq ($(OS), Darwin)
      OSTYPE=mac
      MACOS=1
   else
      OSTYPE=linux
   endif
endif

export OSTYPE

ifndef GOBUILD_AUTO_COMPONENTS_HOSTTYPE
   ifeq ($(OSTYPE), windows)
      TCROOT ?= T:/build/toolchain
      GOBUILD_AUTO_COMPONENTS_HOSTTYPE=windows-2008
      ARCH=win64
   else ifeq ($(OSTYPE), mac)
      TCROOT ?= /build/toolchain
      GOBUILD_AUTO_COMPONENTS_HOSTTYPE=linux-centos64-64
      ARCH=mac64
   else
      TCROOT ?= /build/toolchain
      GOBUILD_AUTO_COMPONENTS_HOSTTYPE=linux-centos64-64
      ARCH=lin64
   endif
endif
export ARCH
export GOBUILD_AUTO_COMPONENTS_HOSTTYPE
export TCROOT

include support/make/defs-${ARCH}.mk

export BUILD_NUMBER ?= 00000
export BUILDROOT ?= $(MAINSRCROOT)/build
export PUBLISH_DIR ?= ${BUILDROOT}/publish
export NPMCACHE = ${BUILDROOT}/npmcache

GOBUILD_AUTO_COMPONENTS ?= 1
OBJDIR ?= obj

VMW_NPM_REGISTRY = https://build-artifactory.eng.vmware.com/artifactory/api/npm/npm-remote

PROJECT_PUBLISH_DIR = $(PUBLISH_DIR)
PROJECT_BUILD_ROOT = $(BUILDROOT)/$(PROJECT)
