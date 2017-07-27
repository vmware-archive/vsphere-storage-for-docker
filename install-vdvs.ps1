# Copyright 2017 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Installation Instructions
# ============================================================================
# Run the script with the plugin url as the parameter in a PowerShell session:
# PS C:\Users\Administrator> .\install-vdvs.ps1 <vdvs-download-url>
# ============================================================================

# Command line parameters
param (
    [Parameter(Mandatory=$true)][string]$uri
)

# Define the constants
$svcName = "vdvs"
$svcDisplayName = "vSphere Docker Volume Service"
$svcDescription = "Enables user to run stateful containerized applications on top of VMware vSphere."
$installPath = "C:\Program Files\VMware\vmdkops"
$exePathName = $installPath + "\vdvs.exe"
$zipFileName = "docker-volume-vsphere.zip"

# Check if vdvs plugin is already installed
$svc = Get-Service | Where-Object {$_.Name -eq $svcName}
if ($svc) {
    echo "Windows service $svcName is already installed."
    $reinstall = Read-Host "Do you want to reinstall [Y/N]?"
    if ($reinstall -eq "Y") {
        if ($svc.Status -eq "Running") {
            echo "Stopping Windows service $svcName..."
            Stop-Service -Name $svcName
        }
        echo "Uninstalling Windows service $svcName..."
        sc.exe delete $svcName
    } else {
        return
    }
}

# Download the vdvs binary
echo "Downloading from $uri..."
Invoke-WebRequest $uri -OutFile $zipFileName
if (! $?) {
    echo "Failed to download from $uri."
    return
}

# Extract to the installation folder
echo "Extracting $zipFileName into $installPath..."
Expand-Archive -Path $zipFileName -DestinationPath $installPath -Force
if (! $?) {
    echo "Failed to extract $zipFileName into $installPath."
    return
}

# Install the vdvs plugin as a service
echo "Installing Windows service $svcName from $exePathName..."
New-Service -Name $svcName -BinaryPathName $exePathName -DisplayName $svcDisplayName -Description $svcDescription
if (! $?) {
    echo "Failed to install Windows service $svcName."
    return
}

# Start the vdvs service
echo "Starting Windows service $svcName..."
Start-Service -Name $svcName
if (! $?) {
    echo "Failed to start Windows service $svcName."
    return
}

# Show the running service
Get-Service -Name $svcName
echo "Windows service $svcName installed successfully!"
