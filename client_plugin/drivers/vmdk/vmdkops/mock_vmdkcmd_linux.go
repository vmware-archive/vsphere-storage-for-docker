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

// An implementation of the VmdkCmdRunner interface that mocks ESX. This removes the requirement forunning ESX at all when testing the plugin.

package vmdkops

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/vmware/docker-volume-vsphere/client_plugin/utils/fs"
)

// MockVmdkCmd struct
type MockVmdkCmd struct{}

const (
	backingRoot     = "/tmp/docker-volumes" // Files for loopback device backing stored here
	fileSizeInBytes = 100 * 1024 * 1024     // file size for loopback block device
)

// NewMockCmd returns a new instance of MockVmdkCmd.
func NewMockCmd() MockVmdkCmd {
	return MockVmdkCmd{}
}

func getBackingFileName(nameBase string) string {
	// make unique name - avoid clashes shall we find any garbage in the directory
	name := fmt.Sprintf("%s/%d/%s", backingRoot, os.Getpid(), nameBase)
	log.WithFields(log.Fields{"name": name}).Debug("Created tmp file for loopback")
	return name
}

// Run returns JSON responses to each command or an error
func (mockCmd MockVmdkCmd) Run(cmd string, name string, opts map[string]string) ([]byte, error) {
	// We store no in memory state, so just try to recreate backingRoot every time
	rootName := fmt.Sprintf("%s/%d", backingRoot, os.Getpid())
	err := fs.Mkdir(rootName)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"cmd": cmd}).Debug("Running Mock Cmd")
	switch cmd {
	case "create":
		err := createBlockDevice(name, opts)
		return nil, err
	case "list":
		return list()
	case "get":
		return nil, get(name)
	case "attach":
		return getBlockDeviceForName(name)
	case "detach":
		return nil, nil
	case "remove":
		err := remove(name)
		return nil, err
	}
	return []byte("null"), nil
}

func list() ([]byte, error) {
	rootName := fmt.Sprintf("%s/%d", backingRoot, os.Getpid())
	files, err := ioutil.ReadDir(rootName)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s", backingRoot)
	}
	volumes := make([]VolumeData, 0, len(files))
	for _, file := range files {
		volumes = append(volumes, VolumeData{Name: file.Name()})
	}
	return json.Marshal(volumes)
}

// validates that the volume exists, returns error or nil (for OK)
func get(name string) error {
	filePath := getBackingFileName(name)
	if _, err := os.Lstat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("Unknown volume %s", name)
	} else if err != nil {
		return err
	}
	return nil
}

func remove(name string) error {
	backing := getBackingFileName(name)
	out, err := exec.Command("blkid", []string{"-L", name}...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to find device for backing file %s via blkid", backing)
	}
	device := strings.TrimRight(string(out), " \n")
	fmt.Printf("Detaching loopback device %s\n", device)
	out, err = exec.Command("losetup", "-d", device).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to detach loopback device node %s with error: %s. Output = %s",
			device, err, out)
	}
	err = os.Remove(backing)
	if err != nil {
		return fmt.Errorf("Failed to remove backing file %s: %s", backing, err)
	}
	return os.Remove(device)
}

func createBlockDevice(label string, opts map[string]string) error {
	backing := getBackingFileName(label)
	err := createBackingFile(backing)
	if err != nil {
		return err
	}
	loopbackCount := getMaxLoopbackCount() + 1
	device := fmt.Sprintf("/dev/loop%d", loopbackCount)
	err = createDeviceNode(device, loopbackCount)
	if err != nil {
		return err
	}
	// Ignore output. This is to prevent spurious failures from old devices
	// that were removed, but not detached.
	exec.Command("losetup", "-d", device).CombinedOutput()
	err = setupLoopbackDevice(backing, device)
	if err != nil {
		return err
	}
	// Use default fstype if not specified
	if _, result := opts["fstype"]; result == false {
		opts["fstype"] = fs.FstypeDefault
	}
	errFstype := fs.VerifyFSSupport(opts["fstype"])
	if errFstype != nil {
		return fmt.Errorf("Not found mkfs for %s", opts["fstype"])
	}
	return fs.MkfsByDevicePath(opts["fstype"], label, device)
}

func getBlockDeviceForName(name string) ([]byte, error) {
	backing := getBackingFileName(name)
	out, err := exec.Command("blkid", []string{"-L", name}...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Failed to find device for backing file %s via blkid", backing)
	}
	device := strings.TrimRight(string(out), " \n")
	return []byte(device), nil
}

func getMaxLoopbackCount() int {
	// always start at 1000
	count := 1000
	files, err := ioutil.ReadDir("/dev")
	if err != nil {
		panic("Failed to read /dev")
	}
	for _, file := range files {
		trimmed := strings.TrimPrefix(file.Name(), "loop")
		if s, err := strconv.Atoi(trimmed); err == nil {
			if s > count {
				count = s
			}
		}
	}
	return count
}

func createBackingFile(backing string) error {
	flags := syscall.O_RDWR | syscall.O_CREAT | syscall.O_EXCL
	file, err := os.OpenFile(backing, flags, 0755)
	if err != nil {
		return fmt.Errorf("Failed to create backing file %s: %s", backing, err)
	}
	err = syscall.Fallocate(int(file.Fd()), 0, 0, fileSizeInBytes)
	if err != nil {
		return fmt.Errorf("Failed to allocate %s: %s", backing, err)
	}
	return nil
}

func createDeviceNode(device string, loopbackCount int) error {
	count := fmt.Sprintf("%d", loopbackCount)
	out, err := exec.Command("mknod", device, "b", "7", count).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to make device node %s: %s. Output = %s",
			device, err, out)
	}
	return nil
}

func setupLoopbackDevice(backing string, device string) error {
	out, err := exec.Command("losetup", device, backing).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to setup loopback device node %s for backing file %s: %s. Output = %s",
			device, backing, err, out)
	}
	return nil
}
