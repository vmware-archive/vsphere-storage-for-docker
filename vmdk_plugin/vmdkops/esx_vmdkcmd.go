// Copyright 2016 VMware, Inc. All Rights Reserved.
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

// +build linux

// The default (ESX) implementation of the VmdkCmdRunner interface.
// This implementation sends synchronous commands to and receives responses from ESX.

package vmdkops

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	log "github.com/Sirupsen/logrus"
)

/*
#cgo CFLAGS: -I ../../esx_service/vmci
#include "vmci_client.c"
*/
import "C"

// EsxVmdkCmd struct - empty , we use it only to implement VmdkCmdRunner interface
type EsxVmdkCmd struct {
	Mtx *sync.Mutex // For serialization of Run comand/response
}

const (
	commBackendName string = "vsocket"
	maxRetryCount          = 5
)

// A request to be passed to ESX service
type requestToVmci struct {
	Ops     string     `json:"cmd"`
	Details VolumeInfo `json:"details"`
}

// VolumeInfo we get about the volume from upstairs
type VolumeInfo struct {
	Name    string            `json:"Name"`
	Options map[string]string `json:"Opts,omitempty"`
}

type vmciError struct {
	Error string `json:",omitempty"`
}

// EsxPort used to connect to ESX, passed in as command line param
var EsxPort int

// Run command Guest VM requests on ESX via vmdkops_serv.py listening on vSocket
// *
// * For each request:
// *   - Establishes a vSocket connection
// *   - Sends json string up to ESX
// *   - waits for reply and returns resulting JSON or an error
func (vmdkCmd EsxVmdkCmd) Run(cmd string, name string, opts map[string]string) ([]byte, error) {
	vmdkCmd.Mtx.Lock()
	defer vmdkCmd.Mtx.Unlock()

	jsonStr, err := json.Marshal(&requestToVmci{
		Ops:     cmd,
		Details: VolumeInfo{Name: name, Options: opts}})
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal json: %v", err)
	}

	cmdS := C.CString(string(jsonStr))
	defer C.free(unsafe.Pointer(cmdS))

	beS := C.CString(commBackendName)
	defer C.free(unsafe.Pointer(beS))

	// Get the response data in json
	ans := (*C.be_answer)(C.malloc(C.sizeof_struct_be_answer))
	defer C.free(unsafe.Pointer(ans))

	for i := 0; i <= maxRetryCount; i++ {
		_, err = C.Vmci_GetReply(C.int(EsxPort), cmdS, beS, ans)
		if err != nil {
			var errno syscall.Errno
			errno = err.(syscall.Errno)
			msg := fmt.Sprintf("'%s' failed: %v (errno=%d).", cmd, err, int(errno))

			// Still below maximum number of retries, log and continue
			if i < maxRetryCount {
				log.Warnf(msg + " Retrying...")
				time.Sleep(time.Second * 1)
				continue
			}

			if errno == syscall.ECONNRESET || errno == syscall.ETIMEDOUT {
				msg += " Cannot communicate with ESX, please refer to the FAQ https://github.com/vmware/docker-volume-vsphere/wiki#faq"
			}
			log.Warnf(msg)
			return nil, errors.New(msg)
		}
		// Received no error, exit loop
		break
	}

	response := []byte(C.GoString(ans.buf))
	C.free(unsafe.Pointer(ans.buf))
	err = unmarshalError(response)
	if err != nil && len(err.Error()) != 0 {
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
	errStruct := vmciError{}
	err := json.Unmarshal(str, &errStruct)
	if err != nil {
		// We didn't unmarshal an error, so there is no error ;)
		return nil
	}
	// Return the unmarshaled error string as an `error`
	return errors.New(errStruct.Error)
}
