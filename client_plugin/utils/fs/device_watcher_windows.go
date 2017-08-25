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

// Device watching logic for Windows OS.
// This code is to be called when a device shows on a bus. The code will ensure
// that a newly attached disk is ready for use by waiting until it's fully accessible.

package fs

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	ps "github.com/vmware/docker-volume-vsphere/client_plugin/utils/powershell"
)

const (
	devChangeTimeoutSec = 1 // Number of seconds to wait for a Win32_DeviceChangeEvent

	// Using a PowerShell script here due to lack of a functional Go WMI library.
	// PowerShell script to register and wait for a Win32_DeviceChangeEvent.
	// The script blocks until a Win32_DeviceChangeEvent occurs or the event times out.
	devChangeWaitScript = `
		Unregister-Event -SourceIdentifier DeviceChangeEvent -ErrorAction 'SilentlyContinue' -Force -Confirm:$false;
		Register-WmiEvent -Class Win32_DeviceChangeEvent -SourceIdentifier DeviceChangeEvent;
		$event = Wait-Event -SourceIdentifier DeviceChangeEvent -Timeout %d;
		If ($event) {
			Write-Host "DeviceChangeEvent";
		}
		Else {
			Write-Host "EventTimedOut";
		};
	`
)

// DeviceWatcher represents a device change watcher.
type DeviceWatcher struct {
	Event      chan string   // chan for emitting events
	Error      chan error    // chan for emitting errors
	stop       bool          // flag to initiate watcher termination
	terminated chan struct{} // chan to notify watcher termination status
}

// NewDeviceWatcher returns a new instance of DeviceWatcher.
func NewDeviceWatcher() *DeviceWatcher {
	return &DeviceWatcher{Event: make(chan string), Error: make(chan error),
		stop: false, terminated: make(chan struct{})}
}

// Init starts a goroutine that emits device events/errors via watcher channels.
func (w *DeviceWatcher) Init() {
	go func() {
		for !w.stop {
			log.Info("Waiting for a device change event ")
			w.awaitEventAndEmit()
		}
		close(w.Event)
		close(w.Error)
		log.Info("Watcher channels closed ")
		w.terminated <- struct{}{}
	}()
}

// awaitEventAndEmit awaits and emits a device change event or an error.
func (w *DeviceWatcher) awaitEventAndEmit() {
	script := fmt.Sprintf(devChangeWaitScript, devChangeTimeoutSec)
	stdout, stderr, err := ps.Exec(script)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "stdout": stdout,
			"stderr": stderr}).Error("Failed to watch device event ")
		select {
		case w.Error <- err:
			log.WithFields(log.Fields{"err": err, "stdout": stdout,
				"stderr": stderr}).Info("Successfully emitted error ")
		default:
			log.WithFields(log.Fields{"err": err, "stdout": stdout,
				"stderr": stderr}).Warn("Couldn't emit error, continuing.. ")
		}
		return
	}

	log.WithFields(log.Fields{"stdout": stdout}).Info("Watcher script executed ")
	event := tailSegment(stdout, lf, 2)
	select {
	case w.Event <- event:
		log.WithFields(log.Fields{"event": event}).Info("Successfully emitted event ")
	default:
		log.WithFields(log.Fields{"event": event}).Warn("Couldn't emit event, continuing.. ")
	}
}

// Terminate initiates watcher termination and returns immediately.
func (w *DeviceWatcher) Terminate() {
	log.Info("Initiated watcher termination ")
	w.stop = true
}

// AwaitTermination blocks until watcher termination is completed.
func (w *DeviceWatcher) AwaitTermination() {
	log.Info("Awaiting watcher termination ")
	<-w.terminated
	close(w.terminated)
	log.Info("Watcher termination complete ")
}
