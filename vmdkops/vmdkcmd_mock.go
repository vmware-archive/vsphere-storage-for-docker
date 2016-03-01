// +build linux,mock_esx

package vmdkops

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var loopback_count int = 1001

// Return JSON responses to each command or an error
func VmdkCmd(cmd string, name string, opts map[string]string) ([]byte, error) {
	log.Printf("Running Mock Cmd %s", cmd)
	switch cmd {
	case "create":
		err := create_block_device(name)
		if err != nil {
			return nil, err
		}
	}
	return []byte("null"), nil
}

func create_block_device(label string) error {
	backing := fmt.Sprintf("/tmp/%s", label)
	if backing_exists(backing) {
		// We have already created this block device
		return nil
	}
	loopback_count := get_max_loopback_count() + 1
	device := fmt.Sprintf("/dev/loop-%d", loopback_count)
	err := create_backing_file(backing)
	if err != nil {
		return err
	}
	err = create_device_node(device, loopback_count)
	if err != nil {
		return err
	}
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

func backing_exists(backing string) bool {
	if _, err := os.Stat(backing); os.IsNotExist(err) {
		return false
	}
	return true
}

func get_max_loopback_count() int {
	// always start at 1000
	count := 1000
	files, err := ioutil.ReadDir("/dev")
	if err != nil {
		panic("Failed to read /dev")
	}
	for _, file := range files {
		substrings := strings.Split(file.Name(), "-")
		if len(substrings) == 2 {
			if s, err := strconv.Atoi(substrings[1]); err == nil {
				if s > count {
					count = s
				}
			}
		}
	}
	return count
}

func create_backing_file(backing string) error {
	of := fmt.Sprintf("of=%s", backing)
	out, err := exec.Command("dd", "if=/dev/zero", of, "bs=1M", "count=100").Output()
	if err != nil {
		return fmt.Errorf("Failed to create backing file %s with error: %s. Output = %s",
			backing, err, out)
	}
	return nil
}

func create_device_node(device string, loopback_count int) error {
	count := fmt.Sprintf("%d", loopback_count)
	out, err := exec.Command("mknod", device, "b", "7", count).Output()
	if err != nil {
		return fmt.Errorf("Failed to make device node %s with error: %s. Output = %s",
			device, err, out)
	}
	return nil
}

func setup_loopback_device(backing string, device string) error {
	out, err := exec.Command("losetup", device, backing).Output()
	if err != nil {
		return fmt.Errorf("Failed to setup loopback device node %s for backing file %s with error: %s. Output = %s",
			device, backing, err, out)
	}
	return nil
}

func make_filesystem(device string) error {
	out, err := exec.Command("mkfs.ext3", device).Output()
	if err != nil {
		return fmt.Errorf("Failed to create filesystem on %s with error: %s. Output = %s",
			device, err, out)
	}
	return nil
}

func label_device(label string, device string) error {
	out, err := exec.Command("e2label", device, label).Output()
	if err != nil {
		return fmt.Errorf("Failed to apply label %s to device %s with error: %s. Output = %s",
			label, device, err, out)
	}
	return nil
}
