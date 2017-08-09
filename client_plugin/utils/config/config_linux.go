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
	// DefaultSharedPluginConfigPath is the default location of Log configuration file for shared plugin
	DefaultSharedPluginConfigPath = "/etc/vsphere-shared.conf"
	// DefaultSharedPluginLogPath is the default location of log (trace) file for shared plugin
	DefaultSharedPluginLogPath = "/var/log/vsphere-shared.log"

	// MountRoot is the path where VMDK and photon volumes are mounted
	MountRoot = "/mnt/vmdk"

	// VSharedMountRoot is the path where shared volumes are mounted
	VSharedMountRoot = "/mnt/vshared"
)
