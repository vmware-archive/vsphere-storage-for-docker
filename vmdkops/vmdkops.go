// +build linux
package vmdkops

import (
	"encoding/json"
	"fmt"
	"github.com/docker/go-plugins-helpers/volume"
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

type VolumeError struct {
	msg string
}

func (e *VolumeError) Error() string {
	return e.msg
}

// Send a command 'cmd' to VMCI, via C API
// Return the resulting JSON or an error. Each Public API function will decode
// the JSON corresponding to it's return type, and return an error if decoding fails.
func vmdkCmd(cmd string, name string, opts map[string]string) ([]byte, error) {
	json_str, err := json.Marshal(&requestToVmci{
		Ops:     cmd,
		Details: VolumeInfo{Name: name, Options: opts}})
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal json: %v", err)
	}

	cmd_s := C.CString(string(json_str))
	defer C.free(unsafe.Pointer(cmd_s))

	be_s := C.CString(commBackendName)
	defer C.free(unsafe.Pointer(be_s))

	// Get the response data in json
	ans := (*C.be_answer)(C.malloc(C.sizeof_struct_be_answer))
	defer C.free(unsafe.Pointer(ans))

	// connect, send command, get reply, disconnect - all in one shot
	ret := C.Vmci_GetReply(C.int(vmciEsxPort), cmd_s, be_s, ans)

	if ret != 0 {
		log.Print("Warning - no connection to ESX over vsocket, trace only")
		return nil, fmt.Errorf("vmdkCmd err: %d (%s)", ret, syscall.Errno(ret).Error())
	}
	return []byte(C.GoString((*C.char)(unsafe.Pointer(&ans.buf)))), nil
}

// public API
func (v VolumeInfo) Create() string {
	_, err := vmdkCmd("create", v.Name, v.Options)
	if err != nil {
		return err.Error()
	}
	return ""
}
func (v VolumeInfo) Remove() string {
	_, err := vmdkCmd("create", v.Name, v.Options)
	if err != nil {
		return err.Error()
	}
	return ""
}
func (v VolumeInfo) Attach() string {
	_, err := vmdkCmd("create", v.Name, v.Options)
	if err != nil {
		return err.Error()
	}
	return ""
}
func (v VolumeInfo) Detach() string {
	_, err := vmdkCmd("create", v.Name, v.Options)
	if err != nil {
		return err.Error()
	}
	return ""
}
func (v VolumeInfo) List() string {
	_, err := vmdkCmd("list", v.Name, v.Options)
	if err != nil {
		return err.Error()
	}
	return ""
}

func VmdkCreate(name string, opts map[string]string) string {
	_, err := vmdkCmd("create", name, opts)
	if err != nil {
		return err.Error()
	}
	return ""
}
func VmdkRemove(name string, opts map[string]string) string {
	_, err := vmdkCmd("remove", name, opts)
	if err != nil {
		return err.Error()
	}
	return ""
}
func VmdkAttach(name string, opts map[string]string) string {
	_, err := vmdkCmd("attach", name, opts)
	if err != nil {
		return err.Error()
	}
	return ""
}
func VmdkDetach(name string, opts map[string]string) string {
	_, err := vmdkCmd("detach", name, opts)
	if err != nil {
		return err.Error()
	}
	return ""
}
func VmdkList() ([]volume.Volume, error) {
	str, err := vmdkCmd("list", "", make(map[string]string))
	if err != nil {
		return nil, err
	}
	result := make([]volume.Volume, 0)
	err = json.Unmarshal(str, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func TestSetDummyBackend() {
	commBackendName = "dummy"
}
