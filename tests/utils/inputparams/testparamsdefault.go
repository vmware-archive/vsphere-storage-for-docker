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

// Support for basic utility/helper methods used in tests on non-windows platforms.

// +build !winutil

package inputparams

import (
	"log"
	"os"
)

// volNameCharset is the valid set of characters for volume name generation.
const volNameCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// getDockerHosts returns a slice of Docker host VM IP addresses.
func getDockerHosts() []string {
	dockerHosts := []string{
		os.Getenv("VM1"),
		os.Getenv("VM2"),
	}
	if dockerHosts[0] == "" || dockerHosts[1] == "" {
		log.Fatal("Two linux docker hosts are needed to run tests.")
	}
	return dockerHosts
}

// normalizeVolumeName returns the name as-is.
func normalizeVolumeName(name string) string {
	return name
}
