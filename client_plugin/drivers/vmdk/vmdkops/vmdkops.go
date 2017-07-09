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

// +build linux windows

package vmdkops

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/fs"
)

//
// * VMDK CADD (Create/Attach/Detach/Delete) operations client code.
// *
// **** PREREQUISITES:
//   Build: open-vm-tools has to be installed - provided "vmci/vmci_sockets.h"
//   Run:   open-vm-tools has to be installed
//

// VmdkCmdRunner interface for sending Vmdk Commands to an ESX server.
type VmdkCmdRunner interface {
	Run(cmd string, name string, opts map[string]string) ([]byte, error)
}

// VmdkOps struct
type VmdkOps struct {
	Cmd VmdkCmdRunner // see *_vmdkcmd.go for implementations.
}

// VolumeData we return to the caller
type VolumeData struct {
	Name       string
	Attributes map[string]string
}

// Create a volume
func (v VmdkOps) Create(name string, opts map[string]string) error {
	log.Debugf("vmdkOp.Create name=%s", name)
	_, err := v.Cmd.Run("create", name, opts)
	return err
}

// Remove a volume
func (v VmdkOps) Remove(name string, opts map[string]string) error {
	log.Debugf("vmdkOps.Remove name=%s", name)
	_, err := v.Cmd.Run("remove", name, opts)
	return err
}

// RawAttach attaches a volume and returns `[]byte` representing the raw response string.
func (v VmdkOps) RawAttach(name string, opts map[string]string) ([]byte, error) {
	log.Debugf("vmdkOps.Attach name=%s", name)
	str, err := v.Cmd.Run("attach", name, opts)
	if err != nil {
		log.WithFields(log.Fields{"name": name, "opts": opts, "error": err}).Error("RawAttach failed ")
		return nil, err
	}
	return str, nil
}

// Attach attaches a volume and returns the disk's VolumeDevSpec.
func (v VmdkOps) Attach(name string, opts map[string]string) (*fs.VolumeDevSpec, error) {
	str, err := v.RawAttach(name, opts)
	if err != nil {
		return nil, err
	}

	var volDev fs.VolumeDevSpec
	err = json.Unmarshal(str, &volDev)
	if err != nil {
		log.WithFields(log.Fields{"name": name, "opts": opts, "bytes": str,
			"error": err}).Error("Failed to unmarshal, detaching volume ")
		// RawAttach may have the volume attached to this client, so detach.
		errDetach := v.Detach(name, nil)
		if errDetach != nil {
			log.WithFields(log.Fields{"name": name,
				"error": errDetach}).Warning("Detach volume failed ")
		}
		return nil, err
	}
	return &volDev, nil
}

// Detach a volume
func (v VmdkOps) Detach(name string, opts map[string]string) error {
	log.Debugf("vmdkOps.Detach name=%s", name)
	_, err := v.Cmd.Run("detach", name, opts)
	return err
}

// List all volumes
func (v VmdkOps) List() ([]VolumeData, error) {
	log.Debugf("vmdkOps.List")
	str, err := v.Cmd.Run("list", "", make(map[string]string))
	if err != nil {
		return nil, err
	}

	var result []VolumeData
	err = json.Unmarshal(str, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Get for volume
func (v VmdkOps) Get(name string) (map[string]interface{}, error) {
	log.Debugf("vmdkOps.Get name=%s", name)
	str, err := v.Cmd.Run("get", name, make(map[string]string))
	if err != nil {
		return nil, err
	}

	var statusMap map[string]interface{}
	statusMap = make(map[string]interface{})

	err = json.Unmarshal(str, &statusMap)
	if err != nil {
		log.Warn("vmdkOps.Get failed decoding volume status for name=%s", name)
	}
	return statusMap, nil
}
