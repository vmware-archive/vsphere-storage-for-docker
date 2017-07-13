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

package main

// A vSphere Shared Docker Data Volume plugin - main

import (
	"os"
	"reflect"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/plugin_server"
)

// main for docker-volume-vsphere
// Parses flags, initializes and mounts refcounters and finally initializes the server.
func main() {
	var driver volume.Driver

	cfg, err := config.InitConfig(config.DefaultSharedPluginConfigPath, config.DefaultSharedPluginLogPath,
		config.SharedDriver, "")
	if err != nil {
		log.Warning("Failed to initialize config variables for shared plugin")
		os.Exit(1)
	}

	if cfg.Driver == config.SharedDriver {
		driver = shared.NewVolumeDriver(cfg, config.MountRoot)
	} else {
		log.Warning("Unknown driver or invalid/missing driver options, exiting - ", cfg.Driver)
		os.Exit(1)
	}

	if reflect.ValueOf(driver).IsNil() == true {
		log.Warning("Error in driver initialization exiting - ", cfg.Driver)
		os.Exit(1)
	}

	plugin_server.StartServer(cfg.Driver, &driver)
}
