// Copyright 2016 VMware, Inc. All Rights Reserved.
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
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/natefinch/lumberjack"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/vmdkops"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	pluginSockDir = "/run/docker/plugins"
	vmdkPluginID  = "vmdk"
	version       = "VMDK Volume Driver v0.3"
)

// An equivalent function is not exported from the SDK.
// API supports passing a full address instead of just name.
// Using the full path during creation and deletion. The path
// is the same as the one generated interally. Ideally SDK
// should have ability to clean up sock file instead of replicating
// it here.
func fullSocketAddress(pluginName string) string {
	return filepath.Join(pluginSockDir, pluginName+".sock")
}

// init log with passed logLevel (and get config from configFIle if it's present)
// returns True if using defaults,  False if using config file
func logInit(logLevel *string, logFile *string, configFile *string) bool {
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse log level: %v", err))
	}

	usingConfigDefaults := false
	c, err := config.Load(*configFile)
	if err != nil {
		if os.IsNotExist(err) {
			usingConfigDefaults = true // no .conf file, so using defaults
			c = config.Config{}
			config.SetDefaults(&c)
		} else {
			panic(fmt.Sprintf("Failed to load config file %s: %v",
				*configFile, err))
		}
	}

	path := c.LogPath
	if logFile != nil {
		path = *logFile
	}
	log.SetOutput(&lumberjack.Logger{
		Filename: path,
		MaxSize:  c.MaxLogSizeMb,  // megabytes
		MaxAge:   c.MaxLogAgeDays, // days
	})

	log.SetFormatter(new(VmwareFormatter))
	log.SetLevel(level)

	if usingConfigDefaults {
		log.Info("No config file found. Using defaults.")
	}
	return usingConfigDefaults
}

// main for docker-volume-vsphere
// Parses flags, inits mount refcounters and finally services Docker requests
func main() {
	// connect to this socket
	port := flag.Int("port", 1019, "Default port for vmci")
	useMockEsx := flag.Bool("mock_esx", false, "Mock the ESX server")
	logLevel := flag.String("log_level", "info", "Logging Level")
	configFile := flag.String("config", config.DefaultConfigPath, "Configuration file path")
	flag.Parse()

	vmdkops.EsxPort = *port

	logInit(logLevel, nil, configFile)

	log.WithFields(log.Fields{
		"version":   version,
		"port":      vmdkops.EsxPort,
		"mock_esx":  *useMockEsx,
		"log_level": *logLevel,
		"config":    *configFile,
	}).Info("Docker VMDK plugin started ")

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChannel
		log.WithFields(log.Fields{"signal": sig}).Warning("Received signal ")
		os.Remove(fullSocketAddress(vmdkPluginID))
		os.Exit(0)
	}()

	driver := newVmdkDriver(*useMockEsx)
	handler := volume.NewHandler(driver)

	log.WithFields(log.Fields{
		"address": fullSocketAddress(vmdkPluginID),
	}).Info("Going into ServeUnix - Listening on Unix socket ")

	log.Info(handler.ServeUnix("root", fullSocketAddress(vmdkPluginID)))
}
