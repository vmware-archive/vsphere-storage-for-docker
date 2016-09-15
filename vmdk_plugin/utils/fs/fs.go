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

// This is the filesystem interface for mounting volumes on the guest.

package fs

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const sysPciDevs = "/sys/bus/pci/devices" // All PCI devices on the host
const sysPciSlots = "/sys/bus/pci/slots"  // PCI slots on the host
const pciAddrLen = 10                     // Length of PCI dev addr

// FstypeDefault contains the default FS when not specified by the user
const FstypeDefault = "ext4"

// VolumeDevSpec - volume spec returned from the server on an attach
type VolumeDevSpec struct {
	Unit                    string
	ControllerPciSlotNumber string
}

// Mkdir creates a directory at the specified path
func Mkdir(path string) error {
	stat, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if stat != nil && !stat.IsDir() {
		return fmt.Errorf("%v already exist and it's not a directory", path)
	}
	return nil
}

// Mkfs creates a filesystem at the specified device
func Mkfs(mkfscmd string, label string, device string) error {	
	out, err := exec.Command(mkfscmd, "-L", label, device).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to create filesystem on %s: %s. Output = %s",
			device, err, out)
	}
	return nil
}

// MkfsLookup lookups supported filesystems based on mkfs tools
func MkfsLookup() string {
	mkftools, _ := filepath.Glob("/sbin/mkfs.*")
	var supportedFs string
	for _, fs := range mkftools {
		if supportedFs != "" {
			supportedFs += ", "+strings.Split(fs, ".")[1]
		} else {
			supportedFs += strings.Split(fs, ".")[1]
		}
	}
	return supportedFs
}

// Mount the filesystem (`fs`) on the device at the given mount point.
func Mount(mountpoint string, fstype string, device string, isReadOnly bool) error {
	log.WithFields(log.Fields{
		"device":     device,
		"fstype":     fstype,
		"mountpoint": mountpoint,
	}).Debug("Calling syscall.Mount() ")

	flags := 0
	if isReadOnly {
		flags = syscall.MS_RDONLY
	}
	err := syscall.Mount(device, mountpoint, fstype, uintptr(flags), "")
	if err != nil {
		return fmt.Errorf("Failed to mount device %s at %s: %s", device, mountpoint, err)
	}
	return nil
}

// Unmount a device from the given mount point.
func Unmount(mountpoint string) error {
	return syscall.Unmount(mountpoint, 0)
}

// GetDevicePath - return device path or error
func GetDevicePath(str []byte) (string, error) {
	var volDev VolumeDevSpec
	err := json.Unmarshal(str, &volDev)
	if err != nil && len(err.Error()) != 0 {
		return "", err
	}

	// Get the device node for the unit returned from the attach.
	// Lookup each device that has a label and if that label matches
	// the one for the given bus number.
	// The device we need is then constructed from the dir name with
	// the matching label.
	pciSlotAddr := fmt.Sprintf("%s/%s/address", sysPciSlots, volDev.ControllerPciSlotNumber)

	fh, err := os.Open(pciSlotAddr)
	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Warn("Get device path failed for unit# %s @ PCI slot %s: ",
			volDev.Unit, volDev.ControllerPciSlotNumber)
		return "", fmt.Errorf("Device not found")
	}

	buf := make([]byte, pciAddrLen)
	_, err = fh.Read(buf)

	fh.Close()
	if err != nil && err != io.EOF {
		log.WithFields(log.Fields{"Error": err}).Warn("Get device path failed for unit# %s @ PCI slot %s: ",
			volDev.Unit, volDev.ControllerPciSlotNumber)
		return "", fmt.Errorf("Device not found")
	}
	return fmt.Sprintf("/dev/disk/by-path/pci-%s.0-scsi-0:0:%s:0", string(buf), volDev.Unit), nil

}
