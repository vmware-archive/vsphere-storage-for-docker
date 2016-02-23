# Contributing Code

* Create a fork or branch (if you can) and make your changes.
* Push your changes and create a pull request.

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

- Each checkin into a branch on the official repo will run builtin unit tests.
  - Unit test failure details are posted on the CI server.

- Each Pull Request will run the full set of tests part of the CI system.
  - On success or failure, the compiled binary and all relevant logs will
    be posted as a docker image on Docker Hub with a tag <branch>-<build>
    (Ex: kerneltime/docker-vmdk-plugin:docker-image-plugin-76)
  - Developer can pull the image using docker pull and debug.

- Each commit into the master operation will run the full set of tests
  part of the CI system.
  - On success a docker image consisting of only the shippable docker
    plugin is posted to docker hub. The image is tagged with latest
    (and <branch>-<build>). Any one can pull this image and run it to
    use the docker plugin.
  - On failure all binaries and log files will be posted with <branch>-<build>
    tag to Docker hub. Developer can pull this image and debug.

A typical workflow for a developer should be.

- Create a branch, push changes and make sure unit tests do not break as reported
  by the CI system.
- Run end to end tests on their testbed.
- When ready post a RP. This will trigger a full set of tests on ESX. After all 
  the tests pass and the review is complete the PR will be merged in.
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

TODO:
1. Push Docker images on failures, needs custom scripting.
2. Deploy VMs on nested ESX.
3. Write tests for end to end testing.
  1. Needs a guarantee that only one build running tests at a time.
4. Write a stub server to allow for unit testing the docker plugin code.

