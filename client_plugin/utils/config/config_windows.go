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

package config

import (
	"os"
	"path/filepath"
)

var (
	// DefaultVMDKPluginConfigPath is the default location of the vmdk plugin config file.
	DefaultVMDKPluginConfigPath = filepath.Join(os.Getenv("PROGRAMDATA"), "vsphere-storage-for-docker", "vsphere-storage-for-docker.conf")
	// DefaultVMDKPluginLogPath is the default location of the vmdk plugin log (trace) file.
	DefaultVMDKPluginLogPath = filepath.Join(os.Getenv("LOCALAPPDATA"), "vsphere-storage-for-docker", "logs", "vsphere-storage-for-docker.log")

	// VMDK volumes are mounted here
	MountRoot = filepath.Join(os.Getenv("LOCALAPPDATA"), "vsphere-storage-for-docker", "mounts")
)
