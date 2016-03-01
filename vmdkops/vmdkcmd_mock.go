// +build linux,mock_esx

package vmdkops

type volume struct {
	name        string
	attached_to string
}

type esxModel struct {
	volumes []volume
}

// Return JSON responses to each command or an error
func VmdkCmd(cmd string, name string, opts map[string]string) ([]byte, error) {
	switch cmd {
	case "create":
		return []byte("null"), nil
	}
	return nil, nil
}
