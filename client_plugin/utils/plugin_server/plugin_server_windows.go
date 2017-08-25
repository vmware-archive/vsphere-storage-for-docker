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

// A VMDK Docker Data Volume plugin implementation for Windows OS.

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	ps "github.com/vmware/docker-volume-vsphere/client_plugin/utils/powershell"
)

const (
	// npipeAddr is the plugin's npipe address.
	npipeAddr = "//./pipe/vsphere-dvs"

	// pluginsPath is the Docker's plugin directory path.
	pluginsPath = `C:\ProgramData\docker\plugins\`

	// configFilePath is the plugin's config file path.
	configFilePath = pluginsPath + "vsphere.json"
)

// pluginConfig is the plugin config.
var pluginConfig = fmt.Sprintf(`{ "Name": "vsphere", "Addr": "npipe://%s" }`, npipeAddr)

// writePluginConfig writes the plugin config to Docker's plugin directory.
func writePluginConfig() error {
	if err := os.MkdirAll(pluginsPath, 0755); err != nil {
		log.WithFields(log.Fields{"path": pluginsPath,
			"err": err}).Error("Failed to create plugin config directory ")
		return err
	}
	if err := ioutil.WriteFile(configFilePath, []byte(pluginConfig), 0644); err != nil {
		log.WithFields(log.Fields{"path": configFilePath,
			"err": err}).Error("Failed to write plugin config ")
		return err
	}
	return nil
}

// NpipePluginServer serves HTTP requests from Docker over windows npipe.
type NpipePluginServer struct {
	PluginServer
	driver   *volume.Driver // The driver implementation
	listener net.Listener   // The npipe listener
}

// NewPluginServer returns a new instance of NpipePluginServer.
func NewPluginServer(driverName string, driver *volume.Driver) *NpipePluginServer {
	return &NpipePluginServer{driver: driver}
}

// Init initializes the npipe listener which serves HTTP requests
// from Docker using the HTTP mux.
func (s *NpipePluginServer) Init() {
	var err error
	var msg string

	if err = writePluginConfig(); err != nil {
		msg = fmt.Sprintf("Failed to write Docker plugin config - exiting")
	} else {
		s.listener, err = winio.ListenPipe(npipeAddr, nil)
		if err != nil {
			msg = fmt.Sprintf("Failed to initialize npipe at %s - exiting", npipeAddr)
		}
	}

	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal(msg)
		fmt.Println(msg)
		os.Exit(1)
	}

	handler := volume.NewHandler(*s.driver)
	log.WithFields(log.Fields{"npipe": npipeAddr}).Info("Going into Serve - Listening on npipe ")
	log.Info(handler.Serve(s.listener))
}

// Destroy shuts down the npipe listener.
func (s *NpipePluginServer) Destroy() {
	log.WithFields(log.Fields{"npipe": npipeAddr}).Info("Closing npipe listener ")
	s.listener.Close()
	ps.Exit()
}
