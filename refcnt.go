//
// Refcount discovery from local docker.
//
// - Connects to Docker over unix socket, enumerates Volume Mounts and builds
//   "volume mounts refcount" map as Docker sees it.
// - Also gets actual mounts from /proc/mounts, and makes sure the refcounts and
//   actual mounts are in sync.
// - The process is initiated on plugin start.  If it fails (e.g. docker is not
//    running), we sleep/retry until Docker answers.
//
// After refcount discovery, results are compared to /proc/mounts content.
//
// We rely on all plugin mounts being in /mnt/vmdk/<volume_name>, and will
// unount stuff there at will - this place SHOULD NOT be used for manual mounts.
//          TODO: reject file names > 255 chars on ‘create’
//
// If a volume IS mounted, but should not be (refcount = 0)
//   - we assume there was a restart of either Docker or VM or even ESX, and
//     the mount is stale (since Docker does not need it)
//   - we unmount / detach it
//
// If a volume is NOT mounted, but should be (refcount > 0)
//   - this should never happen since Docker using volume means bind mount is
//     active so the disk could not have been unmounted
//   - we just log an error and keep going. Recovery in this case is manual
//
// We assume that mounted (in Docker VM) and attached (to Docker VM) is the
// same. If something is attached to VM but not mounted, we leave it to manual
// recovery.
//
// Reason: if  Docker is killed (-9) , or VM (or ESX) crashes. Docker may forget
// all about volume mounts and consider clean slate. This could lead to plugin
// locking volumes which Docker does not need so we need to recover correct
// refcounts
//
// Note: when Docker Is properly restarted ('service docker restart), it shuts
// down accurately  and sends unmounts so refcounts stays in sync. In this case
// there is no need to do anythign specia in the plugin. Thus all this code is
// for crash recovery and cleanup.
//

package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

const (
	apiVersion              = "v1.22"
	dockerUSocket           = "unix:///var/run/docker.sock"
	defaultSleepIntervalSec = 3

	// consts for finding and parsing linux mount information
	linuxMountsFile = "/proc/mounts"
)

// info about individual volume ref counts and mount
type refCount struct {
	count   int    // refcount
	mounted bool   // is the volume mounted (from OS point of view)
	dev     string // mounted from device <dev/> (or Nil)
}

// volume name -> {count, mounted, device}
type refCountsMap map[string]*refCount


var (
    // vmdk or local
	driverName             string
	
	// header for Docker Remote API
	defaultHeaders         map[string]string
	
	// wait time for reconnect
	reconnectSleepInterval time.Duration
	
	// volume name -> {refcount, mounted, device} for volumes with refcount > 0
	refCounts refCountsMap
)

func init() {
	defaultHeaders = map[string]string{"User-Agent": "engine-api-client-1.0"}

	// defaults: in test they can be overwritten (see sanity_test.go)
	reconnectSleepInterval = defaultSleepIntervalSec * time.Second
	driverName = "vmdk"
}

// Init volume usage refcounts from Docker. This will poll until replied to.
func refCountsInit()  {

	refCounts = make(refCountsMap)
	err := refCounts.connectAndDiscover()
	if err != nil {
		panic(fmt.Sprintf("Failed to get volume info from Docker (%v)", err))
	}
	log.Infof("Discovered %d volumes in use.", len(refCounts))
	if log.GetLevel() == log.DebugLevel {
		for name, cnt := range refCounts {
			log.Debugf("Volume name=%s count=%d mounted=%d",
				name, cnt.count, cnt.mounted)
		}
	}
}

// Returns ref count for the volume.
// If volume is not referred (not in the map), return 0
// NOTE: this assumes the caller holds the lock if we run concurrently
func (r refCountsMap) getCount(vol string) int {
	rc := r[vol]
	if rc == nil {
		return 0
	}
	return rc.count
}

// incr refCount for the volume vol. Creates new entry if needed.
func (r refCountsMap) incr(vol string) int {
	vc := r[vol]
	if vc == nil {
		vc = &refCount{count: 0, mounted: false}
		r[vol] = vc
	}
	vc.count++
	return vc.count
}

// decr recfcount for the volume vol and returns the new count
// returns -1  for error (and resets count to 0)
// also deletes the node from the map if refcount drops to 0
func (r refCountsMap) decr(vol string) int {
	vc := r[vol]
	if vc == nil {
		log.Errorf("decr: refcount is missing, setting to 0. name=%s", vol)
		return -1
	} else if vc.count <= 0 {
		log.Errorf("decr: Corrupted refcount, setting to 0. name=%s count=%d",
			vol, vc.count)
		delete(r, vol)
		return -1
	}
	vc.count--
	if vc.count == 0 {
		delete(r, vol)
	}
	return vc.count
}

// connects (and polls if needed) and then calls discover()
func (r refCountsMap) connectAndDiscover() error {
	c, err := client.NewClient(dockerUSocket, apiVersion, nil, defaultHeaders)
	for err != nil {
		log.Errorf("connectAndDiscover: Failed to create client for %s.( %v)",
			dockerUSocket, err)
		return err
	}

	log.Infof("connectAndDiscover: Getting volume data from to %s", dockerUSocket)
	err = r.discoverAndSync(c)
	for err != nil {
		// stay here forever, until connected to Docker or killed
		log.Errorf("Failed to get data from Docker, retrying. (err: %v)", err)
		time.Sleep(reconnectSleepInterval)
		err = r.discoverAndSync(c)
	}
	return nil
}

// enumberates volumes and  builds refCountsMap, then sync with mount info
func (r refCountsMap) discoverAndSync(c *client.Client) error {
	// we assume to  have empty refcounts. Let's enforce
	for name := range r {
		delete(r, name)
	}

	filters := filters.NewArgs()
	filters.Add("status", "running")
	filters.Add("status", "paused")
	containers, err := c.ContainerList(types.ContainerListOptions{
		All:    true,
		Filter: filters,
	})
	if err != nil {
		return err
	}

	log.Debugf("Found %d running or paused containers", len(containers))
	for _, ct := range containers {
		containerJSONInfo, err := c.ContainerInspect(ct.ID)
		if err != nil {
			log.Errorf("ContainerInspect failed for %s (err: %v)", ct.Names, err)
			continue
		}
		log.Debugf("  Mounts for %v", ct.Names)

		for _, mount := range containerJSONInfo.Mounts {
			if mount.Driver == driverName {
				r.incr(mount.Name)
				log.Debugf("  name=%v (driver=%s source=%s)",
					mount.Name, mount.Driver, mount.Source)
			}
		}
	}

	// Check that refcounts and actual mount info from Linux match
	// If they don't, unmount unneeded stuff, or yell if something is
	// not mounted but should be (it's error. we should not get there)

	r.getMountInfo()
	r.syncMountsWithRefCounters()

	return nil
}

// syncronize mount info with refcounts - and unmounts if needed
func (r refCountsMap) syncMountsWithRefCounters() {
	for vol, cnt := range r {
		log.Debugf("Volume %s %+v", vol, cnt)
		if cnt.mounted == true {
			if cnt.count == 0 {
				// Volume mounted but not used - unmount !
				log.Infof("Volume %s (%s) is not used - unmounting", vol, cnt.dev)
				// TBD **** ACTUAL UNMOUNT & DETACH ***
			}
			// else: volume mounted and used - all good, nothing to do

		} else {
			// volume unmounted. We should not even get here since unmounted
			// volumes should have no record in refcount maps (refcnt=0)
			log.Errorf("Volume %s is not mounted but refcnt=%d. %+v",
				vol, cnt.count, cnt)
				// TBD **** CALLBACK UP , PROBBALY TO EXIT ***
		}
	}
}

// scans /proc/mounts and updates refcount map witn mounted volumes
func (r refCountsMap) getMountInfo() error {
	data, err := ioutil.ReadFile(linuxMountsFile)
	if err != nil {
		log.Errorf("Can't get info from %s (%v)", linuxMountsFile, err)
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		field := strings.Fields(line)
		if len(field) < 2 {
			continue // skip empty line and lines too short to have our mount
		}
		// fields format: [/dev/sdb /mnt/vmdk/vol1 ext2 rw,relatime 0 0]
		if filepath.Dir(field[1]) != mountRoot {
			continue
		}
		volName := filepath.Base(field[1])
		refInfo := r[volName]
		if refInfo == nil {
		    refInfo = &refCount{count:0}
		}
		refInfo.mounted = true
		refInfo.dev = field[0]
		log.Debugf("  found a mounted volume %s (%+v)", volName, refInfo)
	}

	return nil
}
