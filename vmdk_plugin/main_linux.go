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

package main

// A VMDK Docker Data Volume plugin - main
// relies on docker/go-plugins-helpers/volume API

import (
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
)

const (
	mountRoot     = "/mnt/vmdk" // VMDK and photon volumes are mounted here
	pluginSockDir = "/run/docker/plugins"
)

// An equivalent function is not exported from the SDK.
// API supports passing a full address instead of just name.
// Using the full path during creation and deletion. The path
// is the same as the one generated internally. Ideally SDK
// should have ability to clean up sock file instead of replicating
// it here.
func fullSocketAddress(pluginName string) string {
	return filepath.Join(pluginSockDir, pluginName+".sock")
}

// Init initializes a handler to service Docker requests using the driver.
func Init(driverName *string, driver *volume.Driver) {
	handler := volume.NewHandler(*driver)

	log.WithFields(log.Fields{
		"address": fullSocketAddress(*driverName),
	}).Info("Going into ServeUnix - Listening on Unix socket ")

	log.Info(handler.ServeUnix("root", fullSocketAddress(*driverName)))
}

// Destroy removes the Docker plugin socket.
func Destroy(driverName *string) {
	os.Remove(fullSocketAddress(*driverName))
}
