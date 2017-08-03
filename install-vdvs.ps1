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
#
# The -Force parameter suppresses any confirmation prompt and performs an
# affirmative action. For example, supplying -Force during installation will
# uninstall any previous installation of the plugin, without confirmation.
# Supplying -Force during uninstallation suppresses the confirmation prompt
# and uninstalls the plugin.
# ============================================================================

<#
.SYNOPSIS
    Installation script for VMware vSphere Docker Volume Plugin.
.DESCRIPTION
    This script helps to download, install, uninstall, and re-install VMware vSphere Docker Volume Plugin on your system.
.EXAMPLE
    ./install-vdvs.ps1 https://bintray.com/vmware/vDVS/download_file?file_path=docker-volume-vsphere.zip
.EXAMPLE
    ./install-vdvs.ps1 -uninstall
.LINK
    https://vmware.github.io/docker-volume-vsphere/
#>

# Command line parameters
param (
    [string] $uri,
    [switch] $uninstall,
    [switch] $Force
)

# Define the constants
$svcName = "vdvs"
$svcDisplayName = "vSphere Docker Volume Service"
$svcDescription = "Enables user to run stateful containerized applications on top of VMware vSphere."
$installPath = "C:\Program Files\VMware\vmdkops"
$exePathName = $installPath + "\vdvs.exe"
$zipFileName = "docker-volume-vsphere.zip"

function Uninstall-Service([System.ServiceProcess.ServiceController]$svc) {
    if ($svc.Status -eq "Running") {
        echo "Stopping Windows service $svcName..."
        Stop-Service -Name $svcName
    }

    echo "Deleting Windows service $svcName..."
    sc.exe delete $svcName

    echo "Deleting $installPath..."
    Remove-Item -Path $installPath -Recurse -Force

    echo "Windows service $svcName uninstalled successfully!"
}

# Check if vDVS plugin is already installed
$svc = Get-Service | Where-Object {$_.Name -eq $svcName}

# Handle uninstallation
if ($uninstall) {
    if ($svc) {
        if ($Force -Or ($result = Read-Host "Do you really want to uninstall $svcName [Y/N]?") -eq "Y") {
            Uninstall-Service($svc)
        }
    } else {
        echo "Windows service $svcName is not installed."
    }
    return
}

# Check URI parameter for installation process
if (! $uri) {
     echo "Usage: install-vdvs.ps1 [[-uri] <String>] [-uninstall] [-Force]"
     return
}

# Handle reinstallation
if ($svc) {
    echo "Windows service $svcName is already installed."
    if ($Force -Or ($result = Read-Host "Do you want to reinstall $svcName [Y/N]?") -eq "Y") {
        Uninstall-Service($svc)
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

# Remove the archive after successful expanding
echo "Deleting $zipFileName..."
Remove-Item -Path $zipFileName -Force

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
