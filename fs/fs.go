// +build linux

// This is the filesystem interface for mounting volumes on the guest.

package fs

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"syscall"
)

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
func Mount(mountpoint string, device string, fs string) error {
	log.WithFields(log.Fields{
		"device":     device,
		"mountpoint": mountpoint,
	}).Debug("Calling syscall.Mount()")
	//_, err = exec.Command("mount", []string{device, mountpoint}...).Output()
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
