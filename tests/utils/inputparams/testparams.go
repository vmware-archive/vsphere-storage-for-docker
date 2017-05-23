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

package inputparams

// This file holds basic utility/helper methods for object creation used
// at test methods

import (
	"flag"
	"strconv"
	"time"
)

var (
	endPoint1  string
	endPoint2  string
	volumeName string
)

func init() {

	flag.StringVar(&endPoint1, "H1", "unix:///var/run/docker.sock", "Endpoint (Host1) to connect to")
	flag.StringVar(&endPoint2, "H2", "unix:///var/run/docker.sock", "Endpoint (Host2) to connect to")
	flag.StringVar(&volumeName, "v", "TestVol", "Volume name to use in tests")
	flag.Parse()
}

// GetVolumeName returns the default volume name (set as environment variable or supplied through cli)
func GetVolumeName() string {
	return volumeName
}

// GetVolumeNameWithTimeStamp prepares unique volume name by appending current time-stamp value
func GetVolumeNameWithTimeStamp(volumeName string) string {
	return volumeName + "_volume_" + strconv.FormatInt(time.Now().Unix(), 10)
}

// GetContainerNameWithTimeStamp prepares unique container name by appending current time-stamp value
func GetContainerNameWithTimeStamp(containerName string) string {
	return containerName + "_container_" + strconv.FormatInt(time.Now().Unix(), 10)
}

// GetEndPoint1 returns first VM endpoint supplied through CLI
func GetEndPoint1() string {
	return endPoint1
}

// GetEndPoint2 returns second VM endpoint supplied through CLI
func GetEndPoint2() string {
	return endPoint2
}
