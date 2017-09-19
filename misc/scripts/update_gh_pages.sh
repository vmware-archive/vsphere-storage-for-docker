#!/bin/bash
set -e

# Following script copies the markdown files from vmware:master to the gh-pages branch.
# Please check for pre-requisite before running the script and proceed further if your
# local workspace meets the pre-requisite requirement.
#
# A. Pre-requisite:
#
# 1. Local copy of vDVS project is needed before running this script
#   $ git clone https://github.com/vmware/docker-volume-vsphere.git)
#
# 2. Before executing this script, its important to have two git remotes: origin and vmware
#   $ git remote -v
#    origin	https://github.com/ashahi1/docker-volume-vsphere.git (fetch)
#    origin	https://github.com/ashahi1/docker-volume-vsphere.git (push)
#    vmware	https://github.com/vmware/docker-volume-vsphere.git (fetch)
#    vmware	https://github.com/vmware/docker-volume-vsphere.git (push)
#
# 3. Now we need to fetch all the remote branches
#   $ git fetch --all
#
#
# B. Run the script from docker-volume-vsphere/
#   $ ./misc/scripts/update_gh_pages.sh
#

BACKUP_DIR=/tmp/backup
TARGET_REPO=origin
SOURCE_REPO=vmware
DOCUMENT_BRANCH=gh-pages
LOCAL_MASTER=master

# Get latest changes from vmware:master
git pull $SOURCE_REPO $LOCAL_MASTER

# Create Temp directory to store .md files
mkdir $BACKUP_DIR

# Copying md files from external folder to backup directory
cp -a docs/external/. $BACKUP_DIR

# Creating branch gh-pages
git checkout -b $DOCUMENT_BRANCH $SOURCE_REPO/$DOCUMENT_BRANCH

# First removing all the md files from jekyll-docs so that md files removed from vmware/master
# are also removed from gh-pages
rm -fr jekyll-docs/*.md

# Copying md files from backup to jekyll-docs directory
cp -a $BACKUP_DIR/. jekyll-docs/

rm -fr docs

# Adding the md files to commit and pushing the changes to gh-pages branch
git add ./\*.md

# Delete backup directory which had .md files
rm -rf $BACKUP_DIR

echo "Performing steps to generate customer facing document"
cd jekyll-docs/

# Generating the html pages
docker run --rm --volume=$(pwd):/srv/jekyll -it jekyll/jekyll:stable jekyll build
rm -rvf ../documentation
mv _site ../documentation

# Updating documentation/index.html
sed -i -e 's/\/\/.md/\/\/index.md/g' ../documentation/index.html

cd ..
git add documentation

# Generating the customer facing documents - vmware.github.io
git commit -m "Generating the customer facing document"
git push $SOURCE_REPO HEAD:$DOCUMENT_BRANCH

# Deleting the gh-pages branch
git checkout $LOCAL_MASTER
git branch -D $DOCUMENT_BRANCH
