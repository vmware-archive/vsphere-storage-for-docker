# Dockerile for packaging https://github.com/vmware/docker-volume-vsphere as
# Docker managed plugin.
#
# Image created with this file is used to unpack to plugin rootfs and then build
# plugin image
#
# We need <fs>progs to allow formatting fresh disks from within the plugin


FROM alpine:3.5

RUN apk update ; apk add e2fsprogs xfsprogs
RUN mkdir -p /mnt/global
COPY docker-volume-global /usr/bin
CMD ["/usr/bin/docker-volume-global"]
