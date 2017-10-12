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

package config

// Read the plugin configuration file. The file is stored in JSON.
// See default-config.json at the root of the project.

import (
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/natefinch/lumberjack"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/log_formatter"
	"io/ioutil"
	"os"
	"runtime"
)

const (
	// PhotonDriver is the driver to handle single attach vmdk-based volumes in Photon Platform
	PhotonDriver = "photon"
	// VMDKDriver is the driver to handle single attach vmdk-based in vSphere/vCenter 6.0+ (deprecated)
	VMDKDriver = "vmdk"
	// VSphereDriver is the driver to handle single attach vmdk-based in vSphere/vCenter 6.0+
	VSphereDriver = "vsphere"
	// VFileDriver is a file sharing volume plugin driver
	VFileDriver = "vfile"

	// DefaultPort is the default ESX service port.
	DefaultPort = 1019

	// Default group ID to use for the socket file
	DefaultGroupID = "root"

	// Local constants
	defaultMaxLogSizeMb  = 100
	defaultMaxLogAgeDays = 28
	defaultLogLevel      = "info"
)

// Config stores the configuration for the plugin
type Config struct {
	Driver         string `json:",omitempty"`
	InternalDriver string `json:",omitempty"`
	LogPath        string `json:",omitempty"`
	MaxLogSizeMb   int    `json:",omitempty"`
	MaxLogAgeDays  int    `json:",omitempty"`
	LogLevel       string `json:",omitempty"`
	Target         string `json:",omitempty"`
	Project        string `json:",omitempty"`
	Host           string `json:",omitempty"`
	GroupID        string `json:",omitempty"`
}

// LogInfo stores parameters for setting up logs
type LogInfo struct {
	LogLevel       *string
	LogFile        *string
	DefaultLogFile string
	ConfigFile     *string
}

var config Config

// Load the configuration from a file and return a Config.
func Load(path string) (Config, error) {
	jsonBlob, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := json.Unmarshal(jsonBlob, &config); err != nil {
		return Config{}, err
	}
	setDefaults(&config)
	return config, nil
}

// setDefaults for any config setting that is at its `bottom`
func setDefaults(config *Config) {
	if config.MaxLogSizeMb == 0 {
		config.MaxLogSizeMb = defaultMaxLogSizeMb
	}
	if config.MaxLogAgeDays == 0 {
		config.MaxLogAgeDays = defaultMaxLogAgeDays
	}
	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
	}
}

// LogInit init log with passed logLevel (and get config from configFile if it's present)
// returns True if using defaults,  False if using config file
func LogInit(logInfo *LogInfo) bool {
	usingConfigDefaults := false
	c, err := Load(*logInfo.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			usingConfigDefaults = true // no .conf file, so using defaults
			c = Config{}
			setDefaults(&c)
		} else {
			panic(fmt.Sprintf("Failed to load config file %s: %v",
				*logInfo.ConfigFile, err))
		}
	}

	path := c.LogPath
	if path == "" {
		path = logInfo.DefaultLogFile
	}

	if logInfo.LogFile != nil {
		path = *logInfo.LogFile
	}
	log.SetOutput(&lumberjack.Logger{
		Filename: path,
		MaxSize:  c.MaxLogSizeMb,  // megabytes
		MaxAge:   c.MaxLogAgeDays, // days
	})

	if *logInfo.LogLevel == "" {
		*logInfo.LogLevel = c.LogLevel
	}

	level, err := log.ParseLevel(*logInfo.LogLevel)
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

// InitConfig set up driver specific options
func InitConfig(defaultConfigPath string, defaultLogPath string, defaultDriver string,
	defaultWindowsDriver string) (Config, error) {
	var err error

	// Get options from ENV (where available), and from command line.
	// ENV takes precedence, so we can modify it in Docker plugin install
	logEnv := os.Getenv("VDVS_LOG_LEVEL")
	logLevel := &logEnv
	if *logLevel == "" {
		logLevel = flag.String("log_level", "", "Logging Level")
	}
	configFile := flag.String("config", defaultConfigPath, "Configuration file path")
	driverName := flag.String("driver", "", "Volume driver")
	groupId := flag.String("group", "", "Plugin socket group id")

	flag.Parse()

	// Load the configuration if one was provided.
	config, err = Load(*configFile)
	if err != nil {
		log.Warningf("Failed to load config file %s: %v", *configFile, err)
	}

	logInfo := &LogInfo{
		LogLevel:       logLevel,
		LogFile:        nil,
		DefaultLogFile: defaultLogPath,
		ConfigFile:     configFile,
	}
	LogInit(logInfo)

	// If no driver provided on the command line,
	// then use the one in the config file or lastly the default.
	if *driverName != "" {
		config.Driver = *driverName
	} else if config.Driver == "" {
		config.Driver = defaultDriver
	}

	// If we couldn't read it from config file, set it
	// to a default value. We will check the CLI param in
	// our own driver code.
	if config.InternalDriver == "" {
		config.InternalDriver = VSphereDriver
	}

	// The windows plugin only supports the vsphere driver.
	if runtime.GOOS == "windows" && config.Driver != defaultWindowsDriver {
		msg := fmt.Sprintf("Plugin only supports the %s driver on Windows, ignoring parameter driver = %s.",
			defaultWindowsDriver, config.Driver)
		log.Warning(msg)
		fmt.Println(msg)
		config.Driver = defaultWindowsDriver
	}

	// If no driver provided on the command line, use the one in the
	// environment and if thats not available then the one in the
	// config file or lastly the default.
	envGid := os.Getenv("VDVS_SOCKET_GID")
	if *groupId != "" {
		config.GroupID = *groupId
	} else if envGid != "" {
		config.GroupID = envGid
	} else if config.GroupID == "" {
		config.GroupID = DefaultGroupID
	}

	log.WithFields(log.Fields{
		"driver":    config.Driver,
		"log_level": *logLevel,
		"config":    *configFile,
		"group":     config.GroupID,
	}).Info("Starting plugin ")

	return config, nil

}

func GetConfig() Config {
	return config
}
