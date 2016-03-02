// +build linux,mock_esx

package vmdkops

import (
	"encoding/json"
	"fmt"
	"github.com/vmware/docker-vmdk-plugin/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	backing_root = "/tmp/docker-volumes" // Files for loopback device backing stored here
)

// Return JSON responses to each command or an error
func VmdkCmd(cmd string, name string, opts map[string]string) ([]byte, error) {
	// We store no in memory state, so just try to recreate backing_root every time
	err := fs.Mkdir(backing_root)
	if err != nil {
		return nil, err
	}
	log.Printf("Running Mock Cmd %s", cmd)
	switch cmd {
	case "create":
		err := create_block_device(name)
		return nil, err
	case "list":
		return list()
	case "attach":
		return nil, nil
	case "detach":
		return nil, nil
	case "remove":
		err := remove(name)
		return nil, err
	}
	return []byte("null"), nil
}

func list() ([]byte, error) {
	files, err := ioutil.ReadDir(backing_root)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s", backing_root)
	}
	volumes := make([]VolumeData, 0, len(files))
	for _, file := range files {
		volumes = append(volumes, VolumeData{Name: file.Name()})
	}
	return json.Marshal(volumes)
}

func remove(name string) error {
	backing := fmt.Sprintf("%s/%s", backing_root, name)
	out, err := exec.Command("blkid", []string{"-L", name}...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("blkid err: %T", err)
	}
	device := strings.TrimRight(string(out), " \n")
	fmt.Printf("Detaching loopback device %s", device)
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

func create_block_device(label string) error {
	backing := fmt.Sprintf("%s/%s", backing_root, label)
	exists, err := backing_exists(backing)
	if err != nil {
		return err
	}
	if exists {
		// We have already created this block device
		return nil
	}
	err = create_backing_file(backing)
	if err != nil {
		return err
	}
	loopback_count := get_max_loopback_count() + 1
	device := fmt.Sprintf("/dev/loop%d", loopback_count)
	err = create_device_node(device, loopback_count)
	if err != nil {
		return err
	}
	// Ignore output. This is to prevent spurious failures from old devices
	// that were removed, but not detached.
	exec.Command("losetup", "-d", device).CombinedOutput()
	err = setup_loopback_device(backing, device)
	if err != nil {
		return err
	}
	err = make_filesystem(device)
	if err != nil {
		return err
	}
	return label_device(label, device)
}

func backing_exists(backing string) (bool, error) {
	if _, err := os.Stat(backing); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func get_max_loopback_count() int {
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

func create_backing_file(backing string) error {
	of := fmt.Sprintf("of=%s", backing)
	out, err := exec.Command("dd", "if=/dev/zero", of, "bs=1M", "count=100").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to create backing file %s with error: %s. Output = %s",
			backing, err, out)
	}
	return nil
}

func create_device_node(device string, loopback_count int) error {
	count := fmt.Sprintf("%d", loopback_count)
	out, err := exec.Command("mknod", device, "b", "7", count).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to make device node %s with error: %s. Output = %s",
			device, err, out)
	}
	return nil
}

func setup_loopback_device(backing string, device string) error {
	out, err := exec.Command("losetup", device, backing).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to setup loopback device node %s for backing file %s with error: %s. Output = %s",
			device, backing, err, out)
	}
	return nil
}

func make_filesystem(device string) error {
	out, err := exec.Command("mkfs.ext3", device).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to create filesystem on %s with error: %s. Output = %s",
			device, err, out)
	}
	return nil
}

func label_device(label string, device string) error {
	out, err := exec.Command("e2label", device, label).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to apply label %s to device %s with error: %s. Output = %s",
			label, device, err, out)
	}
	return nil
}
