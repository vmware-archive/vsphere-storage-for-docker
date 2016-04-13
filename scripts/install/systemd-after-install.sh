#!/bin/bash
systemctl daemon-reload
systemctl enable docker-vmdk-plugin.service
systemctl start docker-vmdk-plugin.service
