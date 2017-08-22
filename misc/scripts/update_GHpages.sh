#!/bin/bash
# Following script copies the markdown files from vmware:master to gh-pages

# get latest changes from master
git pull

#list files with .md file type
list_files=$( git ls-files "*.md" )

#Create Temp directory to store .md files
mkdir ~/backup
echo $list_files
while IFS=' ' read -ra arr; do
      for i in "${arr[@]}"; do
          # process "$i"
          echo $i
          echo "Copying file"
          cp $i ~/backup/
      done
 done <<< "$list_files"

#Creating branch gh-pages
git checkout -b gh-pages origin/gh-pages

#copying md files from backup to jekyll-docs directory
cp -r ~/backup/* jekyll-docs/

# adding the files to commit
git add ./\*.md
git commit -m "Copying changes to gh-pages branch from vmware:master"
git push vmware HEAD:gh-pages

#Delete backup directory which had .md files
rm -rf ~/backup

#Deleting gh-pages branch
git checkout master
git branch -D gh-pages
