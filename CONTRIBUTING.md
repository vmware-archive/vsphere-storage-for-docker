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
