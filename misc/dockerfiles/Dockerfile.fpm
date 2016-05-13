FROM ruby:latest
MAINTAINER cna-storage@vmware.com

RUN apt-get update && \
    apt-get install -y npm php-pear python-setuptools rpm && \
    rm -rf /var/lib/apt/lists/* && \
    apt-get autoclean && \
    gem install fpm
