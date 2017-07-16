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

package vmdk

//
// Supporting functions for the VMware vSphere Docker Volume plugin on Windows.
//

import "strings"

// normalizeVolumeName returns the volume name in its lower-case form.
// Paths on Windows are case-insensitive, so Docker explicitly converts volume
// names to lower-case. The VMDK Ops service returns names like volname@Local2
// which need to be converted to lower-case before sending it in the API
// response, and we use this func for that conversion.
func normalizeVolumeName(name string) string {
	return strings.ToLower(name)
}
