package vmdkops_test

// testing command build and transfer. Uses dummy communication backend -
// does NOT communicate over vmci

// currently manual - go test -v and observe the results
// TODO: have dummy backend return received as is, parse it here as json
// and compare
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
