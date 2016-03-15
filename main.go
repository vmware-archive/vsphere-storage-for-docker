package main

// A VMDK Docker Data Volume plugin - main
// relies on docker/go-plugins-helpers/volume API

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/natefinch/lumberjack"
	"github.com/vmware/docker-vmdk-plugin/config"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	pluginSockDir     = "/run/docker/plugins"
	vmdkPluginID      = "vmdk"
	version           = "VMDK Volume Driver v0.3"
	defaultConfigPath = "/etc/docker-vmdk-plugin.conf"
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

func main() {
	// connect to this socket
	port := flag.Int("port", 15000, "Default port for vmci")
	mockEsx := flag.Bool("mock_esx", false, "Mock the ESX server")
	logLevel := flag.String("log_level", "info", "Logging Level")
	configFile := flag.String("config", defaultConfigPath, "Configuration file path")
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse log level: %v", err))
	}

	usingConfigDefaults := false
	c, err := config.Load(*configFile)
	if err != nil {
		if os.IsNotExist(err) {
			usingConfigDefaults = true
			c = config.Config{}
			config.SetDefaults(&c)
		} else {
			panic(fmt.Sprintf("Failed to load config file %s: %v", *configFile, err))
		}
	}

	log.SetOutput(&lumberjack.Logger{
		Filename: c.LogPath,
		MaxSize:  c.MaxLogSizeMb,  // megabytes
		MaxAge:   c.MaxLogAgeDays, // days
	})

	log.SetFormatter(new(VmwareFormatter))
	log.SetLevel(level)

	if usingConfigDefaults {
		log.Warn("No config file found. Using defaults.")
	}

	log.WithFields(log.Fields{
		"version":   version,
		"port":      *port,
		"mock_esx":  *mockEsx,
		"log_level": *logLevel,
		"config":    *configFile,
	}).Info("Docker VMDK plugin started ")

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChannel
		log.WithFields(log.Fields{"signal": sig}).Error("Received Signal")
		os.Remove(fullSocketAddress(vmdkPluginID))
		os.Exit(0)
	}()

	// register Docker plugin socket (.sock) and listen on it

	driver := newVmdkDriver(*mockEsx)
	handler := volume.NewHandler(driver)

	log.WithFields(log.Fields{
		"address": fullSocketAddress(vmdkPluginID),
	}).Info("Going into ServeUnix - Listening on Unix socket ")

	log.Info(handler.ServeUnix("root", fullSocketAddress(vmdkPluginID)))
}
