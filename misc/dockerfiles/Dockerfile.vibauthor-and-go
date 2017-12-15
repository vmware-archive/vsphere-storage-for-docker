# This docker file is published using the kerneltime account to cnastorage org.
# To use other account with access to the org update the CI/CD yaml file.
# If any changes are to be made contact cna-storage@vmware.com
# to publish the updates to docker hub.
# This is published to docker hub for 2 reasons
# 1. Drone cannot build this image and then use it without forking
#    a separate repo and adding dependencies between dockerfile
#    and the vsphere-storage-for-docker repo.
# 2. Others outside the project can find this useful.


#
## This container is used for different duties in vsphere-storage-for-docker dev process:
##
## - Build of GO code            => need golang 1.5+
## - Build of C VMCI wrappers    => need gcc and 32 bit libs
## - Build of ESX .VIB package   => need vibauthor
## - Manual investigations etc   => may need wget
## - Manipulating dependencies   => need gvt
## - Controlling VMs under ESX   => may need govc command line
##

# Current tag: 0.12 (used in drone.yml).
# Build the image manually (docker image build -t cnastorage/vibauthor-and-go:<tag> -f <this file> .)

FROM centos:6.6

MAINTAINER cna-storage@vmware.com

# Working directory
WORKDIR /root

RUN yum update -y && \
    yum -y install tar wget openssl python-lxml glibc.i686 git file e2fsprogs gcc  glibc-devel.x86_64 glibc-devel.i686 libgcc.i686 glibc-static  && \
    curl https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz | tar -C /usr/local -xzf - && \
    rm -rf ~/*.rpm ~/*.tar.gz && \
    yum clean all && \
    curl -L https://github.com/vmware/govmomi/releases/download/v0.13.0/govc_linux_amd64.gz | gzip -d > /usr/local/bin/govc && \
    chmod +x /usr/local/bin/govc && \
    curl -O http://download3.vmware.com/software/vmw-tools/vibauthor/vmware-esx-vib-author-5.0.0-0.0.847598.i386.rpm && \
    rpm -ivh vmware-esx-vib-author-5.0.0-0.0.847598.i386.rpm && \
    wget -O jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 && chmod +x jq && cp jq /usr/local/bin


ENV PATH=$PATH:/usr/local/go/bin:/go/bin \
    GOROOT=/usr/local/go \
    GOPATH=/go \
    GO15VENDOREXPERIMENT=1

RUN go get -u github.com/FiloSottile/gvt && \
    go get -u github.com/golang/lint/golint && \
    go get -u gopkg.in/check.v1

RUN curl --silent --location https://rpm.nodesource.com/setup_4.x | bash - && \
    yum -y install nodejs && \
    yum -y install ruby-devel rubygems && \
    npm install -g grunt && \
    gem install compass

RUN curl http://downloads.drone.io/drone-cli/drone_linux_amd64.tar.gz | tar zx
RUN install -t /usr/local/bin drone
