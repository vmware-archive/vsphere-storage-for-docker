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
	npipeAddr    = "\\\\.\\pipe\\vsphere-dvs" // Plugin's npipe address on Windows

	// Docker plugin handshake endpoint
	// Also see https://docs.docker.com/engine/extend/plugin_api/#handshake-api
	pluginActivatePath = "/Plugin.Activate"

	// Docker volume plugin endpoints
	// Also see https://docs.docker.com/engine/extend/plugins_volume/#volume-plugin-protocol
	volumeDriverCreatePath = "/VolumeDriver.Create"
)

var (
	mountRoot              = filepath.Join(os.Getenv("LOCALAPPDATA"), "docker-volume-vsphere", "mounts") // VMDK volumes are mounted here
	listener  net.Listener = nil                                                                         // The request listener
)

// Response for /Plugin.Activate to activate Docker plugin.
type PluginActivateResponse struct {
	Implements []string
}

// HttpHandler unmarshals http requests and forwards them to the driver.
// It also marshals the driver response and writes it to the writer.
type HttpHandler struct {
	driver volume.Driver
}

// Creates a new HttpHandler backed by the provided driver.
func NewHttpHandler(driver *volume.Driver) *HttpHandler {
	httpHandler := &HttpHandler{driver: *driver}
	http.HandleFunc(pluginActivatePath, httpHandler.PluginActivate)
	http.HandleFunc(volumeDriverCreatePath, httpHandler.VolumeDriverCreate)
	return httpHandler
}

// Marshals v as a JSON string and writes it to w.
// Returns an error on marshal failure.
func (h *HttpHandler) marshalAndWrite(v interface{}, w *http.ResponseWriter) error {
	bytes, err := json.Marshal(v)
	if err == nil {
		fmt.Fprintf(*w, string(bytes))
	}
	return err
}

// PluginActivate writes handshake response to w.
func (h *HttpHandler) PluginActivate(w http.ResponseWriter, r *http.Request) {
	response := &PluginActivateResponse{Implements: []string{volumeDriver}}
	h.marshalAndWrite(response, &w)
	log.WithFields(log.Fields{"response": response}).Info("Plugin activated ")
}

// VolumeDriverCreate handles volume creation and writes the response to w.
func (h *HttpHandler) VolumeDriverCreate(w http.ResponseWriter, r *http.Request) {
	var volumeReq volume.Request
	err := json.NewDecoder(r.Body).Decode(&volumeReq)
	if err != nil {
		log.Errorf("Failed to service /VolumeDriver.Create request: %v", err)
		http.Error(w, err.Error(), 400)
		return
	}

	response := h.driver.Create(volumeReq)

	h.marshalAndWrite(response, &w)
	log.WithFields(log.Fields{"response": response}).Info("Serviced /VolumeDriver.Create ")
}

// Serves HTTP requests by listening over the provided listener.
func (h *HttpHandler) Serve(listener *net.Listener) error {
	return http.Serve(*listener, nil)
}

// Init registers an HTTP service backed by npipe to service requests using the driver.
func Init(driverName *string, driver *volume.Driver) {
	var err error
	listener, err = winio.ListenPipe(npipeAddr, nil)
	if err != nil {
		log.Warningf("Failed to initialize npipe at %s: %v", npipeAddr, err)
		fmt.Printf("Failed to initialize npipe at %s - exiting", npipeAddr)
		os.Exit(1)
	}

	httpHandler := NewHttpHandler(driver)

	log.WithFields(log.Fields{"npipe": npipeAddr}).Info("Going into Serve - Listening on npipe ")
	log.Info(httpHandler.Serve(&listener))
}

// Destroy shuts down the npipe listener.
func Destroy(driverName *string) {
	listener.Close()
}
