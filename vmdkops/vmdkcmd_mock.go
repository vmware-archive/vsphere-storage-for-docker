// +build linux,mock_esx

package vmdkops

import (
	"log"
)

type volume struct {
	name        string
	attached_to string
}

type esxModel struct {
	volumes []volume
}

// Return JSON responses to each command or an error
func VmdkCmd(cmd string, name string, opts map[string]string) ([]byte, error) {
	log.Printf("Running Mock Cmd %s", cmd)
	switch cmd {
	case "create":
		return []byte("null"), nil
	}
	return []byte("null"), nil
}
