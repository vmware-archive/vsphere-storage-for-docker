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

// A VMDK Docker Data Volume plugin implementation for Windows OS.

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
	npipeAddr = `\\.\pipe\global` // Plugin's npipe address
)

var (
	mountRoot = filepath.Join(os.Getenv("LOCALAPPDATA"), "docker-volume-global", "mounts") // VMDK volumes are mounted here
)

// NpipePluginServer serves HTTP requests from Docker over windows npipe.
type NpipePluginServer struct {
	PluginServer
	driver   *volume.Driver // The driver implementation
	mux      *http.ServeMux // The HTTP mux
	listener net.Listener   // The npipe listener
}

// Response for /Plugin.Activate to activate the Docker plugin.
type PluginActivateResponse struct {
	Implements []string // Slice of implemented driver types
}

// NewPluginServer returns a new instance of NpipePluginServer.
func NewPluginServer(driverName string, driver *volume.Driver) *NpipePluginServer {
	return &NpipePluginServer{driver: driver, mux: http.NewServeMux()}
}

// writeJSON writes the JSON encoding of resp to the writer.
func writeJSON(resp interface{}, writer *http.ResponseWriter) error {
	bytes, err := json.Marshal(resp)
	if err == nil {
		fmt.Fprintf(*writer, string(bytes))
	}
	return err
}

// writeError writes an error message and status to the writer.
func writeError(path string, writer http.ResponseWriter, req *http.Request, status int, err error) {
	log.WithFields(log.Fields{"path": path, "req": req, "status": status,
		"err": err}).Error("Failed to service request ")
	http.Error(writer, err.Error(), status)
}

// PluginActivate writes the plugin's handshake response to the writer.
func (s *NpipePluginServer) PluginActivate(writer http.ResponseWriter, req *http.Request) {
	resp := &PluginActivateResponse{Implements: []string{volumeDriver}}
	errJSON := writeJSON(resp, &writer)
	if errJSON != nil {
		writeError(pluginActivatePath, writer, req, http.StatusInternalServerError, errJSON)
		return
	}
	log.WithFields(log.Fields{"resp": resp}).Info("Plugin activated ")
}

// VolumeDriverCreate creates a volume and writes the response to the writer.
func (s *NpipePluginServer) VolumeDriverCreate(writer http.ResponseWriter, req *http.Request) {
	var volumeReq volume.Request
	err := json.NewDecoder(req.Body).Decode(&volumeReq)
	if err != nil {
		writeError(volumeDriverCreatePath, writer, req, http.StatusBadRequest, err)
		return
	}

	resp := (*s.driver).Create(volumeReq)
	errJSON := writeJSON(resp, &writer)
	if errJSON != nil {
		writeError(volumeDriverCreatePath, writer, req, http.StatusInternalServerError, errJSON)
		return
	}
	log.WithFields(log.Fields{"resp": resp}).Info("Serviced /VolumeDriver.Create ")
}

// registerHttpHandlers registers handlers with the HTTP mux.
func (s *NpipePluginServer) registerHandlers() {
	s.mux.HandleFunc(pluginActivatePath, s.PluginActivate)
	s.mux.HandleFunc(volumeDriverCreatePath, s.VolumeDriverCreate)
}

// Init initializes the npipe listener which serves HTTP requests
// from Docker using the HTTP mux.
func (s *NpipePluginServer) Init() {
	var err error
	s.listener, err = winio.ListenPipe(npipeAddr, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to initialize npipe at %s - exiting", npipeAddr)
		log.WithFields(log.Fields{"err": err}).Fatal(msg)
		fmt.Println(msg)
		os.Exit(1)
	}

	s.registerHandlers()
	log.WithFields(log.Fields{"npipe": npipeAddr}).Info("Going into Serve - Listening on npipe ")
	log.Info(http.Serve(s.listener, s.mux))
}

// Destroy shuts down the npipe listener.
func (s *NpipePluginServer) Destroy() {
	log.WithFields(log.Fields{"npipe": npipeAddr}).Info("Closing npipe listener ")
	s.listener.Close()
}
