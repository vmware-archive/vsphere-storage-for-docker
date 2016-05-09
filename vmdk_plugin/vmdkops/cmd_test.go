// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vmdkops_test

// Test commands with mocked ESX server and guest fs code
// Does not communicate over VMCI

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/vmdkops"
	"testing"
)

func TestCommands(t *testing.T) {
	ops := vmdkops.VmdkOps{Cmd: vmdkops.MockVmdkCmd{}}
	name := "myVolume"
	opts := map[string]string{"size": "2gb", "format": "none"}
	if assert.Nil(t, ops.Create(name, opts)) {

		opts = map[string]string{}
		_, err := ops.Attach(name, opts)
		assert.Nil(t, err)
		assert.Nil(t, ops.Detach(name, opts))
		assert.Nil(t, ops.Remove(name, opts))
	}
	if assert.Nil(t, ops.Create("otherVolume",
		map[string]string{"size": "1gb", "format": "ext4"})) {
		assert.Nil(t, ops.Remove("otherVolume", opts))
	}
}
