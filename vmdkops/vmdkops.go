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

// Data we return to the caller
type VolumeData struct {
	Name       string
	Attributes map[string]string
}

func Create(name string, opts map[string]string) error {
	_, err := VmdkCmd("create", name, opts)
	return err
}
func Remove(name string, opts map[string]string) error {
	_, err := VmdkCmd("remove", name, opts)
	return err
}
func Attach(name string, opts map[string]string) error {
	_, err := VmdkCmd("attach", name, opts)
	return err
}
func Detach(name string, opts map[string]string) error {
	_, err := VmdkCmd("detach", name, opts)
	return err
}
func List() ([]VolumeData, error) {
	str, err := VmdkCmd("list", "", make(map[string]string))
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
func Get(name string) (VolumeData, error) {
	volumes, err := List()
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
