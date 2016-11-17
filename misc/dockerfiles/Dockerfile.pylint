#
# Dockerfile for running pylint
#
FROM alpine
MAINTAINER "CNA Storage Team" <cna-storage@vmware.com>

RUN apk add --update --progress make wget python git
RUN wget --no-check-certificate -O - https://bootstrap.pypa.io/get-pip.py | python 
RUN pip install --upgrade pip pylint pyvmomi pyvim 
