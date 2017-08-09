// Copyright 2016-2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Platform independent fs functionality.

package fs

import (
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
)

// VolumeDevSpec - volume spec returned from the server on an attach
type VolumeDevSpec struct {
	Unit                    string
	ControllerPciSlotNumber string
}

// Mkdir creates a directory at the specified path.
func Mkdir(path string) error {
	stat, err := os.Lstat(path)
	if os.IsNotExist(err) {
		log.WithField("path", path).Info("Directory doesn't exist, creating it ")
		if err := os.MkdirAll(path, 0755); err != nil {
			log.WithFields(log.Fields{"path": path,
				"err": err}).Error("Failed to create directory ")
			return err
		}
	} else if err != nil {
		log.WithFields(log.Fields{"path": path,
			"err": err}).Error("Failed to test directory existence ")
		return err
	}

	if stat != nil && !stat.IsDir() {
		msg := fmt.Sprintf("%v already exists and it's not a directory", path)
		log.Error(msg)
		return fmt.Errorf(msg)
	}
	return nil
}

// Rmdir removes a directory identified by its path
func Rmdir(path string) error {
	return os.Remove(path)
}

// GetMountRootEntries returns the list of volumes under mountRoot
func GetMountRootEntries(mountRoot string) ([]string, error) {
	var vols []string
	// Read entries in mountRoot for all volumes that are or were in use via the plugin
	volumes, err := ioutil.ReadDir(mountRoot)
	if err != nil {
		log.Errorf("Unable to read entries from %s (%v)", mountRoot, err)
		return vols, err
	}

	for _, vol := range volumes {
		vols = append(vols, vol.Name())
	}
	return vols, nil
}
