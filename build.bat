:: Copyright 2017 VMware, Inc. All Rights Reserved.
::
:: Licensed under the Apache License, Version 2.0 (the "License");
:: you may not use this file except in compliance with the License.
:: You may obtain a copy of the License at
::
::    http:::www.apache.org/licenses/LICENSE-2.0
::
:: Unless required by applicable law or agreed to in writing, software
:: distributed under the License is distributed on an "AS IS" BASIS,
:: WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
:: See the License for the specific language governing permissions and
:: limitations under the License.

:: Script for writing vsphere.json into the docker plugin config directory and
:: building vmci_client.dll and vdvs.exe.

:: Build Instructions
:: ==================
:: The plugin can be built on Windows 2016 Server and Windows 10 Pro/Enterprise/Education (anniversary release) VMs.
::
:: Dependencies
:: ------------
::   * Golang
::   * MSVC Build Tools 2017
::   * mingw-w64 for gcc, which is required for CGO in Go.
::
:: Building
:: --------
::   Start cmd and cd to the docker-volume-vsphere directory.
::   Execute build.bat.

:: ---- Setup ----
@echo off
set projectRoot=%cd%
set winBuildDir=%projectRoot%\build\windows
set vmciDir=%projectRoot%\esx_service\vmci
set pluginDir=%projectRoot%\client_plugin\vmdk_plugin
set vmdkopsDir=%projectRoot%\client_plugin\drivers\vmdk\vmdkops
set vsphereJsonPath=C:\ProgramData\docker\plugins\vsphere.json
set msvcBatPath=C:\Program Files (x86)\Microsoft Visual C++ Build Tools\vcbuildtools.bat

:: ---- Prerequisites ----
:: Verify that Go exists on PATH.
where /q go
if errorlevel 1 (
	echo ERROR: Couldn't find Go. Ensure that it is installed and is on your PATH.
	echo Download URL: https://golang.org/dl/
	exit /B
)
echo Found Go.

:: Verify that MSVC Build Tools are accessible.
if not exist "%msvcBatPath%" (
	echo ERROR: Couldn't find Microsoft Visual C++ Build Tools.
	echo Download URL: http://landinghub.visualstudio.com/visual-cpp-build-tools
	exit /B
)
echo Found Microsoft Visual C++ Build Tools.

:: ---- Cleanup ----
echo Cleaning up.
del %winBuildDir%\vmci_client.dll
del %winBuildDir%\vdvs.exe
echo Clean up complete.

:: ---- Build ----
echo Building vDVS...

:: Compile vmci_client.dll.
echo Entering the MSVC Build Tools environment.
call "%msvcBatPath%" amd64

echo Entering the vmci source directory.
cd %vmciDir%

echo Compiling vmci_client.dll.
cl /D_USRDLL /D_WINDLL vmci_client.c /link /defaultlib:ws2_32.lib /DLL /OUT:vmci_client.dll /DEF:vmci_client.def
del vmci_client.exp vmci_client.lib vmci_client.obj
echo Compiled vmci_client.dll successfully.

:: Move vmci_client.dll to the vmdkops directory.
echo Moving vmci_client.dll to the vmdkops directory.
move /y vmci_client.dll %vmdkopsDir%
echo Successfully moved vmci_client.dll to the vmdkops directory.

:: Build vdvs.exe.
echo Entering the plugin directory.
cd %pluginDir%

echo Building vdvs.exe.
go build -v -o vdvs.exe main.go
echo Successfully built vdvs.exe.

:: Write vsphere.json to the docker plugin config directory.
echo Writing vsphere.json to the docker plugin config directory.
del %vsphereJsonPath%
(
echo {
echo   "Name": "vsphere",
echo   "Addr": "npipe:////./pipe/vsphere-dvs"
echo }
) > %vsphereJsonPath%
echo Successfully wrote vsphere.json to the docker plugin config directory.

:: Move binaries to build directory.
echo Moving binaries to the build directory.
if not exist %winBuildDir% mkdir %winBuildDir%
move /y %vmdkopsDir%\vmci_client.dll %winBuildDir%
move /y vdvs.exe %winBuildDir%
echo Successfully moved binaries to the build directory.

cd %projectRoot%
echo vDVS build complete.
echo Binaries are available under %winBuildDir%.
