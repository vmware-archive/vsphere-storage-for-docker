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

// Windows specific constants related to docker cli.

// +build winutil

package dockercli

const (
	// ContainerImage refers to the microsoft/nanoserver container image.
	ContainerImage = " microsoft/nanoserver "

	// TestContainer represents a test microsoft/nanoserver container that keeps running.
	TestContainer = ContainerImage + " ping -t localhost "

	// ContainerMountPoint is the mount point where a volume is mounted inside a container.
	ContainerMountPoint = `C:\vol`
)
