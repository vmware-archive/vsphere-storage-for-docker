package config_test

// Test Loading JSON config files

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware/docker-vmdk-plugin/config"
	"testing"
)

func TestLoad(t *testing.T) {
	conf, err := config.Load("../default-config.json")
	assert.Nil(t, err)
	assert.Equal(t, conf.MaxLogSizeMb, 100)
	assert.Equal(t, conf.MaxLogAgeDays, 28)
	assert.Equal(t, conf.LogPath, "/var/log/docker-vmdk-plugin.log")
}
