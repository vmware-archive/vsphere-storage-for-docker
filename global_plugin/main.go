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

// A Global Docker Data Volume plugin - main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/natefinch/lumberjack"
	"github.com/vmware/docker-volume-vsphere/global_plugin/drivers/global"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/log_formatter"
)

// PluginServer responds to HTTP requests from Docker.
type PluginServer interface {
	// Init initializes the server.
	Init()
	// Destroy destroys the server.
	Destroy()
}

// init log with passed logLevel (and get config from configFile if it's present)
// returns True if using defaults,  False if using config file
func logInit(logLevel *string, logFile *string, configFile *string) bool {
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
	if path == "" {
		path = config.DefaultGlobalLogPath
	}

	if logFile != nil {
		path = *logFile
	}
	log.SetOutput(&lumberjack.Logger{
		Filename: path,
		MaxSize:  c.MaxLogSizeMb,  // megabytes
		MaxAge:   c.MaxLogAgeDays, // days
	})

	if *logLevel == "" {
		*logLevel = c.LogLevel
	}

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse log level: %v", err))
	}

	log.SetFormatter(new(log_formatter.VmwareFormatter))
	log.SetLevel(level)

	if usingConfigDefaults {
		log.Info("No config file found. Using defaults.")
	}
	return usingConfigDefaults
}

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
	configFile := flag.String("config", config.DefaultGlobalConfigPath, "Configuration file path")
	driverName := flag.String("driver", "", "Volume driver")

	flag.Parse()

	logInit(logLevel, nil, configFile)

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
			*driverName = globalDriver
		}
	}

	if runtime.GOOS == "windows" {
		msg := fmt.Sprintf("Support of %s driver on Windows TBD.",
			*driverName)
		log.Warning(msg)
		fmt.Println(msg)
		os.Exit(1)
	}

	log.WithFields(log.Fields{
		"driver":    *driverName,
		"log_level": *logLevel,
		"config":    *configFile,
	}).Info("Starting plugin ")

	// The Global driver doesn't depend on the config file for options
	if *driverName == globalDriver {
		driver = global.NewVolumeDriver(mountRoot, *driverName)
	} else {
		log.Warning("Unknown driver or invalid/missing driver options, exiting - ", *driverName)
		os.Exit(1)
	}

	if reflect.ValueOf(driver).IsNil() == true {
		log.Warning("Error in driver initialization exiting - ", *driverName)
		os.Exit(1)
	}

	server := NewPluginServer(*driverName, &driver)

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
