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
	"golang.org/x/exp/inotify"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	sysPciDevs      = "/sys/bus/pci/devices"   // All PCI devices on the host
	sysPciSlots     = "/sys/bus/pci/slots"     // PCI slots on the host
	pciAddrLen      = 10                       // Length of PCI dev addr
	diskPathByDevID = "/dev/disk/by-id/wwn-0x" // Path for devices named by ID
	scsiHostPath    = "/sys/class/scsi_host/"  // Path for scsi hosts
	devWaitTimeout  = 1 * time.Second
	bdevPath        = "/sys/block/"
	deleteFile      = "/device/delete"
	watchPath       = "/dev/disk/by-id"
)

// FstypeDefault contains the default FS when not specified by the user
const FstypeDefault = "ext4"

// BinSearchPath contains search paths for host binaries
var BinSearchPath = []string{"/bin", "/sbin", "/usr/bin", "/usr/sbin"}

// VolumeDevSpec - volume spec returned from the server on an attach
type VolumeDevSpec struct {
	Unit                    string
	ControllerPciSlotNumber string
}

// DevAttachWaitPrep create the watcher.
func DevAttachWaitPrep(name string, devPath string) (*inotify.Watcher, bool) {
	watcher, errWatcher := inotify.NewWatcher()

	if errWatcher != nil {
		log.WithFields(log.Fields{"Name": name}).Error("Failed to create watcher, skip inotify ")
		return nil, true
	}

	err := watcher.Watch(devPath)

	if err != nil {
		log.WithFields(log.Fields{"Name": name}).Error("Failed to watch /dev, skip inotify ")
		return nil, true
	}

	return watcher, false
}

// DevAttachWait waits for attach operation to be completed
func DevAttachWait(watcher *inotify.Watcher, name string, device string) {
loop:
	for {
		select {
		case ev := <-watcher.Event:
			log.Debug("event: ", ev)
			if ev.Name == device {
				// Log when the device is discovered
				log.WithFields(
					log.Fields{"device": device, "event": ev},
				).Info("Scan complete ")
				break loop
			}
		case err := <-watcher.Error:
			log.WithFields(
				log.Fields{"device": device, "error": err},
			).Error("Hit error during watch ")
			break loop
		case <-time.After(devWaitTimeout):
			log.WithFields(
				log.Fields{"timeout": devWaitTimeout, "device": device},
			).Warning("Exceeded timeout while waiting for device attach to complete")
			break loop
		}
	}

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
	var err error
	var out []byte

	// Workaround older versions of e2fsprogs, issue 629.
	// If mkfscmd is of an ext* filesystem use -F flag
	// to avoid having mkfs command to expect user confirmation.
	if strings.Split(mkfscmd, ".")[1][0:3] == "ext" {
		out, err = exec.Command(mkfscmd, "-F", "-L", label, device).CombinedOutput()
	} else {
		out, err = exec.Command(mkfscmd, "-L", label, device).CombinedOutput()
	}
	if err != nil {
		return fmt.Errorf("Failed to create filesystem on %s: %s. Output = %s",
			device, err, out)
	}
	return nil
}

// MkfsLookup finds existent filesystem tools
func MkfsLookup() map[string]string {
	supportedFs := make(map[string]string)

	for _, sp := range BinSearchPath {
		mkftools, _ := filepath.Glob(sp + "/mkfs.*")
		for _, mkfs := range mkftools {
			supportedFs[strings.Split(mkfs, ".")[1]] = mkfs
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

// MountWithID - mount device with ID
func MountWithID(mountpoint string, fstype string, id string, isReadOnly bool) error {
	log.WithFields(log.Fields{
		"device ID":  id,
		"fstype":     fstype,
		"mountpoint": mountpoint,
	}).Debug("Calling syscall.Mount() ")

	// Scan so we may have the device before attempting a mount
	// Loop over all hosts and scan each one
	device, err := GetDevicePathByID(id)
	if err != nil {
		return fmt.Errorf("Invalid device path %s for %s: %s",
			device, mountpoint, err)
	}

	flags := 0
	if isReadOnly {
		flags = syscall.MS_RDONLY
	}
	err = syscall.Mount(device, mountpoint, fstype, uintptr(flags), "")
	if err != nil {
		return fmt.Errorf("Failed to mount device %s at %s fstype %s: %s",
			device, mountpoint, fstype, err)
	}
	return nil
}

// Unmount a device from the given mount point.
func Unmount(mountPoint string) error {
	err := syscall.Unmount(mountPoint, 0)
	if err != nil {
		return fmt.Errorf("Unmount device at %s failed: %s",
			mountPoint, err)
	}
	return nil
}

func makeDevicePathWithID(id string) string {
	return diskPathByDevID + strings.Join(strings.Split(id, "-"), "")
}

// GetDevicePathByID - return full path for device with given ID
func GetDevicePathByID(id string) (string, error) {
	hosts, err := ioutil.ReadDir(scsiHostPath)
	if err != nil {
		return "", err
	}
	time.Sleep(10)
	for _, host := range hosts {
		//Scan so we may have the device before attempting a mount
		scanHost := scsiHostPath + host.Name() + "/scan"
		bytes := []byte("- - -")
		log.WithFields(log.Fields{"disk id": id, "scan cmd": scanHost}).Info("Rescanning ... ")
		err = ioutil.WriteFile(scanHost, bytes, 0644)
		if err != nil {
			return "", err
		}
	}

	watcher, skipInotify := DevAttachWaitPrep(id, watchPath)

	device := makeDevicePathWithID(id)

	if skipInotify {
		time.Sleep(10)
	} else {
		// Wait for the attach to complete, may timeout
		// in which case we continue creating the file system.
		DevAttachWait(watcher, id, device)
	}
	_, err = os.Stat(device)
	if err != nil {
		return "", err
	}
	return device, nil
}

// DeleteDevicePathWithID - delete device with given ID
func DeleteDevicePathWithID(id string) error {
	// Delete the device node
	device := makeDevicePathWithID(id)
	dev, err := os.Readlink(device)
	if err != nil {
		return fmt.Errorf("Failed to read sym link for %s: %s",
			device, err)
	}
	links := strings.Split(dev, "/")
	node := bdevPath + links[len(links)-1] + deleteFile
	bytes := []byte("1")

	log.Debugf("Deleteing device node - id: %s, node: %s", id, node)
	err = ioutil.WriteFile(node, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
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
