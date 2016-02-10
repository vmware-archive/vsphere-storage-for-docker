#
# Dockerfile for docker-vmdk-plugin build system
#
#

FROM  golang:latest

MAINTAINER cna-storage@vmware.com

ENV GO15VENDOREXPERIMENT=1

RUN apt-get update && apt-get install -y libc6-dev-i386
