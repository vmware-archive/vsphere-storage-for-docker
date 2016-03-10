package vmdkops_test

// Test commands with mocked ESX server and guest fs code
// Does not communicate over VMCI

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware/docker-vmdk-plugin/vmdkops"
	"testing"
)

func TestCommands(t *testing.T) {
	ops := vmdkops.VmdkOps{Cmd: vmdkops.MockVmdkCmd{}}
	name := "myVolume"
	opts := map[string]string{"size": "2gb", "format": "none"}
	assert.Nil(t, ops.Create(name, opts))

	opts = map[string]string{}
	assert.Nil(t, ops.Attach(name, opts))
	assert.Nil(t, ops.Detach(name, opts))
	assert.Nil(t, ops.Remove(name, opts))
	assert.Nil(t,
		ops.Create("otherVolume",
			map[string]string{"size": "1gb", "format": "ext4"}))
}
