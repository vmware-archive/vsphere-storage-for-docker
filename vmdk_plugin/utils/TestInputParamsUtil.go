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

package TestInputParamsUtil

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
	flag.StringVar(&volumeName, "v", "DockerTestVol", "Volume name to use in tests")
	flag.StringVar(&endPoint1, "H1", "unix:///var/run/docker.sock", "Endpoint (Host1) to connect to")
	flag.StringVar(&endPoint2, "H2", "unix:///var/run/docker.sock", "Endpoint (Host2) to connect to")
	flag.Parse()
}

func GetVolumeName() string {
	return volumeName
}

func GetVolumeNameWithTimeStamp(volName string) string {
	return volumeName + "_" + strconv.FormatInt(time.Now().Unix(), 10)
}

func GetEndPoint1() string {
	return endPoint1
}

func GetEndPoint2() string {
	return endPoint2
}
