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

// A home to hold test constants which will be used to vsan related tests.

package admincli

const (
	// PolicyName is the name of vsan policy which will be used in test
	PolicyName = "some-policy"

	// PolicyContent is the content of vsan policy which will be used in test
	PolicyContent = "'((\"proportionalCapacity\" i50)''(\"hostFailuresToTolerate\" i0))'"

	// CreatePolicy Create a policy
	CreatePolicy = vmdkopsAdmin + " policy create --name="

	// ListPolicy referring to list all existing policies
	ListPolicy = vmdkopsAdmin + "policy ls "

	// RemovePolicy referring to remove a policy
	RemovePolicy = vmdkopsAdmin + "policy rm --name="

	// VsanPolicyFlag is the flag that will be passed in "vmdkops_admin policy XXX" command
	VsanPolicyFlag = "vsan-policy-name"
)
