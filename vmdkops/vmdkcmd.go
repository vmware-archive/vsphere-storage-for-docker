// +build linux

// The default implementation of the VmdkCmdRunner interface.
// This implementation sends synchronous commands to and receives responses from ESX.

package vmdkops

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

/*
#include "vmci/vmci_client.c"
*/
import "C"

type VmdkCmd struct{}

const (
	vmciEsxPort     int    = 15000 // port we are connecting on. TBD: config?
	commBackendName string = "vsocket"
)

// A request to be passed to ESX service
type requestToVmci struct {
	Ops     string     `json:"cmd"`
	Details VolumeInfo `json:"details"`
}

// Info we get about the volume from upstairs
type VolumeInfo struct {
	Name    string            `json:"Name"`
	Options map[string]string `json:"Opts,omitempty"`
}

type vmciError struct {
	Error string
}

//
// * Guest VM requests running an operation on ESX via vmdkops_serv.py listening on vSocket
// *
// * For each request:
// *   - Establishes a vSocket connection
// *   - Sends json string up to ESX
// *   - waits for reply and returns resulting JSON or an error

func (_ VmdkCmd) Run(cmd string, name string, opts map[string]string) ([]byte, error) {
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
	response := []byte(C.GoString(ans.buf))
	C.free(unsafe.Pointer(ans.buf))
	err = unmarshalError(response); if err != nil {
		return nil, err
	}
	// There was no error, so return the slice containing the json response
	return response, nil
}

func unmarshalError(str []byte) error {
	// Unmarshalling null always succeeds
	if string(str) == "null" {
		return nil
	}
	err_struct := vmciError{}
	err := json.Unmarshal(str, &err_struct)
	if err != nil {
		// We didn't unmarshal an error, so there is no error ;)
		return nil
	}
	// Return the unmarshaled error string as an `error`
	return errors.New(err_struct.Error)
}
