package vmdkops_test

// Test commands with mocked ESX server and guest fs code
// Does not communicate over VMCI

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware/docker-vmdk-plugin/vmdkops"
	"testing"
)

func TestCommands(t *testing.T) {
	name := "myVolume"
	opts := map[string]string{"size": "2gb", "format": "none"}
	assert.Nil(t, vmdkops.Create(name, opts))

	opts = map[string]string{}
	assert.Nil(t, vmdkops.Remove(name, opts))
	assert.Nil(t, vmdkops.Attach(name, opts))
	assert.Nil(t, vmdkops.Detach(name, opts))
	assert.Nil(t,
		vmdkops.Create("otherVolume",
			map[string]string{"size": "1gb", "format": "ext4"}))
}
