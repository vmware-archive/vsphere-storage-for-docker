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

// A VMDK Docker Data Volume plugin implementation for windows.

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Microsoft/go-winio"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
)

const (
	volumeDriver = "VolumeDriver"             // Docker driver type
	npipeAddr    = "\\\\.\\pipe\\vsphere-dvs" // Plugin's npipe address on windows

	// Docker plugin handshake endpoint
	// Also see https://docs.docker.com/engine/extend/plugin_api/#handshake-api
	pluginActivatePath = "/Plugin.Activate"

	// Docker volume plugin endpoints
	// Also see https://docs.docker.com/engine/extend/plugins_volume/#volume-plugin-protocol
	volumeDriverCreatePath = "/VolumeDriver.Create"
)

var (
	mountRoot = filepath.Join(os.Getenv("LOCALAPPDATA"), "docker-volume-vsphere", "mounts") // VMDK volumes are mounted here
)

// NpipeListener serves HTTP requests over windows npipe.
type NpipeServer struct {
	Server
	driver   *volume.Driver
	mux      *http.ServeMux
	listener net.Listener
}

// Response for /Plugin.Activate to activate the Docker plugin.
type PluginActivateResponse struct {
	Implements []string
}

// NewServer creates a new instance of NpipeServer.
func NewServer(driverName string, driver *volume.Driver) *NpipeServer {
	return &NpipeServer{driver: driver, mux: http.NewServeMux()}
}

// Marshals v as a JSON string and writes it to w.
// Returns an error if marshalling fails.
func marshalAndWrite(v interface{}, w *http.ResponseWriter) error {
	bytes, err := json.Marshal(v)
	if err == nil {
		fmt.Fprintf(*w, string(bytes))
	}
	return err
}

// PluginActivate writes handshake response to w.
func (s *NpipeServer) PluginActivate(w http.ResponseWriter, r *http.Request) {
	response := &PluginActivateResponse{Implements: []string{volumeDriver}}
	marshalAndWrite(response, &w)
	log.WithFields(log.Fields{"response": response}).Info("Plugin activated ")
}

// VolumeDriverCreate creates a volume and writes the response to w.
func (s *NpipeServer) VolumeDriverCreate(w http.ResponseWriter, r *http.Request) {
	var volumeReq volume.Request
	err := json.NewDecoder(r.Body).Decode(&volumeReq)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Failed to service /VolumeDriver.Create request")
		http.Error(w, err.Error(), 400)
		return
	}

	response := (*s.driver).Create(volumeReq)
	marshalAndWrite(response, &w)
	log.WithFields(log.Fields{"response": response}).Info("Serviced /VolumeDriver.Create ")
}

// registerHttpHandlers registers HTTP handlers.
func (s *NpipeServer) registerHandlers() {
	s.mux.HandleFunc(pluginActivatePath, s.PluginActivate)
	s.mux.HandleFunc(volumeDriverCreatePath, s.VolumeDriverCreate)
}

// Init registers an HTTP service backed by npipe to service requests using the driver.
func (s *NpipeServer) Init() {
	var err error
	s.listener, err = winio.ListenPipe(npipeAddr, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to initialize npipe at %s - exiting", npipeAddr)
		log.WithFields(log.Fields{"error": err}).Error(msg)
		fmt.Println(msg)
		os.Exit(1)
	}

	s.registerHandlers()
	log.WithFields(log.Fields{"npipe": npipeAddr}).Info("Going into Serve - Listening on npipe ")
	log.Info(http.Serve(s.listener, s.mux))
}

// Destroy shuts down the npipe listener.
func (s *NpipeServer) Destroy() {
	s.listener.Close()
}
