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

// This util provides various helper methods that can be used by testcases to
// fetch information related to a tenant.

package e2e

import (
	"log"
	"strconv"
	"strings"
	"time"
)

var ADMIN_CLI_TENANT = "/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py tenant "

// This method can be used to determine if a particular tenant exists
func CheckTenantExist(ipAddr string, tenantName string) bool {
	cmd := ADMIN_CLI_TENANT + "ls 2>/dev/null | cut -d ' ' -f 3"
	out, err := InvokeCommand(ipAddr, cmd)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if string(out) == "" {
		log.Fatal("Null value is returned by admin cli when looking for tenant - ", tenantName)
	}
	if !strings.Contains(string(out), tenantName) {
		return false
	}
	return true
}

// method to delete a particular tenant
func DeleteTenant(ipAddr string, tenantName string) {
	isTenantExist := CheckTenantExist(ipAddr, tenantName)
	if isTenantExist {
		cmd := ADMIN_CLI_TENANT + "rm --name=" + tenantName
		_, err := InvokeCommand(ipAddr, cmd)
		if err != nil {
			log.Fatalf("Failed to delete the tenant - [%s]. Error - %v ", tenantName, err)
		}
	}
}

// returns name with time stamp attached to it
func GetNameWithTimeStamp(name string) string {
	return name + "_" + strconv.FormatInt(time.Now().Unix(), 20)
}
