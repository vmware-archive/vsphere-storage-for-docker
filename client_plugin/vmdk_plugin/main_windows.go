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

// A VMDK Docker Data Volume plugin for Windows
package main

import (
	"github.com/kardianos/service"
	log "github.com/Sirupsen/logrus"
)

var logger service.Logger

type program struct{}

func (p *program) Start(s service.Service) error {
    // Start should not block. Do the actual work async.
    go p.run()
    return nil
}

func (p *program) run() {
	// Do work here
	startPluginServer()
}

func (p *program) Stop(s service.Service) error {
    // Stop should not block. Return directly.
    return nil
}

// Run vDVS plugin on Windows
func runVdvs() {
	svcConfig := &service.Config {
        Name:        "vdvs",
        DisplayName: "vSphere Docker Volume Service",
        Description: "Enables user to run stateful containerized applications on top of VMware vSphere.",
    }

    prg := &program{}
    svc, err := service.New(prg, svcConfig)
    if err != nil {
        log.Fatal("Failed to create the service: ", err)
    }

	logger, err = svc.Logger(nil)
    if err != nil {
        log.Fatal(err)
	}
	
    err = svc.Run()
    if err != nil {
		logger.Error(err)
    }
}