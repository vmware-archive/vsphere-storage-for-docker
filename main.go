package main

// A VMDK Docker Data Volume plugin - main
// relies on docker/go-plugins-helpers/volume API

import (
	"flag"
	"fmt"
	"github.com/docker/go-plugins-helpers/volume"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	pluginSockDir = "/run/docker/plugins"
	vmdkPluginId  = "vmdk"
	version       = "VMDK Volume Driver v0.2"
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
	flag.Parse()
	log.Printf("%s (port: %d, mock_esx: %v)", version, *port, *mockEsx)

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChannel
		log.Printf("Received Signal:%v", sig)
		os.Remove(fullSocketAddress(vmdkPluginId))
		os.Exit(0)
	}()

	// register Docker plugin socket (.sock) and listen on it

	driver := newVmdkDriver(*mockEsx)
	handler := volume.NewHandler(driver)

	log.Print("Going into ServeUnix - Listening on Unix socket: %s", fullSocketAddress(vmdkPluginId))
	fmt.Println(handler.ServeUnix("root", fullSocketAddress(vmdkPluginId)))
}
