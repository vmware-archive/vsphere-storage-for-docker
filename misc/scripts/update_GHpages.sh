#!/bin/bash

# Following script copies the markdown files from vmware:master to gh-pages
# Local copy of vDVS project is needed before running this script (git clone https://github.com/vmware/docker-volume-vsphere.git)
# Script has to be run from the docker-volume-vsphere/
# e.g. ./misc/scripts/updateGHpages.sh

BACKUP_DIR=/tmp/backup
TARGET_REPO=origin
SOURCE_REPO=vmware
DOCUMENT_BRANCH=gh-pages
LOCAL_MASTER=master

# get latest changes from vmware:master
git pull $SOURCE_REPO $LOCAL_MASTER

#Create Temp directory to store .md files
mkdir $BACKUP_DIR

#listing md files only from docs directory
cd docs/
#list files with .md file type
list_files=$( git ls-files "*.md" )

#copying md files to backup dir
while IFS=' ' read -ra arr; do
      for i in "${arr[@]}"; do
          echo "Copying file $i"
          cp $i $BACKUP_DIR
      done
 done <<< "$list_files"

#Creating branch gh-pages
git checkout -b $DOCUMENT_BRANCH $SOURCE_REPO/$DOCUMENT_BRANCH

cd ..

#copying md files from backup to jekyll-docs directory
cp -a $BACKUP_DIR/. jekyll-docs/

rm -fr docs

# adding the md files to commit and pushing the changes to gh-pages branch
git add ./\*.md
git commit -m "Copying changes to gh-pages branch from vmware:master"
git push $SOURCE_REPO HEAD:$DOCUMENT_BRANCH

#Delete backup directory which had .md files
rm -rf $BACKUP_DIR

echo "Performing steps to generate customer facing document"
cd jekyll-docs/
docker run --rm --volume=$(pwd):/srv/jekyll -it jekyll/jekyll:stable jekyll build
rm -rvf ../documentation
mv _site ../documentation

#updating documentation/index.html
sed -i -e 's/\/gh-pages\/jekyll-docs\/\/.md/\/gh-pages\/jekyll-docs\/\/index.md/g' ../documentation/index.html

cd ..
git add documentation
git commit -m "Generating the customer facing document"
git push $SOURCE_REPO HEAD:$DOCUMENT_BRANCH

#Deleting gh-pages branch
git checkout $LOCAL_MASTER
git branch -D $DOCUMENT_BRANCH
