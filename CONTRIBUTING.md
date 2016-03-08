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
5. On failure for now the publishing will be manual. Going forward 0.6 release of drone will support publish on failure. One option might be to run a forked drone release in co-ordination with the CNA folks.
