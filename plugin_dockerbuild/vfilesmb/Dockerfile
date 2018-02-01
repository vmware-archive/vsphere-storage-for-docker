FROM photon
MAINTAINER Project Hatchway <hatchway@vmware.com>

# install required binary and libraries
#ADD usr/ /
#ADD usr/sbin/ /usr/sbin
#ADD usr/bin/ /usr/bin
#ADD usr/lib64/ /usr/lib64/
ADD deps.tar.gz /

# install prerequisites
RUN tdnf install -y nfs-utils python-pip

# Prepare paths
RUN mkdir -p /etc/samba && \
    mkdir -p /var/lib/samba/private && \
    mkdir -p /var/log/samba && \
    mkdir -p /run/samba

# Prepare config file for samba
COPY *.conf /etc/samba/

# Create user for vfile
RUN useradd vfile
RUN echo -e "vfile\nvfile" | smbpasswd -a -s vfile

# Install supervisord
RUN pip install supervisor

# exposes samba's default ports
EXPOSE 137/udp 138/udp 139 445

ENTRYPOINT ["supervisord", "-c", "/etc/samba/supervisord.conf"]
