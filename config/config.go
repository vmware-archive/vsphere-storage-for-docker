package config

// Read the plugin configuration file. The file is stored in JSON.
// See default-config.json at the root of the project.

import (
	"encoding/json"
	"io/ioutil"
)

const (
	defaultLogPath       = "/var/log/docker-vmdk-plugin.log"
	defaultMaxLogSizeMb  = 100
	defaultMaxLogAgeDays = 28
)

// Config stores the configuration for the plugin
type Config struct {
	LogPath       string `json:",omitempty"`
	MaxLogSizeMb  int    `json:",omitempty"`
	MaxLogAgeDays int    `json:",omitempty"`
}

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
	SetDefaults(&config)
	return config, nil
}

// SetDefaults for any config setting that is at its `bottom`
func SetDefaults(config *Config) {
	if config.LogPath == "" {
		config.LogPath = defaultLogPath
	}
	if config.MaxLogSizeMb == 0 {
		config.MaxLogSizeMb = defaultMaxLogSizeMb
	}
	if config.MaxLogAgeDays == 0 {
		config.MaxLogAgeDays = defaultMaxLogAgeDays
	}
}
