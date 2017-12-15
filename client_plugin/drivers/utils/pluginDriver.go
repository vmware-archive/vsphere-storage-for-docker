// Copyright 2017 VMware, Inc. All Rights Reserved.
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

package utils

//
// Common Utility for driver interface.
//

import (
	"os"
	"path/filepath"

	"github.com/vmware/vsphere-storage-for-docker/client_plugin/utils/refcount"
)

// PluginDriver - helper struct to hold common utilities for driver interface
type PluginDriver struct {
	RefCounts     *refcount.RefCountsMap
	MountIDtoName map[string]string // map of mountID -> full volume name
	MountRoot     string
}

// GetMountPoint returns the mount point based on MountRoot and volume name
func (u *PluginDriver) GetMountPoint(volName string) string {
	return filepath.Join(u.MountRoot, volName) + string(os.PathSeparator)
}

// In following three operations on refcount, if refcount
// map hasn't been initialized, return 1 to prevent detach and remove.

// Return the number of references for the given volume
func (u *PluginDriver) GetRefCount(vol string) uint {
	if u.RefCounts.IsInitialized() != true {
		return 1
	}
	return u.RefCounts.GetCount(vol)
}

// Increment the reference count for the given volume
func (u *PluginDriver) IncrRefCount(vol string) uint {
	if u.RefCounts.IsInitialized() != true {
		return 1
	}
	return u.RefCounts.Incr(vol)
}

// Decrement the reference count for the given volume
func (u *PluginDriver) DecrRefCount(vol string) (uint, error) {
	if u.RefCounts.IsInitialized() != true {
		return 1, nil
	}
	return u.RefCounts.Decr(vol)
}
