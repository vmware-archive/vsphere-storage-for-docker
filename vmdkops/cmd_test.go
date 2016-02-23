package vmdkops_test

// testing command build and transfer. Uses dummy communication backend -
// does NOT communicate over vmci

// currently manual - go test -v and observe the results
// TODO: have dummy backend return received as is, parse it here as json
// and compare
import (
	"github.com/vmware/docker-vmdk-plugin/vmdkops"
	"testing"
)

func TestCommands(t *testing.T) {

	vmdkops.TestSetDummyBackend()

	name := "myVolume"
	opts := map[string]string{"size": "2gb", "format": "none"}

	vmdkops.Create(name, opts)
	vmdkops.Remove(name, opts)
	vmdkops.Attach(name, opts)
	vmdkops.Detach(name, opts)

	err := vmdkops.Create("otherVolume", map[string]string{"size": "1gb", "format": "ext4"})
	if err != "" {
		t.Error(err)
	}
}
