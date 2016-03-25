# Contributing Code

* Create a fork or branch (if you can) and make your changes
* Push your changes and create a pull request.

# Typical Developer Workflow

Make changes to code and run build. Make will basic unit tests

```
make
```

Build environment is described in README.md. The result of the build is a set
of binaries in ./bin directory.

In order to test locally, you'd need a test setup. Local test setup automation is planned but but not done yet, so currently the environment
has to be set up manually.


Test environment  typically consist of 1 ESX and 2  guest VMs running inside of the
ESX. We also support 1 ESX and 1 guest VM. We require ESX 6.0 and later,
and a Linux VM running  Docker 1.10+ enabled for  plain text TCP connection, i.e.
Docker Daemon running with "-H tcp://0.0.0.0:2375 -H unix:///var/run/docker.sock" 
options. Note that both tcp: and unix: need to be present
Please check  "Configuring and running Docker"
(https://docs.docker.com/engine/admin/configuring/)  page on how to configure this - also there is a github link at the bottom, for systemd config files.

To deploy the plugin and test code onto a test environment we support a set of
Makefile targets. There targets rely on environment variables to point to the
correct environment.

Environment variables:
- You **need** to set ESX and either VM_IP (in which case we'll use 1 VM) or
both VM1 and VM2 environment variables

Examples:
```
# Build and deploy the code. Also deploy (but do not run) tests
ESX=10.20.105.54 VM_IP=10.20.105.201 make deploy-all
```

or

```
# clean, build, deploy, test and clean again
export ESX=10.20.105.54
export VM1=10.20.105.121
export VM2=10.20.104.210

make all
```

Or just put the 'export' statement in your ~/.bash_profile and run

```
# just build
make
# build, deploy, test
make deploy-all testremote
# only test
make testremote
```

If the code needs to run in debugger or the console output is desired.
Login to the machine kill the binary and re-run it manually.

```
Standard invocation on ESX:
python -B /usr/lib/vmware/vmdkops/bin/vmci_srv.py 

Standard invocation on VM: (as root)
/usr/local/bin/docker-vmdk-plugin
```

To remove the code from the testbed, use the same steps as above (i.e define 
ESX, VM1 and VM2) and use the following make targets:

```
# remove stuff from the build
make clean
# remove stuff from ESX
make clean-esx
# remove stuff from VMs
make clean-vm
# clean all (all 3 steps above)
make clean-all
```

If additional python scripts are added to the ESX code, update the vib description file to include them.

```
./vmdkops-esxsrv/descriptor.xml
```

# Managing GO Dependencies

Use [gvt](https://github.com/FiloSottile/gvt) and check in the dependency.
Example:
```
gvt fetch github.com/docker/go-plugins-helpers/volume
git add vendor
git commit -m "Added new dependency go-plugins-helpers"
```

# Testing and CI/CD

The work flow for coding looks like this

- Each checkin into a branch on the official repo will run the full set of
  tests.

- On Pull Requests the full set of tests will be run.
  - The logs for the build and test will be posted to the CI server.
  - Due to security concerns the artifacts will not be published.
    https://github.com/drone/drone/issues/1476

- Each commit into the master operation will run the full set of tests
  part of the CI system.
  - On success a docker image consisting of only the shippable docker
    plugin is posted to docker hub. The image is tagged with latest
    (and <branch>-<build>). Any one can pull this image and run it to
    use the docker plugin.

A typical workflow for a developer should be.

- Create a branch, push changes and make sure tests do not break as reported
  by the CI system.
- When ready post a PR. This will trigger a full set of tests on ESX. After all
  the tests pass and the review is complete the PR will be merged in. If the PR 
  depends on new code checked into master, merge in the changes as a rebase and
  push the changes to the branch.
- When the PR is merged in the CI system will re-run the tests against the master.
  On success a new Docker image will be ready for customers to deploy (This is only
  for the docker plugin, the ESX code needs to be shipped separately).

## CI System

We are using the CI system that has been up by the CNA folks (@casualjim && @frapposelli).
The CI system is based on https://drone.io/

The URL for the server is located https://ci.vmware.run/

There is a webhook setup between github repo and the CI server. The CI server uses
.drone.yml file to drive the CI workflow.

The credentials for Docker Hub deployment is secured using http://readme.drone.io/usage/secrets/

### Running CI/CD on a dev setup
Each developer can run tests part of the CI/CD system locally in their sandbox.

* Setup:
The current CI/CD workflow assumes that the testbed consists of:
   - One ESX
   - 2 Linux VMs running on the same datastore in the ESX.

* Install [Drone CLI](https://github.com/drone/drone-cli)
```
curl http://downloads.drone.io/drone-cli/drone_linux_amd64.tar.gz | tar zx
sudo install -t /usr/local/bin drone
```

* If not already done, checkout source code in $GOPATH
```
mkdir -p $GOPATH/src/github.com/vmware
cd $GOPATH/src/github.com/vmware
git clone https://github.com/vmware/docker-vmdk-plugin.git
```

* Edit the .drone.yml file to reflect the devsetup. If the machine running drone is also one of the target machines, edit the Makefile to not restart docker.

Sample .drone.yml :
```
diff --git a/.drone.yml b/.drone.yml
index 5550172..9a4b872 100644
--- a/.drone.yml
+++ b/.drone.yml
@@ -5,13 +5,13 @@ build:
       # Needed for creating docker images
       - /var/run/docker.sock:/var/run/docker.sock
     environment:
-      - GOVC_USERNAME=$$CI_VMWARE_ESX_USER
-      - GOVC_PASSWORD=$$CI_VMWARE_ESX_PASS
+      - GOVC_USERNAME=root
+      - GOVC_PASSWORD=
       - GOVC_INSECURE=1
-      - GOVC_URL=$$CI_ESX_IP
+      - GOVC_URL=10.20.105.54
     commands:
-      - export VM1=`govc vm.ip ubuntu`
-      - export VM2=`govc vm.ip ubuntu-2`
+      - export VM1=10.20.105.121
+      - export VM2=10.20.104.210
```

Do not restart docker if machine running drone is also a VM in the devsetup.
```
--- a/scripts/deploy-tools.sh
+++ b/scripts/deploy-tools.sh
@@ -151,7 +151,7 @@ function cleanvm {
 
         echo "   Asking docker to remove volumes ($volumes)..."
         # make sure docker engine is not hanging due to old/dead plugins
-        $SSH $target service docker restart
+        #$SSH $target service docker restart
         # and now clean up
         for vol in $volumes
         do
```
 
 .PHONY: clean-esx
 clean-esx:
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
cd cd $GOPATH/src/github.com/vmware/docker-vmdk-plugin/
drone exec --trusted -i ~/.ssh/id_rsa
```
