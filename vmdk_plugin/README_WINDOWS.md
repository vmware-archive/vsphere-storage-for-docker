# Project Skylight

### Table of contents
1. Prerequisites
2. Compiling the `vmci_client.dll`
3. Running the plugin on Windows
4. Issues

### Prerequisites
1. Install the vDVS driver (VIB) on ESXi.
2. Install a Windows 2016 Server VM on ESXi.
3. Install [Docker EE](https://docs.docker.com/docker-ee-for-windows/install/) on your Windows 2016 Server VM.
4. Install [Git](https://git-scm.com/download/win) on your Windows 2016 Server VM.
5. Install [Go](https://golang.org/dl/)
6. Install [MSVC Build Tools](https://www.visualstudio.com/downloads/#build-tools-for-visual-studio-2017)
7. Add relevant bin directories to the `%PATH%` environment variable.

P.S. We can also do similar installation procedure to get the plugin working on a Windows 10 VM.

### Compiling the `vmci_client.dll`
1. Fire up a command prompt and type `powershell`.
2. Get the vDVS plugin `go get github.com/venilnoronha/docker-volume-service`
3. Rename the `venilnoronha` to `vmware` under `$GOPATH/src/github.com/`.
4. `cd` into the `esx_service/vmci` directory within `docker-volume-service`.
5. Compile the `vmci_client.dll` with something like:
```
cl /D_USRDLL /D_WINDLL .\vmci_client.c -I "C:\Program Files (x86)\Windows Kits\10\Include\10.0.15063.0\ucrt" -I "C:\Program Files (x86)\Windows Kits\10\Include\10.0.15063.0\um" -I "C:\Program Files (x86)\Windows Kits\10\Include\10.0.15063.0\shared" -I "C:\Program Files (x86)\Windows Kits\10\Include\10.0.15063.0\winrt" -I "C:\Program Files (x86)\Microsoft Visual Studio\2017\Community\VC\Tools\MSVC\14.10.25017\include" /link /defaultlib:ws2_32.lib /libpath:"C:\Program Files (x86)\Windows Kits\10\Lib\10.0.15063.0\um\x64" /libpath:"C:\Program Files (x86)\Windows Kits\10\Lib\10.0.15063.0\ucrt\x64" /libpath:"C:\Program Files (x86)\Microsoft Visual Studio\2017\Community\VC\Tools\MSVC\14.10.25017\lib\x64" /DLL /OUT:vmci_client.dll /DEF:vmci_client.def
```

### Running the plugin on Windows
1. Copy the generated `vmci_client.dll` to `vmdk_plugin/` and `vmdk_plugin/drivers/vmdk/vmdkops/` directories.
2. `go get github.com/Microsoft/go-winio` package into the `vendor` directory.
3. Delete `vmdk_plugin/drivers/vmdk/vmdkops/cmd_test.go`
4. In `vmdk_plugin/drivers/vmdk/vmdkops/mock_vmdkcmd.go`.
    - Replace `// +build linux` with `// +build linux windows`.
    - Comment `syscall.Fallocate` and related code.
5. In `vmdk_plugin/utils/refcount/refcount.go`.
    - Replace `// +build linux` with `// +build linux windows`.
6. In `vmdk_plugin/utils/fs/fs.go`.
    - Replace `// +build linux` with `// +build linux windows`.
    - Empty out all function bodies and related imports.
7. `cd` to the `vmdk_plugin` directory.
8. In `main.go` comment the line that says `...VmwareFormatter...`.
9. Do `go run -v main.go` and wait for the `Pipe listening...` message.
10. Create a `vsphere.json` file under `C:\ProgramData\docker\plugins\` with the following contents.
```
{
    "Name": "vsphere",
    "Addr": "npipe:////./pipe/vsphere-dvs"
}
```
10. Create volumes using docker cli. Ex: `docker volume create --driver=vsphere mongo-volume`.


### Issues
1. fs.go is linux specific. (inotify can be replaced with fsnotify)
2. refcount.go is linux specific.
3. docker on windows doesn't support case sensitive volume names. (needs handling in vmdk_ops.py driver)
