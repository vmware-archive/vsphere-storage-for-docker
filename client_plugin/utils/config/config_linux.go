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

package config

const (
	// Default paths - used in log init in main() and test:

	// DefaultVMDKPluginConfigPath is the default location of Log configuration file
	DefaultVMDKPluginConfigPath = "/etc/docker-volume-vsphere.conf"
	// DefaultVMDKPluginLogPath is the default location of log (trace) file
	DefaultVMDKPluginLogPath = "/var/log/docker-volume-vsphere.log"
	// DefaultVFilePluginConfigPath is the default location of Log configuration file for vFile plugin
	DefaultVFilePluginConfigPath = "/etc/vfile.conf"
	// DefaultVFilePluginLogPath is the default location of log (trace) file for vFile plugin
	DefaultVFilePluginLogPath = "/var/log/vfile.log"

	// MountRoot is the path where VMDK and photon volumes are mounted
	MountRoot = "/mnt/vmdk"

	// VFileMountRoot is the path where vFile volumes are mounted
	VFileMountRoot = "/mnt/vfile"
)
