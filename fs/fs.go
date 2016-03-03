// +build linux

package fs

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

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

func Mount(mountpoint string, name string) error {
	// TODO: evenntually drop the exec.command(mount) and replace with
	// docker.mount.mount (?)
	// OR
	//	if err := syscall.Mount(device, target, mType, flag, data); err != nil {
	//		return err
	//	}
	out, err := exec.Command("blkid", []string{"-L", name}...).Output()
	if err != nil {
		log.Printf("blkid err: %T (%s)", err, string(out))
		return fmt.Errorf("blkid err: %T", err)
	} else {
		device := strings.TrimRight(string(out), " \n")
		log.Printf("Mounting %s for label %s", device, mountpoint)
		out, err = exec.Command("mount", []string{device, mountpoint}...).Output()
		if err != nil {
			return fmt.Errorf("Failed to mount device %s at %s", device, mountpoint)
		}
	}
	return nil
}

// TODO : use docker.mount or syscall.mount, and goroutine
// syscall.Unmount(d.volumes(r.Name).device, flag)
func Unmount(mountpoint string) error {
	_, err := exec.Command("umount", mountpoint).Output()
	return err
}
