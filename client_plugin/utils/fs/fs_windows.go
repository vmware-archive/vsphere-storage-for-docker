// Copyright 2017 VMware, Inc. All Rights Reserved.
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

// This is the filesystem interface for mounting volumes on a Windows guest.

package fs

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	// FstypeDefault specifies the default FS to be used when not specified by the user.
	FstypeDefault = ntfs

	maxDiskAttachWaitSec = 10 * time.Second // Max time to wait for a disk to be attached
	ntfs                 = "ntfs"
	powershell           = "powershell"
	diskNotFound         = "DiskNotFound"

	// Using a PowerShell script here due to lack of a functional Go WMI library.
	// PowerShell script to identify the disk number given a scsi controller
	// number and a unit number. The script returns an integer if the disk was
	// found, else it returns DiskNotFound.
	scsiAddrToDiskNumScript = `
		Get-PnpDevice -FriendlyName 'VMware Virtual disk SCSI Disk Device' |
		Select-Object -ExpandProperty InstanceId |
		ForEach-Object {
			$instanceId = $_
			$uiNum = (
				Get-PnpDeviceProperty -InstanceId $instanceId -KeyName 'DEVPKEY_Device_UINumber' |
				Select -expand Data -erroraction 'silentlycontinue'
			)
			$addr = (
				Get-PnpDeviceProperty -InstanceId $instanceId -KeyName 'DEVPKEY_Device_Address' |
				Select -expand Data -erroraction 'silentlycontinue'
			)
			If ($uiNum -eq '%s' -And $addr -eq '%s') {
				Write-Host ""
				Get-WmiObject Win32_DiskDrive |
				Where-Object { $_.PNPDeviceId -eq $instanceId } |
				Select-Object -ExpandProperty Index
				exit
			}
		}
		Write-Host ""
		Write-Host "DiskNotFound"
	`

	// PowerShell script to format a disk with a filesystem.
	// Note: -Confirm:$false suppresses the confirmation prompt.
	formatDiskScript = `
		Set-Disk -Number %s -IsOffline $false
		Set-Disk -Number %s -IsReadOnly $false
		Initialize-Disk -Number %s -PartitionStyle MBR -PassThru |
		New-Partition -UseMaximumSize |
		Format-Volume -FileSystem %s -NewFileSystemLabel "%s" -Confirm:$false
	`
)

// VerifyFSSupport checks whether the fstype filesystem is supported.
func VerifyFSSupport(fstype string) error {
	if fstype != ntfs {
		log.WithFields(log.Fields{"fstype": fstype}).Error("Unsupported fstype ")
		return fmt.Errorf("Not found mkfs for %s\nSupported filesystems: %s",
			fstype, ntfs)
	}
	return nil
}

// DevAttachWaitPrep initializes and returns a new DiskWatcher.
func DevAttachWaitPrep() (*DeviceWatcher, error) {
	watcher := NewDeviceWatcher()
	watcher.Init()
	return watcher, nil
}

// DevAttachWait waits until the specified disk is attached, or returns
// an error on watcher failure.
func DevAttachWait(watcher *DeviceWatcher, volDev *VolumeDevSpec) error {
	defer watcher.Terminate()
	for {
		log.WithFields(log.Fields{"volDev": *volDev}).Info("Waiting for a watcher event ")
		select {
		case event := <-watcher.Event:
			log.WithFields(log.Fields{"volDev": *volDev,
				"event": event}).Info("Watcher emitted an event ")
			if diskNum, err := getDiskNum(volDev); err != nil {
				log.WithFields(log.Fields{"volDev": *volDev,
					"err": err}).Warn("Couldn't map volDev to diskNum, continuing.. ")
			} else {
				log.WithFields(log.Fields{"volDev": *volDev,
					"diskNum": diskNum}).Info("Successfully mapped volDev to diskNum ")
				return nil
			}
			log.WithFields(log.Fields{"volDev": *volDev}).Warn("Couldn't locate disk, waiting.. ")

		case err := <-watcher.Error:
			log.WithFields(log.Fields{"volDev": *volDev,
				"err": err}).Error("Watcher returned an error ")
			return err

		case <-time.After(maxDiskAttachWaitSec):
			msg := "Disk mapping timed out "
			log.WithFields(log.Fields{"volDev": *volDev}).Error(msg)
			return errors.New(msg)
		}
	}
}

// DevAttachWaitFallback is a NOP.
func DevAttachWaitFallback() {
	// NOP since DevAttachWaitPrep never returns an error
}

// Mkdir creates a directory at the specified path.
func Mkdir(path string) error {
	// TODO: Implement for the volume mount workflow.
	return nil
}

// Mkfs creates a filesystem at the specified volDev.
func Mkfs(fstype string, label string, volDev *VolumeDevSpec) error {
	diskNum, err := getDiskNum(volDev)
	if err != nil {
		log.WithFields(log.Fields{"fstype": fstype, "label": label, "volDev": *volDev,
			"err": err}).Error("Failed to locate disk ")
		return err
	}

	script := fmt.Sprintf(formatDiskScript, diskNum, diskNum, diskNum, fstype, label)
	out, err := exec.Command(powershell, script).CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{"fstype": fstype, "label": label, "volDev": *volDev,
			"diskNum": diskNum, "err": err, "out": string(out)}).Error("Format disk script failed ")
		return err
	} else {
		log.WithFields(log.Fields{"fstype": fstype, "label": label, "volDev": *volDev,
			"diskNum": diskNum, "out": string(out)}).Info("Format disk script executed successfully ")
		return nil
	}
}

// Mount mounts the filesystem on the volDev at the given mountpoint.
func Mount(mountpoint string, fstype string, volDev *VolumeDevSpec, isReadOnly bool) error {
	// TODO: Implement for the volume mount workflow.
	return nil
}

// Unmount unmounts a disk from the given mount point.
func Unmount(mountpoint string) error {
	// TODO: Implement for the volume unmount workflow.
	return nil
}

// getDiskNum returns the disk number corresponding to volDev, or an error on
// failing to identify the disk.
func getDiskNum(volDev *VolumeDevSpec) (string, error) {
	script := fmt.Sprintf(scsiAddrToDiskNumScript, volDev.ControllerPciSlotNumber, volDev.Unit)
	out, err := exec.Command(powershell, script).CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{"volDev": *volDev,
			"err": err, "out": string(out)}).Error("Failed to execute the disk mapping script ")
		return "", err
	}
	log.WithFields(log.Fields{"volDev": *volDev,
		"out": string(out)}).Info("Disk mapping script executed ")

	diskNum := strings.Replace(tailSegment(out, lf, 2), cr, "", -1)
	if diskNum == diskNotFound {
		msg := fmt.Sprintf("Could not identify disk for controller = %s, unit = %s",
			volDev.ControllerPciSlotNumber, volDev.Unit)
		log.Error(msg)
		return "", errors.New(msg)
	}
	log.WithFields(log.Fields{"volDev": *volDev,
		"diskNum": diskNum}).Info("Successfully located disk ")
	return diskNum, nil
}

// tailSegment returns the nth to last segment of a string delimited with delim.
func tailSegment(b []byte, delim string, n int) string {
	segments := strings.Split(string(b), delim)
	segment := segments[len(segments)-n]
	return segment
}

// Functions needed by the photon driver, but not implemented for the Windows OS.

// DeleteDevicePathWithID returns an error.
func DeleteDevicePathWithID(id string) error {
	return errors.New("DeleteDevicePathWithID is not supported")
}

// GetDevicePathByID returns an error.
func GetDevicePathByID(id string) (string, error) {
	return "", errors.New("GetDevicePathByID is not supported")
}

// MkfsByDevicePath returns an error.
func MkfsByDevicePath(fstype string, label string, device string) error {
	return errors.New("MkfsByDevicePath is not supported")
}

// MountByDevicePath returns an error.
func MountByDevicePath(mountpoint string, fstype string, device string, isReadOnly bool) error {
	return errors.New("MountByDevicePath is not supported")
}

// MountWithID returns an error.
func MountWithID(mountpoint string, fstype string, id string, isReadOnly bool) error {
	return errors.New("MountWithID is not supported")
}
