# CI/CD Overview

## Testing on CI system

The work flow for testing proposed code looks like this

- Each checkin into a branch on the official repo will run the full set of
  tests.

- On Pull Requests the full set of tests will be run.
  - The logs for the build and test will be posted to the CI server.
  - Due to security concerns the artifacts will not be published.
    https://github.com/drone/drone/issues/1476

- Each commit into the master branch will run the full set of tests
  as part of the CI system.
  - On success a docker image consisting of only the shippable docker
    plugin is posted to docker hub. The image is tagged with latest
    (and `<branch>-<build>`). Any one can pull this image and run it to
    use the docker plugin.

## CI System

We are using the CI system that has been supported by the CNA folks (@casualjim, @frapposelli & @mhagen-vmware).
The CI system is based on https://drone.io/  URL for the server is  https://ci.vmware.run/
To be able to change the integration between CI and GitHub, first become admin using the self service portal.

Behind the firewall there is a HW node running ESX that hosts 2 ESX VMs (v6.0u2 and v6.5). Each ESX VM in turn contains a VSAN datastore as well as a VMFS datastore with 2 VMs per datastore (VMs are reused to form swarm cluster). CI testbed is also configured to have swarm cluster and there are 3 VMs participating in the swarm cluster (2 VMs created on vmfs datastore and 1 VM created on vsan datastore).

There is a webhook setup between github repo and the CI server. The CI server uses
.drone.yml file to drive the CI workflow.

The credentials for Docker Hub deployment is secured using http://readme.drone.io/usage/secrets/

### Known issues

When there is a failure unrelated to your PR, you may want to *Restart* the failed CI run instead of pushing dummy commit to trigger CI run.

Here are some known issues with current CI/CD system, if you run into any of the following **please** hit *Restart* button to re-run tests against your PR.

- [Issue#1436](https://github.com/vmware/docker-volume-vsphere/issues/1436): Deploy esx failed with error: Failed to create ramdisk vibtransaction

```
=> Deploying to ESX root@192.168.31.62 Sat Jun 17 04:26:23 UTC 2017

Errors:
 [InstallationError]
 Failed to create ramdisk vibtransaction
 Please refer to the log file for more details.

=> deployESXInstall: Installation hit an error on root@192.168.31.62 Sat Jun 17 04:26:32 UTC 2017


make[1]: *** [deploy-esx] Error 2
make: *** [deploy-esx] Error 2
=> Build + Test not successful Sat Jun 17 04:26:32 UTC 2017
```

### Running CI/CD on a dev setup
Each developer can run tests part of the CI/CD system locally in their sandbox. This section describes how to configure your development setup for running CI/CD. If you are looking for configuring local testbed setup, **please refer** [Setting up local testbed](#local-testbed-setup).

* Setup:
The current CI/CD workflow assumes that the testbed consists of:
   - One ESX
   - 3 Linux VMs running on the same datastore in the ESX.

* Install [Drone CLI](https://github.com/drone/drone-cli)
```
curl http://downloads.drone.io/drone-cli/drone_linux_amd64.tar.gz | tar zx
sudo install -t /usr/local/bin drone
```

* If not already done, checkout source code in $GOPATH
```
mkdir -p $GOPATH/src/github.com/vmware
cd $GOPATH/src/github.com/vmware
git clone https://github.com/vmware/docker-volume-vsphere.git
```

* Setup ssh keys on linux nodes & ESX

Linux:
```
export NODE=root@10.20.105.54
cat ~/.ssh/id_rsa.pub | ssh $NODE  "mkdir -p ~/.ssh && cat >>  ~/.ssh/authorized_keys"
```

ESX:
```
cat ~/.ssh/id_rsa.pub | ssh $NODE " cat >> /etc/ssh/keys-root/authorized_keys"
```
Test SSH keys, login form the drone node should not require typing in a password.

* Run drone exec

```
cd $GOPATH/src/github.com/vmware/docker-volume-vsphere/
drone exec --trusted --yaml .drone.dev.yml -i ~/.ssh/id_rsa -e VM1=<ip VM1> -e VM2=<ip VM2> -e ESX=<ip ESX>
```