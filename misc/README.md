This directory contains scripts and other files (e.g. Dockerfiles) needed for
the following: 

* building RPM / DEB and VIB packages for docker-volume-vsphere plugin
* running CI (we use https://drone.io/)

The subdirs are as follows:
* **dockerfiles** - all dockerfiles we use for build / deploy / package
* **drone-scripts** - wrappers used by drone.io to invoke misc. `make` targets
* **scripts** - helper scripts used in build and deploy
