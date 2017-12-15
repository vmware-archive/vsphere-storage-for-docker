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

package plugin_server

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/vsphere-storage-for-docker/client_plugin/utils/codecov"
)

// PluginServer responds to HTTP requests from Docker.
type PluginServer interface {
	// Init initializes the server.
	Init()
	// Destroy destroys the server.
	Destroy()
}

// StartServer starts a plugin server based on runtime OS
func StartServer(driverName string, driver *volume.Driver) {
	server := NewPluginServer(driverName, driver)

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChannel
		log.WithFields(log.Fields{"signal": sig}).Warning("Received signal ")
		coverage.Capture()
		server.Destroy()
		os.Exit(0)
	}()

	server.Init()
}
