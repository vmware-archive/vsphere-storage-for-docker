package main

// A VMDK Docker Data Volume plugin - main
// relies on docker/go-plugins-helpers/volume API

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	pluginSockDir = "/run/docker/plugins"
	vmdkPluginID  = "vmdk"
	version       = "VMDK Volume Driver v0.3"
	logPath       = "/var/log/docker-vmdk-plugin.log"
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
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.WithFields(log.Fields{"level": *logLevel}).Panic("Invalid Log Level")
	}

	flags := syscall.O_APPEND | syscall.O_CREAT | syscall.O_WRONLY
	file, err := os.OpenFile(logPath, flags, 0755)
	if err != nil {
		log.WithFields(log.Fields{"path": logPath, "error": err}).Panic("Failed to open logfile")
	}

	log.SetOutput(file)
	log.SetFormatter(new(VmwareFormatter))
	log.SetLevel(level)

	log.WithFields(log.Fields{
		"version":   version,
		"port":      *port,
		"mock_esx":  *mockEsx,
		"log_level": *logLevel,
	}).Info("Docker VMDK plugin started")

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
	}).Info("Going into ServeUnix - Listening on Unix socket")

	log.Info(handler.ServeUnix("root", fullSocketAddress(vmdkPluginID)))
}
