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
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	ps "github.com/vmware/vsphere-storage-for-docker/client_plugin/utils/powershell"
)

const (
	// FstypeDefault specifies the default FS to be used when not specified by the user.
	FstypeDefault = ntfs

	// maxDiskAttachWaitSec is the max time to wait for a disk to be attached.
	maxDiskAttachWaitSec = 30 * time.Second

	ntfs         = "ntfs"
	diskNotFound = "DiskNotFound"

	// Using a PowerShell script here due to lack of a functional Go WMI library.
	// PowerShell script to identify the disk number given a scsi controller
	// number and a unit number. The script returns an integer if the disk was
	// found, else it returns DiskNotFound.
	scsiAddrToDiskNumScript = `
		$found = $false;
		Get-PnpDevice -FriendlyName 'VMware Virtual disk SCSI Disk Device' |
		Select-Object -ExpandProperty InstanceId |
		ForEach-Object {
			$instanceId = $_;
			$uiNum = $(
				Get-PnpDeviceProperty -InstanceId $instanceId -KeyName 'DEVPKEY_Device_UINumber' |
				Select -Expand Data -ErrorAction 'SilentlyContinue';
			);
			$addr = $(
				Get-PnpDeviceProperty -InstanceId $instanceId -KeyName 'DEVPKEY_Device_Address' |
				Select -Expand Data -ErrorAction 'SilentlyContinue';
			);
			If ($uiNum -eq '%s' -And $addr -eq '%s') {
				$found = $true;
				Write-Host "";
				Get-WmiObject Win32_DiskDrive |
				Where-Object { $_.PNPDeviceId -eq $instanceId } |
				Select-Object -ExpandProperty Index;
				Return;
			};
		};
		If (-Not $found) {
			Write-Host "";
			Write-Host "DiskNotFound";
		};
	`

	// PowerShell script to format a disk with a filesystem.
	// Note: -Confirm:$false suppresses the confirmation prompt.
	formatDiskScript = `
		Set-Disk -Number %s -IsOffline $false;
		Set-Disk -Number %s -IsReadOnly $false;
		Initialize-Disk -Number %s -PartitionStyle MBR -PassThru |
		New-Partition -UseMaximumSize |
		Format-Volume -FileSystem %s -NewFileSystemLabel "%s" -Confirm:$false;
	`

	// mountListScript is a PowerShell script that lists disk numbers and their access paths.
	// Sample output:
	//   0 \\?\Volume{1cca2a47-0000-0000-0000-100000000000}\
	//   1 C:\Users\Administrator\AppData\Local\vsphere-storage-for-docker\mounts\volName\ \\?\Volume{145c5662-0000-0000-0000-100000000000}\
	mountListScript = `
		Get-Partition |
		ForEach-Object { Write-Host $_.DiskNumber, $_.AccessPaths };
	`

	// mountDiskScript is a PowerShell script that sets the disk's mode to RO if
	// needed, and then mounts its first partition at the given mountpoint.
	mountDiskScript = `
		Set-Disk -Number %s -IsReadOnly $%t;
		Add-PartitionAccessPath -DiskNumber %s -PartitionNumber 1 -AccessPath "%s";
	`

	// unmountDiskScript is a PowerShell script that identifies the disk mounted
	// at the given mountpoint, and then unmounts it.
	unmountDiskScript = `
		$found = $false;
		Get-Partition |
		ForEach-Object {
			If ($_.AccessPaths -contains "%s") {
				$found = $true;
				$diskNum = $_.DiskNumber;
				Remove-PartitionAccessPath -DiskNumber $diskNum -PartitionNumber 1 -AccessPath "%s";
				Write-Host $diskNum;
				Return;
			};
		};
		If (-Not $found) {
			Write-Host "DiskNotFound";
		};
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

// Mkfs creates a filesystem at the specified volDev.
func Mkfs(fstype string, label string, volDev *VolumeDevSpec) error {
	diskNum, err := getDiskNum(volDev)
	if err != nil {
		log.WithFields(log.Fields{"fstype": fstype, "label": label, "volDev": *volDev,
			"err": err}).Error("Failed to locate disk ")
		return err
	}

	script := fmt.Sprintf(formatDiskScript, diskNum, diskNum, diskNum, fstype, label)
	stdout, stderr, err := ps.Exec(script)
	if err != nil {
		log.WithFields(log.Fields{"fstype": fstype, "label": label, "volDev": *volDev, "diskNum": diskNum,
			"err": err, "stdout": stdout, "stderr": stderr}).Error("Format disk script failed ")
		return err
	} else {
		log.WithFields(log.Fields{"fstype": fstype, "label": label, "volDev": *volDev, "diskNum": diskNum,
			"stdout": stdout}).Info("Format disk script executed successfully ")
		return nil
	}
}

// Mount mounts the filesystem on the volDev at the given mountpoint.
func Mount(mountpoint string, fstype string, volDev *VolumeDevSpec, isReadOnly bool) error {
	diskNum, err := getDiskNum(volDev)
	if err != nil {
		log.WithFields(log.Fields{"mountpoint": mountpoint, "fstype": fstype,
			"volDev": *volDev, "isReadOnly": isReadOnly, "err": err}).Error("Failed to locate disk ")
		return err
	}

	script := fmt.Sprintf(mountDiskScript, diskNum, isReadOnly, diskNum, mountpoint)
	stdout, stderr, err := ps.Exec(script)
	if err != nil {
		log.WithFields(log.Fields{"mountpoint": mountpoint, "fstype": fstype,
			"volDev": *volDev, "isReadOnly": isReadOnly, "diskNum": diskNum,
			"err": err, "stdout": stdout, "stderr": stderr}).Error("Failed to mount disk ")
		return err
	}

	log.WithFields(log.Fields{"mountpoint": mountpoint, "fstype": fstype, "volDev": *volDev,
		"isReadOnly": isReadOnly, "diskNum": diskNum, "stdout": stdout}).Info("Disk successfully mounted ")
	return nil
}

// Unmount unmounts a disk from the given mount point.
func Unmount(mountpoint string) error {
	// PowerShell returns access paths with a trailing slash.
	if !strings.HasSuffix(mountpoint, `\`) {
		mountpoint += `\`
	}

	script := fmt.Sprintf(unmountDiskScript, mountpoint, mountpoint)
	stdout, stderr, err := ps.Exec(script)
	if err != nil {
		log.WithFields(log.Fields{"mountpoint": mountpoint, "err": err,
			"stdout": stdout, "stderr": stderr}).Error("Failed to unmount disk ")
		return err
	} else if tailSegment(stdout, lf, 2) == diskNotFound {
		msg := fmt.Sprintf("Failed to unmount disk from '%s'", mountpoint)
		log.WithField("stdout", stdout).Error(msg)
		return errors.New(msg)
	}
	log.WithFields(log.Fields{"mountpoint": mountpoint,
		"stdout": stdout}).Info("Disk unmounted ")
	return nil
}

// getDiskNum returns the disk number corresponding to volDev, or an error on
// failing to identify the disk.
func getDiskNum(volDev *VolumeDevSpec) (string, error) {
	script := fmt.Sprintf(scsiAddrToDiskNumScript, volDev.ControllerPciSlotNumber, volDev.Unit)
	stdout, stderr, err := ps.Exec(script)
	if err != nil {
		log.WithFields(log.Fields{"volDev": *volDev, "err": err, "stdout": stdout,
			"stderr": stderr}).Error("Failed to execute the disk mapping script ")
		return "", err
	}
	log.WithFields(log.Fields{"volDev": *volDev,
		"stdout": stdout}).Info("Disk mapping script executed ")

	diskNum := strings.Replace(tailSegment(stdout, lf, 2), cr, "", -1)
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
func tailSegment(out string, delim string, n int) string {
	segments := strings.Split(out, delim)
	segment := segments[len(segments)-n]
	return segment
}

// GetMountInfo returns a map of mounted volumes and disk numbers.
func GetMountInfo(mountRoot string) (map[string]string, error) {
	volumeMountMap := make(map[string]string)

	stdout, stderr, err := ps.Exec(mountListScript)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "stdout": stdout,
			"stderr": stderr}).Error("Couldn't execute script to list mounts")
		return volumeMountMap, err
	}
	log.WithFields(log.Fields{"stdout": stdout}).Info("List mounts script executed")

	for _, line := range strings.Split(stdout, lf) {
		fields := strings.SplitN(line, " ", 2)
		if len(fields) < 2 {
			continue // skip empty line and lines too short to have our mount
		}
		for _, path := range strings.SplitN(fields[1], `\ `, -1) {
			path = filepath.Clean(path) // remove trailing slash
			if filepath.Dir(path) == mountRoot {
				volumeMountMap[filepath.Base(path)] = fields[0]
				break
			}
		}
	}

	log.WithFields(log.Fields{"map": volumeMountMap}).Info("Successfully retrieved mounts")
	return volumeMountMap, nil
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
