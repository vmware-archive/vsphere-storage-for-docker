// +build linux

package vmdkops

import (
	"encoding/json"
	"fmt"
)

//
// * VMDK CADD (Create/Attach/Detach/Delete) operations client code.
// *
// *
// * TODO: drop fprintf  and return better errors
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

type VmdkOps struct {
	Cmd VmdkCmdRunner
}

// Data we return to the caller
type VolumeData struct {
	Name       string
	Attributes map[string]string
}

func (v VmdkOps) Create(name string, opts map[string]string) error {
	_, err := v.Cmd.Run("create", name, opts)
	return err
}
func (v VmdkOps) Remove(name string, opts map[string]string) error {
	_, err := v.Cmd.Run("remove", name, opts)
	return err
}
func (v VmdkOps) Attach(name string, opts map[string]string) error {
	_, err := v.Cmd.Run("attach", name, opts)
	return err
}
func (v VmdkOps) Detach(name string, opts map[string]string) error {
	_, err := v.Cmd.Run("detach", name, opts)
	return err
}
func (v VmdkOps) List() ([]VolumeData, error) {
	str, err := v.Cmd.Run("list", "", make(map[string]string))
	if err != nil {
		return nil, err
	}
	result := make([]VolumeData, 0)
	err = json.Unmarshal(str, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
func (v VmdkOps) Get(name string) (VolumeData, error) {
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

