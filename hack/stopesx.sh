#!/bin/bash
kill `ps -c | grep vmci_srv.py | grep -v grep| awk '{ print $1 }'`
