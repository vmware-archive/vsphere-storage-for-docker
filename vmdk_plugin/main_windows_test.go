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

// Basic tests for the NpipePluginServer implementation.

package main

import (
	"bufio"
	"fmt"
	"github.com/Microsoft/go-winio"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

const (
	mock           = "mock"
	requestPattern = "POST %s HTTP/1.0\r\n\r\n"
	httpOK         = "HTTP/1.0 200 OK\r\n"
)

func TestNewPluginServer(t *testing.T) {
	server := NewPluginServer(mock, nil)
	assert.NotNil(t, server, "NewPluginServer should return a server instance")
}

func initPluginServer() *NpipePluginServer {
	server := NewPluginServer(mock, nil)
	go server.Init()                   // server.Init() blocks forever
	time.Sleep(100 * time.Millisecond) // wait for goroutine to execute
	return server
}

func TestPluginServerInit(t *testing.T) {
	server := initPluginServer()
	_, err := winio.ListenPipe(npipeAddr, nil)
	assert.NotNil(t, err, "PluginServer failed to listen over npipe %s", npipeAddr)
	server.Destroy()
}

func TestPluginServerDestroy(t *testing.T) {
	server := initPluginServer()
	server.Destroy()
	listener, err := winio.ListenPipe(npipeAddr, nil)
	assert.Nil(t, err, "PluginServer failed to release npipe %s", npipeAddr)
	listener.Close()
}

func request(conn net.Conn, path string) string {
	fmt.Fprintf(conn, fmt.Sprintf(requestPattern, path))
	resp, _ := bufio.NewReader(conn).ReadString('\n')
	return resp
}

func TestPluginServerActivate(t *testing.T) {
	server := initPluginServer()
	conn, err := winio.DialPipe(npipeAddr, nil)
	assert.Nil(t, err, "Failed to dial to npipe %s", npipeAddr)
	resp := request(conn, pluginActivatePath)
	assert.Equal(t, httpOK, resp, "Plugin activate request failed")
	server.Destroy()
}
