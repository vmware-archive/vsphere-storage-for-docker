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

package plugin_utils

// This file holds utility/helper methods required in plugin module

import (
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/vmware/vsphere-storage-for-docker/client_plugin/drivers"
	"github.com/vmware/vsphere-storage-for-docker/client_plugin/utils/fs"
)

const (
	// index datastore from volume meta
	// "datastore" key is defined in vmdkops service
	datastoreKey = "datastore"

	// PluginInitError message to indicate that plugin initialization(refcounting) is not yet complete
	PluginInitError = "Plugin initialization in progress."
)

// VolumeInfo - Volume fullname, datastore and metadata
type VolumeInfo struct {
	VolumeName    string
	DatastoreName string
	VolumeMeta    map[string]interface{}
}

// AlreadyMounted - check if volume is already mounted on the mountRoot
func AlreadyMounted(name string, mountRoot string) bool {
	volumeMap, err := fs.GetMountInfo(mountRoot)

	if err != nil {
		return false
	}

	if _, ok := volumeMap[name]; ok {
		return true
	}
	return false
}

// makeFullVolName - return a full name in format volume@datastore
func makeFullVolName(volName string, datastoreName string) string {
	return strings.Join([]string{volName, datastoreName}, "@")
}

// IsFullVolName - Check if volume name is full volume name
func IsFullVolName(volName string) bool {
	return strings.ContainsAny(volName, "@")
}

// GetVolumeInfo - return VolumeInfo with a qualified volume name.
// Optionally returns datastore and volume metadata if retrieved from ESX.
// If Volume Metadata is nil then caller can use getVolume()
func GetVolumeInfo(name string, datastoreName string, d drivers.VolumeDriver) (*VolumeInfo, error) {
	// If fullname already, return for Linux, convert to short name for Windows
	// so, the correct datastore name is used here.
	if IsFullVolName(name) {
		if runtime.GOOS == "windows" {
			name = strings.Split(name, "@")[0]
		} else {
			return &VolumeInfo{name, "", nil}, nil
		}
	}

	// if datastore name is provided, append and return
	if datastoreName != "" {
		return &VolumeInfo{makeFullVolName(name, datastoreName), datastoreName, nil}, nil
	}

	// Do a get trip to esx and construct full name
	volumeMeta, err := d.GetVolume(name)
	if err != nil {
		log.Errorf("Unable to get volume metadata %s (err: %v)", name, err)
		return nil, err
	}
	datastoreName = volumeMeta[datastoreKey].(string)

	return &VolumeInfo{makeFullVolName(name, datastoreName), datastoreName, volumeMeta}, nil
}
