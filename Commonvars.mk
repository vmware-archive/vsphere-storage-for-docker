# Copyright 2016 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This Makefile's purpose is to hold common variables to reuse across
# Makefiles for vDVS.

# Exporting all variables to reuse
export

# Use the following environment vars to change the behavior to build managed plugin:
#
# DOCKER_HUB_REPO - Dockerhub repo to use, in both name forming and in pushing to dockerhub.
#                   Defaults to the result of `whoami` command.
#                   Note: you first need to run 'docker login' in order for 'make push' to succeed
# VERSION_TAG     - How you want the version to look. Default is "current git tag +1 "
# EXTRA_TAG       - additional info you want in tag. Default is "-dev"
#
# To check resulting settings use  "make info"
# examples:
#   make
#          --> resulting name (for my build as of 3/17/17)  msterin/docker-volume-vsphere:0.13-dev
#
# 	DOCKER_HUB_REPO=vmware EXTRA_TAG= VERSION_TAG=latest make
#           --> resulting name vmware/docker-volume-vsphere:latest
#
#   DOCKER_HUB_REPO=cnastorage EXTRA_TAG=-CI make
#           --> resulting name cnastorage/docker-volume-vspehere:0.13-CI
#

# grab the latest commit SHA and related tag from git so we can construct plugin tag
ifndef VERSION_TAG
GIT_SHA := $(shell git rev-parse --revs-only --short HEAD)
GIT_TAG := $(shell git describe --tags --abbrev=0 $(GIT_SHA))
endif

# Allow these vars to be suplied in environment
DOCKER_HUB_REPO ?= $(shell whoami)
VERSION_TAG ?= $(GIT_TAG).$(GIT_SHA)
EXTRA_TAG ?= -dev

# final tag
PLUGIN_TAG := $(VERSION_TAG)$(EXTRA_TAG)

# plugin name - used as a base for full plugin name and container for extracting rootfs
PLUGIN_NAME ?= $(DOCKER_HUB_REPO)/docker-volume-vsphere
VFILE_PLUGIN_NAME = $(DOCKER_HUB_REPO)/vfile

# Managed plugin alias name
MANAGED_PLUGIN_NAME := "vsphere:latest"
VFILE_MANAGED_PLUGIN_NAME := "vfile:latest"

# build places binaries here:
BIN := ../build
MANAGED_PLUGIN_LOC := ../plugin_dockerbuild
