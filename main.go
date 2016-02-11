package main

// A VMDK Docker Data Volume plugin - main
// relies on docker/go-plugins-helpers/volume API

import (
	"fmt"
	"github.com/docker/go-plugins-helpers/volume"
	"log"
	//	"os"
	//	"os/signal"
	//	"syscall"
	"flag"
)

const (
	plugingSockDir = "/run/docker/plugins"
	vmdkPluginId   = "vmdk"
	version        = "VMDK Volume Driver v0.1"
)

func main() {
	// connect to this socket
	port := flag.Int("port", 15000, "default port  for vmci")
	flag.Parse()

	log.Printf("%s (port: %d - ignored and it's OK)", version, *port)
	//	TODO: register signal handling
	//	sigChannel := make(chan os.Signal, 1)
	//	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	//
	//	go func() {
	//		sig := <-sigChannel
	//		log.Printf("got signal %v", sig)
	////		os.Remove(getPath(sockFile))
	//		os.Exit(3) // TODO: call gracious exit
	//	}()

	// register Docker plugin socket (.sock) and listen on it

	driver := newVmdkDriver()
	handler := volume.NewHandler(driver)

	log.Print("Going into ServeUnix - listening on Unix socket")
	fmt.Println(handler.ServeUnix("root", vmdkPluginId))

}
