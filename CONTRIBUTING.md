## Code Contribution guidelines
### Dev Setup and debugging help
Read the [FAQ on the Wiki](https://github.com/vmware/docker-volume-vsphere/wiki#faq)
### Pull Requests
* Create a fork or branch (if you can) and make your changes
   * Branch should be suffixed with github user id: `(branch name).(github user id)` Example: `mydevbranch.kerneltime`
   * If you want to trigger CI test runs for pushing into a branch prefix the branch with `runci/` Example: `runci/mylatestchange.kerneltime`
   * If you want to skip running CI test e.g. **Documentation change/Inline comment change** or when the change is in `WIP` or `PREVIEW` phase, add `[CI SKIP]` or `[SKIP CI]` to the PR title.
* Each PR should be accompanied with unit/integration tests whenever it is possible.
* Add detailed description to pull request including reference to issues.
* Add details of tests in "Testing Done".
* Locally run integration tests.
* Squash your commits before you publish pull request.
* If there are any documentation changes then they must be in the same PR.
* We don't have formal coding conventions at this point.
* Add `[WIP]` or `[PREVIEW]` to the PR title to let reviewers know that the PR is not ready to be merged.

#### Documentation PR
Any changes to `[master]/docs` will not be promoted to `gh-pages` (document release branch). The process is not automated yet and you need to copy them manually. Please use following steps and generate the separate PR to merge changes to `gh-pages`.

```
1. Checkout the local gh-pages branch
e.g. git checkout -b localdocs vmware/gh-pages

2. Copy markdown file changes to `gh-pages/jekyll-docs` respective markdown file.
e.g. any changes to `[master]/docs/user-guide/admin-cli.md` should be copied to `gh-pages/jekyll-docs/admin-cli.md`

3. When ready post a PR
```

**Note**: Make sure your code follows same style and convention as existing code.

See  [Typical Developer Workflow](#typical-developer-workflow) to get started.


### Merge Approvals:
* Pull request requires a minimum of 2 approvals, given via "Ship it",
"LGTM" or "+1" comments.
* Author is responsible for resolving conflicts, if any, and merging pull request.
* After merge, you must ensure integration tests pass successfully.
Failure to pass test would result in reverting a change.

Do not hesitate to ask your colleagues if you need help or have questions.
Post your question to Telegram or drop a line to cna-storage@vmware.com

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

A typical workflow for a developer should be.

- Create a branch, push changes and make sure tests do not break as reported
  by the CI system.
- When ready post a PR, it will trigger a full set of tests on ESX. After all
  the tests pass and the review is complete the PR will be merged in. If the PR
  depends on new code checked into master, merge in the changes as a rebase and
  push the changes to the branch.
- When there is a failure unrelated to your PR, you may want to *Restart* the failed CI run instead of pushing dummy commit to trigger CI run.
- When the PR is merged in the CI system will re-run the tests against the master.
  On success a new Docker image will be ready for customers to deploy (This is only
  for the docker plugin, the ESX code needs to be shipped separately).

### Build, test, rinse, repeat
Use `make` or `make dockerbuild` to build.

Build environment is described in README.md. The result of the build is a set
of binaries in ./build directory.

### Local Testbed setup

In order to test locally, you'd need a test setup. Local testbed setup automation is planned but not done yet, so currently the environment has to be set up manually.

Test environment  typically consist of 1 ESX and 3 guest VMs running inside of the ESX to run all automated tests.

**TODO**: As a next step, an approach will be provided to run selective test and above testbed requirement will be simplified. Basic test run will not be blocked when minimum testbed requirement is met i.e. 1 ESX and 1 or 2 VMs.

**Note**: All 3 guest VMs can be reused to run swarm testcase by configuring swarm cluster (by making one node as master and registered others as worker nodes). It is not mandatory to create extra VM to configure swarm cluster but feel free if you would like to do so. Please make sure to set environment variables correctly as mentioned in section below.

We require ESX 6.0 and later, and a Linux VM running  Docker 1.13+ enabled for  plain text TCP connection, i.e.
Docker Daemon running with "-H tcp://0.0.0.0:2375 -H unix:///var/run/docker.sock"
options. Note that both tcp: and unix: need to be present
Please check  "Configuring and running Docker"
(https://docs.docker.com/engine/admin/configuring/)  page on how to configure this - also there is a github link at the bottom, for systemd config files.

### Photon VM specific configuration

If test environment has Docker running on Photon OS VMs, VMs should be configured to accept traffic on port 2375. Edit /etc/systemd/scripts/iptables, and add following rule at the end of the file. Restart iptable service after updating iptables file.

```
#Enable docker connections
iptables -A INPUT -p tcp --dport 2375 -j ACCEPT
```

### Testing configuration
To deploy the plugin and test code onto a test environment we support a set of
Makefile targets. These targets rely on environment variables to point to the
correct environment.

**Prerequisite**

The prerequisite to build and deploy the plugin is to have a DockerHub account. If you don't have a DockerHub account, you need to create an account [here](https://hub.docker.com/).

**Environment variables**:
- You **need** to set ESX and VM1, VM2 and VM3 environment variables

- The build will use your `username` (the output of `whoami`) to decide on the `DOCKER_HUB_REPO` name to complete our move to use managed plugin. If you want to use another DockerHub repository you need to set `DOCKER_HUB_REPO` as environment variable.

- Test verification is extended using govmomi integration and `govc` cli is **required to set** following environment variables.
  - `GOVC_USERNAME` & `GOVC_PASSWORD`: user credentials for logging in to `ESX IP`

- You **need** to set following environment variables to run swarm cluster related tests. You **need** to configure swarm cluster in order to run swarm related testcase otherwise there will be a test failure. As mentioned above, you may reuse the same set of VMs here; no need to create separately.

  - `MANAGER1` - swarm cluster manager node IP
  - `WORKER1` & `WORKER2` - swarm cluster worker node IP

**Note**: You need to manually remove older rpm/deb installation from your test VMs. With [PR 1163](https://github.com/vmware/docker-volume-vsphere/pull/1163), our build/deployment script start using managed plugin approach.

Examples:
```
# Build and deploy the code. Also deploy (but do not run) tests
DOCKER_HUB_REPO=cnastorage ESX=10.20.105.54 VM_IP=10.20.105.201 make deploy-all
```

or

```
# clean, build, deploy, test and clean again
export ESX=10.20.105.54
export VM1=10.20.105.121
export VM2=10.20.104.210
export VM3=10.20.104.241
export DOCKER_HUB_REPO=cnastorage
export GOVC_USERNAME=root
export GOVC_PASSWORD=<ESX login passwd>
export MANAGER1=10.20.105.122
export WORKER1=10.20.105.123
export WORKER2=10.20.105.124

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
ESX, VM1, VM2 and VM3) and use the following make targets:

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
description and "Testing Done:" section are accurate.
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
# Use this command to install gvt if not already installed on your system
go get -u github.com/FiloSottile/gvt
gvt fetch github.com/docker/go-plugins-helpers/volume
git add vendor
git commit -m "Added new dependency go-plugins-helpers"
exit
```

## Capturing ESX Code Coverage
Coverage is captured using make coverage target (On CI it is called using drone script).
User can configure the setup and use this target to see the coverage.
### Setup ESX box to install python coverage package
* Install https://coverage.readthedocs.io/en/coverage-4.4.1/ on your machine using pip <br />
```Desktop$ pip install coverage```
- scp the coverage package dir installed on ur machine i.e. <Python Location>/site-packages/coverage to /lib64/python3.5/site-packages/ on ESX box. <br />
eg:- ```Desktop$ scp -r /Library/Python/3.5/site-packages/coverage root@$ESX:/lib64/python3.5/site-packages/```
- scp the coverage binary i.e. /usr/local/bin/coverage to /bin/ on ESX box <br />
eg:- ```Desktop$ scp /usr/local/bin/coverage root@$ESX:/bin/```
* On ESX create sitecustomize.py inside ```/lib64/python3.5/``` <br />
The content of sitecustomize.py is
```
import coverage
coverage.process_startup()
```
* On ESX create coverage.rc file in ESX home dir as ```vi /coverage.rc``` <br />
The content of coverage.rc is
```
[run]
omit=*_test.py
parallel=True
source=/usr/lib/vmware/vmdkops
```
* On ESX modify ```/etc/security/pam_env.conf``` and add following line.
```
COVERAGE_PROCESS_START DEFAULT=/coverage.rc
```

## CI/CD System

The CI/CD system is based on [Drone platform](https://drone.io/) and the server is  https://ci.vmware.run/. More information is found at our [CI.md](https://github.com/vmware/docker-volume-vsphere/blob/master/CI.md)

## Cutting a new release guidelines

Once a release has been tagged, the CI system will start a build and push the binaries to GitHub Releases page. Creating the release on GitHub is a manual process but the binaries for it will be auto generated.

Typical steps followed.

### Tag a release
```
git tag -a -m "0.11 Release for Jan 2017" 0.11
```

Push the tag
```
git push origin 0.11
```

Check to see if the new release shows up on GitHub and the CI build has started.


### Generate the change log
```
docker run -v `pwd`:/data --rm muccg/github-changelog-generator -u vmware -p docker-volume-vsphere -t <github token> --exclude-labels wontfix,invalid,duplicate,could-not-reproduce
```

Manually eye ball the list to make sure Issues are relevant to the release (Some times labels such as wontfix have not been applied to an Issue)

Head to GitHub and author a new release add the changelog for the tag created.

**Note**: Some manual steps are required before publishing new release as shown below.
1. Download deliverables from Github release page
2. Remove VIB/DEB/RPMs from the ```Downloads``` sections
3. Perform steps from internal Confluence page to sign the VIB.

**Note**: Update [vDVS_bulletin.xml](https://github.com/vmware/docker-volume-vsphere/blob/master/docs/misc/vDVS_bulletin.xml#L19) to keep it current with the release and check changes to `[vmware/master]/docs/misc/vDVS_bulletin.xml` (see below to update the content per release)

```
<id>vDVS_driver-0.15</id> (0.15 refers to version)
<releaseDate>2017-06-30T18:47:36.688572+00:00</releaseDate> (release date where time is optional)
<vibID>vmware-esx-vmdkops-0.15.b93c186</vibID> (vmware-esx-vmdkops-<release_version>.<commit's SHA hash>)
```

4. Head to [Bintray](https://bintray.com/vmware/product/vDVS/view) to publish signed VIB
5. Push managed plugin to docker store
6. Add ```Downloads``` section with direct links; take [Release 0.13](https://github.com/vmware/docker-volume-vsphere/releases/tag/0.13) as the reference

### Publish vDVS managed plugin to Docker Store
**Note**: not automated as of 04/04/17

To push plugin image
```
DOCKER_HUB_REPO=vmware EXTRA_TAG= VERSION_TAG=latest make all
DOCKER_HUB_REPO=vmware EXTRA_TAG= VERSION_TAG=<version_tag> make all
```

### Publish signed VIB to Bintray
1. Create a new version to upload [signed VIB](https://bintray.com/vmware/vDVS/VIB)
2. Upload files at newly created version page
3. Publish the newly uploaded VIB by marking it ```public```

Update documentation following steps listed below.

## Documentation

Documentation is published to [GitHub
Pages](https://vmware.github.io/docker-volume-vsphere/) using
[jekyll](https://jekyllrb.com/).

1. Documentation is updated each time a release is tagged.
2. The latest documentation for the master can be found in
   [docs](https://github.com/vmware/docker-volume-vsphere/tree/master/docs) in
   markdown format

To update documentation
```
# 1. Checkout the gh-pages branch
git checkout gh-pages
# 2. Go to jekyll-docs directory
cd jekyll-docs
# 3. Build the jekyll site
docker run --rm --volume=$(pwd):/srv/jekyll -it jekyll/jekyll:stable jekyll build
# 4. Remove the old site and copy the new one.
rm -rvf ../documentation
mv _site ../documentation
# 6. Push to GitHub
git add documentation
git commit
git push origin gh-pages
```
