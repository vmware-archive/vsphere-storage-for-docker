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
	"strings"
	"syscall"
)

const sysPciDevs = "/sys/bus/pci/devices" // All PCI devices on the host

// VolumeDevSpec - volume spec returned from the server on an attach
type VolumeDevSpec struct {
	Unit string
	Bus  string
}

// Mkdir creates a directory at the specifiied path
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

// Mount discovers which devices are for which volume using blkid.
// It then mounts the filesystem (`fs`) on the device at the given mountpoint.
func Mount(mountpoint string, fs string, device string) error {
	log.WithFields(log.Fields{
		"device":     device,
		"mountpoint": mountpoint,
	}).Debug("Calling syscall.Mount() ")

	err := syscall.Mount(device, mountpoint, fs, 0, "")
	if err != nil {
		return fmt.Errorf("Failed to mount device %s at %s: %s", device, mountpoint, err)
	}
	return nil
}

// Unmount a device from the given mountpoint.
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
	busLabel := fmt.Sprintf("SCSI%s", volDev.Bus)

	dirh, err := os.Open(sysPciDevs)
	if err != nil {
		return "", err
	}

	defer dirh.Close()

	// Change this to do a read in slices, needed for large dirs.
	names, err := dirh.Readdirnames(-1)

	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Warn("Get device by path: ")
		return "", err
	}

	// Only read in as much as the size of the label we want to compare.
	buf := make([]byte, len(busLabel))
	for _, elem := range names {
		label := fmt.Sprintf("%s/%s/%s", sysPciDevs, elem, "label")
		if _, err = os.Stat(label); os.IsNotExist(err) {
			continue
		}
		labelh, err := os.Open(label)
		if err != nil {
			log.WithFields(log.Fields{"Error": err}).Warn("Get device by path: ")
			return "", err
		}
		_, err = labelh.Read(buf)

		labelh.Close()
		if err != nil && err != io.EOF {
			log.WithFields(log.Fields{"Error": err}).Warn("Get device by path: ")
			return "", err
		}
		if strings.Compare(busLabel, string(buf)) == 0 {
			// Return the device string by path for the device.
			return fmt.Sprintf("/dev/disk/by-path/pci-%s-scsi-0:0:%s:0", elem, volDev.Unit), nil
		}
	}

	return "", fmt.Errorf("Device not found for unit %s on bus %s", volDev.Unit, volDev.Bus)
}
