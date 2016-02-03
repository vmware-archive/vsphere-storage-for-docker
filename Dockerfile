#
# Dockerfile for docker-vmdk-plugin build system
#
# uses 'go build' with docker plugin helpers preinstalled
# Dockerized golang info: https://github.com/docker-library/golang
#
# To run make (which compiles Go and C),  run something like that: 
#   docker build -t bldContainer .
#   docker run --rm -u `id -u` -v $PWD:/work -w /work bldContainer make 
#

FROM  golang:latest

MAINTAINER cna-storage@vmware.com

# Defining this arg (and it's default) explicitly allows to use
# docker run --build-arg WHO=<your-name> if building from a repo fork
# 
ARG WHO=vmware

# install vmkdops module in the container
# we copy to /go/src to avoid "go get", which does not work well with private 
# repos inside dockerfiles. Note that GOPATH=/go in the container.
#
COPY vmdkops /go/src/github.com/${WHO}/docker-vmdk-plugin/vmdkops 
RUN go install  github.com/${WHO}/docker-vmdk-plugin/vmdkops
RUN go get github.com/docker/go-plugins-helpers/volume

# we need 32bit headers for ESX-side shared libs
RUN apt-get update && apt-get install -y libc6-dev-i386


