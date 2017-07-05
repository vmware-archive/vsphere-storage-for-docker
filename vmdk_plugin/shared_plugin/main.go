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

// A Shared Docker Data Volume plugin - main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/drivers/shared"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/log_formatter"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/plugin_server"
)

// main for docker-volume-vsphere
// Parses flags, initializes and mounts refcounters and finally initializes the server.
func main() {
	var driver volume.Driver

	// Get options from ENV (where available), and from command line.
	// ENV takes precedence, so we can modify it in Docker plugin install
	logEnv := os.Getenv("VDVS_LOG_LEVEL")
	logLevel := &logEnv
	if *logLevel == "" {
		logLevel = flag.String("log_level", "", "Logging Level")
	}
	configFile := flag.String("config", config.DefaultSharedPluginConfigPath, "Configuration file path")
	driverName := flag.String("driver", "", "Volume driver")
	kvStorePort := flag.String("KVport", "", "Port number for KV store service")
	strTimeout := flag.String("KVtimeout", "", "Timeout for requests to KV store")

	flag.Parse()

	log_formatter.LogInit(logLevel, nil, configFile)

	// Load the configuration if one was provided.
	c, err := config.Load(*configFile)
	if err != nil {
		log.Warningf("Failed to load config file %s: %v", *configFile, err)
	}

	// If no driver provided on the command line, use the one in the
	// config file or the default.
	if *driverName == "" {
		if err == nil && c.Driver != "" {
			*driverName = c.Driver
		} else {
			*driverName = sharedDriver
		}
	}

	// The port number for KV store should be read from the configuration file
	if *kvStorePort == "" {
		if err == nil && c.KVport != "" {
			*kvStorePort = c.KVport
		} else {
			*kvStorePort = defaultKVstorePort
		}
	}
	log.Infof("Using KV store port: %s", *kvStorePort)

	// Set timeout for KV store operations
	kvtimeout := 0
	if *strTimeout == "" {
		// No CLI input. Set from config file.
		// On failure set default.
		if err == nil && c.KVtimeout != "" {
			kvtimeout, err = strconv.Atoi(c.KVtimeout)
			log.Infof("Setting KV timeout from config file: %v", err)
		}
		if err != nil {
			kvtimeout = defaultKVtimeout
			log.Infof("Setting KV timeout to default value")
		}
	} else {
		// Set from CLI input
		// On failure set default.
		kvtimeout, err = strconv.Atoi(*strTimeout)
		log.Infof("Setting KV timeout from CLI input")
		if err != nil {
			kvtimeout = defaultKVtimeout
			log.Infof("Failed to set KV timeout from CLI input. Falling back to default value. Reason: %v", err)
		}
	}

	// The windows plugin only supports the vsphere driver.
	if runtime.GOOS == "windows" && *driverName != sharedDriver {
		msg := fmt.Sprintf("Shared Plugin only supports the %s driver on Windows, ignoring parameter driver = %s.",
			sharedDriver, c.Driver)
		log.Warning(msg)
		fmt.Println(msg)
		*driverName = sharedDriver
	}

	log.WithFields(log.Fields{
		"driver":    *driverName,
		"log_level": *logLevel,
		"config":    *configFile,
	}).Info("Starting plugin ")

	// The Shared driver doesn't depend on the config file for options
	if *driverName == sharedDriver {
		driver = shared.NewVolumeDriver(plugin_server.MountRoot, *driverName, *kvStorePort, kvtimeout)
	} else {
		log.Warning("Unknown driver or invalid/missing driver options, exiting - ", *driverName)
		os.Exit(1)
	}

	if reflect.ValueOf(driver).IsNil() == true {
		log.Warning("Error in driver initialization exiting - ", *driverName)
		os.Exit(1)
	}

	server := plugin_server.NewPluginServer(*driverName, &driver)

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChannel
		log.WithFields(log.Fields{"signal": sig}).Warning("Received signal ")
		server.Destroy()
		os.Exit(0)
	}()

	server.Init()
}
