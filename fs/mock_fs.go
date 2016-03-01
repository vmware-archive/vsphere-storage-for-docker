// +build linux,mock_esx

package fs

func Mkdir(path string) error {
	return nil
}

func Mount(mountpoint string, name string) error {
	return nil
}

func Unmount(mountpoint string) error {
	return nil
}
