// Copyright 2017 VMware, Inc. All Rights Reserved.
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

// This test suite tries to add, remove and replace vm to the _DEFAULT vmgroup
// Expected behavior is that add/rm/replace vm for _DEFAULT vmgroup should fail

// +build runonce

package e2e

import (
	"log"
	"os"
	"strings"

	con "github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/govc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	. "gopkg.in/check.v1"
)

const (
	ErrorMsg = "This feature is not supported for vmgroup _DEFAULT."
)

type DefaultVMGroupTestSuite struct {
	esxIP    string
	hostName string
}

func (s *DefaultVMGroupTestSuite) SetUpSuite(c *C) {
	s.hostName = govc.RetrieveVMNameFromIP(os.Getenv("VM2"))
	s.esxIP = inputparams.GetEsxIP()
	out, err := admincli.ConfigInit(s.esxIP)
	c.Assert(err, IsNil, Commentf(out))
}

func (s *DefaultVMGroupTestSuite) TearDownSuite(c *C) {
	out, err := admincli.ConfigRemove(s.esxIP)
	c.Assert(err, IsNil, Commentf(out))
}

var _ = Suite(&DefaultVMGroupTestSuite{})

// Test step:
// Add a vm to the _DEFAULT Vmgroup
func (s *DefaultVMGroupTestSuite) TestAddVMToDefaultTenant(c *C) {
	log.Printf("START: defaultvmgroup.TestAddVMToDefaultTenant")

	out, err := admincli.AddVMToVMgroup(s.esxIP, con.DefaultVMgroup, s.hostName)
	c.Assert(err, IsNil, Commentf(out))
	c.Assert(strings.TrimRight(string(out), "\n"), Equals, ErrorMsg, Commentf("Unexpected Behavior: "+
		"We are able to add vm to the _DEFAULT tenant which is NOT allowed as per current spec."))

	log.Printf("END: defaultvmgroup.TestAddVMToDefaultTenant")
}

// Test step:
// Remove a vm from the _DEFAULT Vmgroup
func (s *DefaultVMGroupTestSuite) TestRemoveDefaultTenantVMs(c *C) {
	log.Printf("START: defaultvmgroup.TestRemoveDefaultTenantVMs")

	out, err := admincli.RemoveVMFromVMgroup(s.esxIP, con.DefaultVMgroup, s.hostName)
	c.Assert(err, IsNil, Commentf(out))
	c.Assert(strings.TrimRight(string(out), "\n"), Equals, ErrorMsg, Commentf("Unexpected Behavior: "+
		"We are able to remove vm from the _DEFAULT tenant which is NOT allowed as per current spec."))

	log.Printf("END: defaultvmgroup.TestRemoveDefaultTenantVMs")
}

// Test step:
//  Replace a vm from the _DEFAULT Vmgroup
func (s *DefaultVMGroupTestSuite) TestReplaceDefaultTenantVMs(c *C) {
	log.Printf("START: defaultvmgroup.TestReplaceDefaultTenantVMs")

	out, err := admincli.ReplaceVMFromVMgroup(s.esxIP, con.DefaultVMgroup, s.hostName)
	c.Assert(err, IsNil, Commentf(out))
	c.Assert(strings.TrimRight(string(out), "\n"), Equals, ErrorMsg, Commentf("Unexpected Behavior: "+
		"We are able to replace vm from the _DEFAULT tenant which is NOT allowed as per current spec."))

	log.Printf("END: defaultvmgroup.TestReplaceDefaultTenantVMs")
}
