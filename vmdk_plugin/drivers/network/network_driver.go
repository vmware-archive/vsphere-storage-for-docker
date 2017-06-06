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

package network

//
// NFS backed  VolumeImpl Driver.
//
// Provide support for NFS based file backed volumes.
//

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/drivers"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/config"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/fs"
	"github.com/vmware/docker-volume-vsphere/vmdk_plugin/utils/plugin_utils"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	fsType      = "nfs"
	networkRoot = "remote/dockvols/"
	defaultPerm = 0775
	// Vol stats
	stat_size  = "Size(bytes)"
	stat_mtime = "ModifiedTime"
	stat_mode  = "Mode"
	stat_path  = "Path"
	stat_ctime = "CreateTime"
	stat_atime = "AccessTime"
	stat_ucnt  = "Current user count"
)

type mountedDir struct {
	volsDir    string
	remotePath string
}

// VolumeImplDriver - NFS based volume driver meta-data
type VolumeImplDriver struct {
	config       config.Config
	mountRoot    string
	volDriver    drivers.VolumeDriver
	defaultLabel string
	//mountedRdirs []mountedDir // TBD for unmount on cleanup
}

// NewVolumeImplDriver creates Driver which to real ESX (useMockEsx=False) or a mock
func Init(vdriver drivers.VolumeDriver, mountDir string, config config.Config) (*VolumeImplDriver, error) {
	var d *VolumeImplDriver

	// Init all known backends - VMDK and network volume drivers
	d = new(VolumeImplDriver)
	d.config = config
	d.defaultLabel = config.RemoteDirs.Default
	d.mountRoot = filepath.Join(mountDir, networkRoot)
	d.volDriver = vdriver
	return d, nil
}

// Get info about a single volume
func (d *VolumeImplDriver) Get(r volume.Request) volume.Response {
	status, err := d.GetVolume(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	mountpoint := d.GetMountPoint(r.Name)
	return volume.Response{Volume: &volume.Volume{Name: r.Name,
		Mountpoint: mountpoint,
		Status:     status}}
}

func (d *VolumeImplDriver) getVolumeList() ([]string, error) {
	var volumes []string

	msg := "Failed list volume "
	for dslabel := range d.config.RemoteDirs.RemoteDirTbl {
		rdir := d.config.RemoteDirs.RemoteDirTbl[dslabel]
		if strings.Compare(rdir.FSType, fsType) != 0 {
			continue
		}

		// Returns the path under which the named volume is created
		path, err := d.checkLabelIsMounted(rdir, dslabel)
		if err != nil {
			mountErrMsg := msg + " remote dir not accessible " + dslabel + "- "
			return nil, errors.New(mountErrMsg + err.Error())
		}

		chkLabel := "@" + dslabel
		err = filepath.Walk(path, func(wpath string, finfo os.FileInfo, err error) error {
			if finfo.IsDir() && strings.Contains(finfo.Name(), chkLabel) {
				volumes = append(volumes, finfo.Name())
			}
			return err
		})

		if err != nil {
			return nil, err
		}
	}

	return volumes, nil
}

// List volumes known to the driver
func (d *VolumeImplDriver) List(r volume.Request) ([]*volume.Volume, error) {
	volumes, err := d.getVolumeList()
	if err != nil {
		return nil, err
	}
	responseVolumes := make([]*volume.Volume, 0, len(volumes))
	for _, vol := range volumes {
		mountpoint := d.GetMountPoint(vol)
		responseVol := volume.Volume{Name: vol, Mountpoint: mountpoint}
		responseVolumes = append(responseVolumes, &responseVol)
	}
	return responseVolumes, nil
}

// mountRDirAndGetVolPath - For a given name (volname@label) resolve the remote dir
// label, mount the remote dir if needed and return the path to the volume folder.
// Note: volume folder may not be created yet.
func (d *VolumeImplDriver) mountRDirAndGetVolPath(name string, msg string) (string, error) {
	dslabel := d.getDSLabel(name)

	if dslabel == "" {
		return "", errors.New(msg + " unknown label or FS type.")
	}

	rdir := d.config.RemoteDirs.RemoteDirTbl[dslabel]

	// Returns the path under which the named volume is created
	path, err := d.checkLabelIsMounted(rdir, dslabel)
	if err != nil {
		mountErrMsg := msg + " remote dir not accessible " + dslabel + "- "
		return "", errors.New(mountErrMsg + err.Error())
	}

	volpath := filepath.Join(path, strings.Split(name, "@")[0])
	return volpath, nil
}

// GetVolume - return volume meta-data, name is like volname@label.
func (d *VolumeImplDriver) GetVolume(name string) (map[string]interface{}, error) {
	var volStat map[string]interface{}
	var size int64
	var stats syscall.Stat_t

	msg := "Failed Get volume  - " + name
	volpath, err := d.mountRDirAndGetVolPath(name, msg)
	if err != nil {
		return nil, err
	}

	err = syscall.Lstat(volpath, &stats)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(volpath, func(wpath string, finfo os.FileInfo, err error) error {
		if !finfo.IsDir() {
			size += finfo.Size()
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	// Create volume status map
	volStat[stat_path] = volpath
	volStat[stat_size] = size
	volStat[stat_mode] = stats.Mode
	stat_time := time.Unix(stats.Mtim.Sec, stats.Mtim.Nsec)
	volStat[stat_mtime] = stat_time.Format(time.UnixDate)

	stat_time = time.Unix(stats.Atim.Sec, stats.Atim.Nsec)
	volStat[stat_atime] = stat_time.Format(time.UnixDate)

	stat_time = time.Unix(stats.Ctim.Sec, stats.Ctim.Nsec)
	volStat[stat_ctime] = stat_time.Format(time.UnixDate)

	volStat[stat_ucnt] = d.volDriver.GetRefCount(name)

	return volStat, nil
}

// MountVolume- name is like volname@label, return the mount path for the volume
func (d *VolumeImplDriver) MountVolume(name string, fstype string, id string, isReadOnly bool, skipAttach bool) (string, error) {
	msg := "Failed mount volume  - " + name
	volpath, err := d.mountRDirAndGetVolPath(name, msg)

	if err != nil {
		return "", err
	}

	return volpath, err
}

// UnmountVolume - nothing to do
func (d *VolumeImplDriver) UnmountVolume(name string) error {
	return nil
}

// getPathsForRDir - resolves the remote dir to a path on the host
// returns the path to create volumes and the remote path to mount
func (d *VolumeImplDriver) getPathsForRDir(rdir config.RemoteDir, dslabel string) mountedDir {
	var vdir, rpath string
	var pdir *config.RemoteDir
	var tmpdir config.RemoteDir

	mpath := filepath.Join(d.mountRoot, dslabel)

	// If rdir has a redirect to a parent then use the parent
	if rdir.Src != "" {
		// Use the remote dir that the Src references
		tmpdir = d.config.RemoteDirs.RemoteDirTbl[rdir.Src]
		pdir = &tmpdir
	}

	if pdir != nil {
		// If a parent dir must be used then the volumes are placed
		// within the volpath of the parent + the volpath for the
		// label provided
		vdir = filepath.Join(mpath, pdir.VolPath, rdir.VolPath)
		rpath = pdir.Path
	} else {
		vdir = filepath.Join(mpath, rdir.VolPath)
		rpath = rdir.Path
	}
	return mountedDir{volsDir: vdir, remotePath: rpath}
}

// checkLabel - resolve label to mount params used to mount remote dir,
// mount remote dir and create vol-path relative to that.
func (d *VolumeImplDriver) checkLabelIsMounted(rdir config.RemoteDir, dslabel string) (string, error) {
	dirPaths := d.getPathsForRDir(rdir, dslabel)

	// Check and mount mountPath
	mountPath := filepath.Join(d.mountRoot, dslabel)
	if !plugin_utils.IsMounted(mountPath, d.mountRoot) {
		err := fs.DoNFSMountCmd(mountPath, rdir.FSType, dirPaths.remotePath, rdir.Args)
		if err != nil {
			return "", err
		}
	}

	// Check and create the vol path under mountPath
	err := makeVolumeDir(dirPaths.volsDir)

	if err != nil {
		return "", err
	}

	return dirPaths.volsDir, err
}

func makeVolumeDir(path string) error {
	err := os.MkdirAll(path, defaultPerm)
	if os.IsExist(err) {
		return nil
	}
	return err
}

// getDSLabel - return the label for the remote dir from
// the given name (volname@label), using the default if
// no label was specified. Check the FS type matches the
// type supported.
func (d *VolumeImplDriver) getDSLabel(name string) string {
	dslabel := plugin_utils.GetDSLabel(name)

	if dslabel == "" {
		// Use the default label else error
		dslabel = d.defaultLabel
	}

	if dslabel == "" {
		return ""
	} else {
		rdir := d.config.RemoteDirs.RemoteDirTbl[dslabel]
		// TBD - handle Src setting for a remote dir
		if strings.Compare(rdir.FSType, fsType) != 0 {
			return ""
		}
		return dslabel
	}
}

// Create - create a volume.
func (d *VolumeImplDriver) Create(r volume.Request) volume.Response {
	// Check if the default label is to be used or pick the
	// specified label and mount that if not already mounted
	msg := "Failed create volume  - " + r.Name
	volpath, err := d.mountRDirAndGetVolPath(r.Name, msg)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	err = makeVolumeDir(volpath)
	if err != nil {
		// The remote dir is left mounted there may be other users
		// already using volumes on this rdir. Or another create or
		// mount may come by and we don't want to repeat all the
		// actions again.
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Err: ""}
}

// Remove - removes individual volume. Docker would call it only if is not using it anymore
func (d *VolumeImplDriver) Remove(r volume.Request) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Removing volume ")
	msg := "Failed remove volume  - " + r.Name
	volpath, err := d.mountRDirAndGetVolPath(r.Name, msg)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	err = os.RemoveAll(volpath)
	return volume.Response{Err: err.Error()}
}

// Path - give docker a reminder of the volume mount path
func (d *VolumeImplDriver) Path(r volume.Request) volume.Response {
	return volume.Response{Mountpoint: d.GetMountPoint(r.Name)}
}

// Mount - Provide a volume to docker container - called once per container start.
func (d *VolumeImplDriver) Mount(r volume.MountRequest, volumeInfo *plugin_utils.VolumeInfo) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Mounting volume ")
	path, err := d.MountVolume(r.Name, "", "", true, true)

	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Mountpoint: path}
}

// Unmount request from Docker. If mount refcount is drop to 0.
// Unmount and detach from VM
func (d *VolumeImplDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.WithFields(log.Fields{"name": r.Name}).Info("Unmounting Volume ")
	return volume.Response{Err: ""}
}

func (d *VolumeImplDriver) GetMountPoint(name string) string {
	dslabel := d.getDSLabel(name)
	rdir := d.config.RemoteDirs.RemoteDirTbl[dslabel]
	rpath := d.getPathsForRDir(rdir, dslabel)
	return filepath.Join(rpath.volsDir, strings.Split(name, "@")[0])
}

// IsMounted - return true if the remote dir hosting the named volume
// is mounted, name is like (volname@label).
func (d *VolumeImplDriver) IsMounted(name string) bool {
	// Check if the label for the given name is mounted
	if label := plugin_utils.GetDSLabel(name); label != "" {
		return plugin_utils.IsMounted(label, d.mountRoot)
	}
	return false
}
