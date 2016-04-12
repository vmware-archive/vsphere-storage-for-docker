// +build linux

package vmdkops

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
)

//
// * VMDK CADD (Create/Attach/Detach/Delete) operations client code.
// *
// * TODO: allow concurrency from multiple containers. Specifically, split into
// * Vmci_SubmitRequst() [not blocking] and Vmci_GetReply() [blocking] so the
// * goroutines can be concurrent
// *
// * TODO: add better length mgmt for ANSW_BUFSIZE
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

// Attach a volume
func (v VmdkOps) Attach(name string, opts map[string]string) ([]byte, error) {
	log.Debugf("vmdkOps.Attach name=%s", name)
        str, err := v.Cmd.Run("attach", name, opts)
        if err != nil {
           return nil, err
        }
        return str, err
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
func (v VmdkOps) Get(name string) (VolumeData, error) {
	log.Debugf("vmdkOps.Get name=%s", name)
	volumes, err := v.List()
	if err != nil {
		return VolumeData{}, err
	}
	for _, vol := range volumes {
		if vol.Name == name {
			return vol, nil
		}
	}
	return VolumeData{}, fmt.Errorf("Volume does not exist: %s", name)
}
