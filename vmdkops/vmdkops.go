// +build linux
package vmdkops

import (
	"encoding/json"
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

//
// * VMDK CADD (Create/Attach/Detach/Delete) operations client code.
// *
// * Requests operation from a Guest VM (sending json request over vSocket),
// * and expect the vmdkops_srv.py on the ESX host listening on vSocket.
// *
// * For each request:
// *   - Establishes a vSocket connection
// *   - Sends json string up to ESX
// *   - waits for reply and returns it
// *
// * Each requests has 4 bytes MAGIC, then 4 byts length (msg string length),
// * then 'length' + 1 null-terminated JSON string with command
// * On reply, returns MAGIC, length and message ("" or error message.)
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

/*
#include "vmci/vmci_client.c"
*/
import "C"

const (
	vmciEsxPort int = 15000 // port we are connecting on. TBD: config?
)

var commBackendName string = "vsocket" // could be changed in test

// Info we get about the volume from upstairs
type VolumeInfo struct {
	Name    string            `json:"Name"`
	Options map[string]string `json:"Opts,omitempty"`
}

// A request to be passed to ESX service
type requestToVmci struct {
	Ops     string     `json:"cmd"`
	Details VolumeInfo `json:"details"`
}

// Send a command 'cmd' to VMCI, via C API
func vmdkCmd(cmd string, name string, opts map[string]string) string {

	json_str, err := json.Marshal(&requestToVmci{
		Ops:     cmd,
		Details: VolumeInfo{Name: name, Options: opts}})
	if err != nil {
		return fmt.Sprintf("Failed to marshal json: %s", err)
	}

	cmd_s := C.CString(string(json_str))
	defer C.free(unsafe.Pointer(cmd_s))

	be_s := C.CString(commBackendName)
	defer C.free(unsafe.Pointer(be_s))

	// connect, send command, get reply, disconnect - all in one shot
	ret := C.Vmci_GetReply(C.int(vmciEsxPort), cmd_s, be_s)

	if ret != 0 {
		log.Print("Warning - no connection to ESX over vsocket, trace only")
		return fmt.Sprintf("vmdkCmd err: %d (%s)", ret, syscall.Errno(ret).Error())
	}

	return ""
}

// public API
func (v VolumeInfo) Create() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) Remove() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) Attach() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) Detach() string {
	return vmdkCmd("create", v.Name, v.Options)
}
func (v VolumeInfo) List() string {
	return vmdkCmd("list", v.Name, v.Options)
}

func VmdkCreate(name string, opts map[string]string) string {
	return vmdkCmd("create", name, opts)
}
func VmdkRemove(name string, opts map[string]string) string {
	return vmdkCmd("remove", name, opts)
}
func VmdkAttach(name string, opts map[string]string) string {
	return vmdkCmd("attach", name, opts)
}
func VmdkDetach(name string, opts map[string]string) string {
	return vmdkCmd("detach", name, opts)
}
func VmdkList(name string, opts map[string]string) string {
	return vmdkCmd("list", name, opts)
}

func TestSetDummyBackend() {
	commBackendName = "dummy"
}
