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

// A VMDK Docker Data Volume plugin implementation for linux that
// relies on the docker/go-plugins-helpers/volume API.

import (
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
)

const (
	mountRoot     = "/mnt/global"         // VMDK and photon volumes are mounted here
	pluginSockDir = "/run/docker/plugins" // Unix sock for the plugin is maintained here
)

// SockPluginServer serves HTTP requests from Docker over unix sock.
type SockPluginServer struct {
	PluginServer
	sockAddr string         // Server's unix sock address
	driver   *volume.Driver // The driver implementation
}

// An equivalent function is not exported from the SDK.
// API supports passing a full address instead of just name.
// Using the full path during creation and deletion. The path
// is the same as the one generated internally. Ideally SDK
// should have ability to clean up sock file instead of replicating
// it here.
func fullSocketAddress(pluginName string) string {
	return filepath.Join(pluginSockDir, pluginName+".sock")
}

// NewPluginServer creates a new instance of SockPluginServer.
func NewPluginServer(driverName string, driver *volume.Driver) *SockPluginServer {
	return &SockPluginServer{sockAddr: fullSocketAddress(driverName), driver: driver}
}

// Init registers the volume driver with a handler to service HTTP
// requests from Docker.
func (s *SockPluginServer) Init() {
	handler := volume.NewHandler(*s.driver)

	log.WithFields(log.Fields{
		"address": s.sockAddr,
	}).Info("Going into ServeUnix - Listening on Unix socket ")

	log.Info(handler.ServeUnix("root", s.sockAddr))
}

// Destroy removes the Docker plugin sock.
func (s *SockPluginServer) Destroy() {
	os.Remove(s.sockAddr)
}
