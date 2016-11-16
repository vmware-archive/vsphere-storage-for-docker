# Contributing

## Topics
* [Code contribution guidelines](#code-contribution-guidelines)
* [Bug filing guidelines](#bug-filing-guidelines)
* [Testing and CI](#testing-and-ci)

## Code Contribution guidelines
### Dev Setup and debugging help
Read the [FAQ on the Wiki](https://github.com/vmware/docker-volume-vsphere/wiki#faq)
### Pull Requests
* Create a fork or branch (if you can) and make your changes
   * Branch should be suffixed with github user id: (branch name).(github user id) Example: mydevbranch.kerneltime
   * If you want to trigger CI test runs for pushing into a branch prefix the branch with runci/ Example: runci/mylatestchange.kerneltime
* Each PR must be accompanied with unit/integration tests
* Add detailed description to pull request including reference to issues.
* Add details of tests in "Testing Done".
* Locally run integration tests.
* Squash your commits before you publish pull request.
* If there are any documentation changes then they must be in the same PR.
* We don't have formal coding conventions at this point.
Make sure your code follows same style and convention as existing code.

See  [Typical Developer Workflow](#typical-developer-workflow) to get started.


### Merge Approvals:
* Pull request requires a minimum of 2 approvals, given via "Ship it",
"LGTM" or "+1" comments.
* Author is responsible for resolving conflicts, if any, and merging pull request.
* After merge, you must ensure integration tests pass successfully.
Failure to pass test would result in reverting a change.

Do not hesitate to ask your colleagues if you need help or have questions.
Post your question to Telegram or drop a line to cna-storage@vmware.com

## Documentation

Documentation is published to [GitHub
Pages](https://vmware.github.io/docker-volume-vsphere/) using
[mkdocs](http://www.mkdocs.org/).

1. Documentation is updated each time a release is tagged.
2. The latest documentation for the master can be found in
   [docs](https://github.com/vmware/docker-volume-vsphere/tree/master/docs) in
   markdown format

To update documentation
```
make documentation # Start the mkdocs docker image
mkdocs serve -a 0.0.0.0:8000
# 1. Open browser on same machine and head to 127.0.0.1:8000
# 2. Make edits and refresh browser
# 3. Build the website
mkdocs build
# 4. Checkout the gh-pages
git checkout gh-pages
# 5. Remove the old site and copy the new one.
rm -rvf documentation
mv site documentation
# 6. Push to GitHub
git add documentation
git commit
git push origin gh-pages
```

## Bug filing guidelines
* Search for duplicates!
* Include crisp summary of issue in "Title"
* Suggested template:

```
Environment Details: (#VMs, ESX, Datastore, Application etc)

Steps to Reproduce:
1.
2.
3.

Expected Result:

Actual Result:

Triage:
Copy-paste relevant snippets from log file, code. Use github markdown language for basic formatting.

```
## Typical Developer Workflow
### Build, test, rinse, repeat
Use `make` or `make dockerbuild` to build.

Build environment is described in README.md. The result of the build is a set
of binaries in ./build directory.

In order to test locally, you'd need a test setup. Local test setup automation is planned but but not done yet, so currently the environment has to be set up manually.

Test environment  typically consist of 1 ESX and 2  guest VMs running inside of the
ESX. We also support 1 ESX and 1 guest VM. We require ESX 6.0 and later,
and a Linux VM running  Docker 1.10+ enabled for  plain text TCP connection, i.e.
Docker Daemon running with "-H tcp://0.0.0.0:2375 -H unix:///var/run/docker.sock"
options. Note that both tcp: and unix: need to be present
Please check  "Configuring and running Docker"
(https://docs.docker.com/engine/admin/configuring/)  page on how to configure this - also there is a github link at the bottom, for systemd config files.

If test environment has Docker running on Photon OS VMs, VMs should be configured to accept traffic on port 2375. Edit /etc/systemd/scripts/iptables, and add following rule at the end of the file. Restart iptable service after updating iptables file.

```
#Enable docker connections
iptables -A INPUT -p tcp --dport 2375 -j ACCEPT
```

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
make build-all deploy-all test-all
# only remote test
make testremote
```

If the code needs to run in debugger or the console output is desired.
Login to the machine kill the binary and re-run it manually.

```
Standard invocation on ESX:
python -B /usr/lib/vmware/vmdkops/bin/vmdk_ops.py

Standard invocation on VM: (as root)
/usr/local/bin/docker-volume-vsphere
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
./esx_service/descriptor.xml
```
### git, branch management and pull requests
This section is for developers working directly on our git repo,
and is intended to clarify recommended process for internal developers.

We use git and GitHub, and follow customary process for dev branches and pull
requests. The text below assumes familiarity with git concepts. If you need a
refresher on git, there is plenty of good info on the web,
e.g. http://rogerdudler.github.io/git-guide/is.

The typical source management workflow is as follows:
* in a local repo clone, create a branch off master and work on it
(e.g. `git checkout -b <your_branch> master`)
* when local test is done and it's time to validate with CI tests, push your branch
to GitHub and make sure CI passes
(e.g. `git push origin <your_branch>`)
* when you branch is ready, rebase to latest master
and squash commit if needed (e.g. `git fetch origin; git rebase -i origin/master`).
Each commit should have
a distinct functionality you want to handle as a unit of work. Then re-push your
branch, with `git push origin <your_branch>`, or, if needed,
`git push -f origin <your_branch>`
* create a PR on github using already pushed branch. Please make sure the title,
description and "tested:" section are accurate.
* When applying review comments, create a new commit so the diff can be easily seen
by reviewers.
* When ready to merge (see "Merge Approvals" section above), squash multiple
"review" commits (if any) into one, rebase to master and re-push.
This is done to make sure CI still passes.
* Then merge the PR.
* With any questions/issues about the steps, telegram to cna-storage

## Managing GO Dependencies

Use [gvt](https://github.com/FiloSottile/gvt) and check in the dependency.
Example:
```
make gvt # Start docker image used to build which includes gvt
gvt fetch github.com/docker/go-plugins-helpers/volume
git add vendor
git commit -m "Added new dependency go-plugins-helpers"
exit
```

## Testing and CI

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
