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

// ETCD implementation for KV Store interface

package etcdops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	etcdClient "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/vfile/dockerops"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/vfile/kvstore"
)

/*
   defaultEtcdClientPort:      default port for etcd clients to talk to the peers
   defaultEtcdPeerPort:        default port for etcd peers talk to each other
   etcdClusterToken:           ID of the cluster to create/join
   etcdListenURL:              etcd listening interface
   etcdScheme:                 Protocol used for communication
   etcdDataDir:                Data directory for ETCD
   etcdClusterStateNew:        Used to indicate the formation of a new
                               cluster
   etcdClusterStateExisting:   Used to indicate that this node is joining
                               an existing etcd cluster
   etcdRequestTimeout:         After how long should an etcd request timeout
   etcdUpdateTimeout:          Timeout for waiting etcd server change status
   checkSleepDuration:         How long to wait in any busy waiting situation
                               before checking again
   gcTicker:                   ticker for garbage collector to run a collection
   etcdClientCreateError:      Error indicating failure to create etcd client
   etcdUnhealthyErrorMsg:      Message indicating etcd is unhealthy
   etcdNoRef:                  global refcount = 0, no dependency on this volume
   etcdLockTicker:             ticker for blocking wait a ETCD lock
   etcdLockTimer:              timeout for blocking wait a ETCD lock
   etcdLockTimeoutErrMsg:      Message indicating the failure to block wait a ETCD lock
*/
const (
	etcdDataDir              = "/etcd-data"
	defaultEtcdClientPort    = ":2379"
	defaultEtcdPeerPort      = ":2380"
	etcdClusterToken         = "vfile-etcd-cluster"
	etcdListenURL            = "0.0.0.0"
	etcdScheme               = "http://"
	etcdClusterStateNew      = "new"
	etcdClusterStateExisting = "existing"
	etcdRequestTimeout       = 2 * time.Second
	etcdUpdateTimeout        = 10 * time.Second
	checkSleepDuration       = time.Second
	gcTicker                 = 15 * time.Second
	etcdClientCreateError    = "Failed to create etcd client"
	etcdUnhealthyErrorMsg    = "ETCD maybe unhealthy"
	etcdNoRef                = "0"
	etcdLockTicker           = 1 * time.Second
	etcdLockTimer            = 20 * time.Second
	etcdLockTimeoutErrMsg    = "ETCD Lock blocking wait timeout"
	etcdLockLeaseTime        = 20
)

type EtcdKVS struct {
	dockerOps *dockerops.DockerOps
	nodeID    string
	nodeAddr  string
	// isManager indicates if current node is a manager or not
	isManager bool
	// etcdCMD records the cmd struct for ETCD process
	// it's used for stopping ETCD process when node is demoted
	etcdCMD *exec.Cmd
	// watcher is used for killing the watch request when node is demoted
	watcher *etcdClient.Client
	// etcdClientPort is the port for etcd clients to talk to the peers
	etcdClientPort string
	// etcdPeerPort is port for etcd peers talk to each other
	etcdPeerPort string
}

type EtcdLock struct {
	// Key for this lock
	Key string
	// lockCli to hold the client for the lock
	lockCli *etcdClient.Client
	// lockSession to hold the session for the lock
	lockSession *concurrency.Session
	// lockMutex to hold the mutex for the lock
	lockMutex *concurrency.Mutex
}

// VFileVolConnectivityData - Contains metadata of vFile volumes
type VFileVolConnectivityData struct {
	Port        int    `json:"port,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
}

// NewKvStore function: start or join ETCD cluster depending on the role of the node
func NewKvStore(dockerOps *dockerops.DockerOps) *EtcdKVS {
	var e *EtcdKVS

	// get swarm info from docker client
	nodeID, addr, isManager, err := dockerOps.GetSwarmInfo()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to get swarm Info from docker client ")
		return nil
	}

	etcdClientPort, etcdPeerPort := getEtcdPorts()

	e = &EtcdKVS{
		dockerOps:      dockerOps,
		nodeID:         nodeID,
		nodeAddr:       addr,
		isManager:      isManager,
		etcdClientPort: etcdClientPort,
		etcdPeerPort:   etcdPeerPort,
	}

	if !isManager {
		log.WithFields(
			log.Fields{"nodeID": nodeID},
		).Info("Swarm node role: worker. Return from NewKvStore ")
		// start helper before return
		go e.etcdHelper()
		return e
	}

	// check if ETCD data-dir already exists
	_, err = os.Stat(etcdDataDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("failed to stat ETCD data-dir: %v", err)
			return nil
		}
		// when error is IsNotExist, ETCD data-dir is not existing,
		// need to continue to create/join a new ETCD cluster
		log.Infof("ETCD data-dir does not exist, continue to create/join a new ETCD cluster")
	} else {
		// ETCD data-dir already exists, just re-join
		log.Infof("ETCD data-dir exists, continue to rejoin")
		err = e.rejoinEtcdCluster()
		if err != nil {
			log.Errorf("Failed to rejoin the ETCD cluster: %v", err)
			return nil
		}
		return e
	}

	// check my local role
	isLeader, err := dockerOps.IsSwarmLeader(nodeID)
	if err != nil {
		log.WithFields(
			log.Fields{
				"nodeID": nodeID,
				"error":  err},
		).Error("Failed to check swarm leader status from docker client ")
		return nil
	}

	// if leader, proceed to start ETCD cluster
	if isLeader {
		log.WithFields(
			log.Fields{"nodeID": nodeID},
		).Info("Swarm node role: leader, start ETCD cluster ")
		err = e.startEtcdCluster()
		if err != nil {
			log.WithFields(
				log.Fields{
					"nodeID": nodeID,
					"error":  err},
			).Error("Failed to start ETCD Cluster ")
			return nil
		}
		// start helper before return
		go e.etcdHelper()
		return e
	}

	// if manager, join ETCD cluster
	err = e.joinEtcdCluster()
	if err != nil {
		log.WithFields(log.Fields{
			"nodeID": nodeID,
			"error":  err},
		).Error("Failed to join ETCD Cluster")
		return nil
	}
	// start helper before return
	go e.etcdHelper()
	return e
}

func getEtcdPorts() (string, string) {
	etcdClientPort := os.Getenv("VFILE_ETCD_CLIENT_PORT")
	etcdPeerPort := os.Getenv("VFILE_ETCD_PEER_PORT")

	if etcdClientPort == "" {
		etcdClientPort = defaultEtcdClientPort
	} else {
		etcdClientPort = ":" + etcdClientPort
	}

	if etcdPeerPort == "" {
		etcdPeerPort = defaultEtcdPeerPort
	} else {
		etcdPeerPort = ":" + etcdPeerPort
	}
	log.Infof("getEtcdPorts: clientPort=%s peerPort=%s", etcdClientPort, etcdPeerPort)
	return etcdClientPort, etcdPeerPort
}

// rejoinEtcdCluster function is called when a node need to rejoin a ETCD cluster
func (e *EtcdKVS) rejoinEtcdCluster() error {
	nodeID := e.nodeID
	nodeAddr := e.nodeAddr
	etcdClientPort := e.etcdClientPort
	etcdPeerPort := e.etcdPeerPort
	log.Infof("rejoinEtcdCluster on node with nodeID %s and nodeAddr %s", nodeID, nodeAddr)
	lines := []string{
		"--name", nodeID,
		"--data-dir", etcdDataDir,
		"--advertise-client-urls", etcdScheme + nodeAddr + etcdClientPort,
		"--initial-advertise-peer-urls", etcdScheme + nodeAddr + etcdPeerPort,
		"--listen-client-urls", etcdScheme + etcdListenURL + etcdClientPort,
		"--listen-peer-urls", etcdScheme + etcdListenURL + etcdPeerPort,
	}

	// start the routine to create an etcd cluster
	err := e.etcdStartService(lines)
	if err != nil {
		log.Errorf("Failed to start ETCD for rejoinEtcdCluster")
		return err
	}

	// check if successfully joined the etcd cluster, then start the watcher
	return e.checkLocalEtcd()
}

// startEtcdCluster function is called by swarm leader to start a ETCD cluster
func (e *EtcdKVS) startEtcdCluster() error {
	nodeID := e.nodeID
	nodeAddr := e.nodeAddr
	etcdClientPort := e.etcdClientPort
	etcdPeerPort := e.etcdPeerPort
	log.Infof("startEtcdCluster on node with nodeID %s and nodeAddr %s", nodeID, nodeAddr)

	files, err := filepath.Glob(etcdDataDir)
	log.Debugf("Files in etcdDataDir: %s", files)
	// create ETCD data directory with 755 permission since only root needs full permission
	err = os.Mkdir(etcdDataDir, 0755)
	if err != nil {
		log.Errorf("Failed to create directory etcd-data: err %v", err)
		return err
	}

	lines := []string{
		"--name", nodeID,
		"--data-dir", etcdDataDir,
		"--advertise-client-urls", etcdScheme + nodeAddr + etcdClientPort,
		"--initial-advertise-peer-urls", etcdScheme + nodeAddr + etcdPeerPort,
		"--listen-client-urls", etcdScheme + etcdListenURL + etcdClientPort,
		"--listen-peer-urls", etcdScheme + etcdListenURL + etcdPeerPort,
		"--initial-cluster-token", etcdClusterToken,
		"--initial-cluster", nodeID + "=" + etcdScheme + nodeAddr + etcdPeerPort,
		"--initial-cluster-state", etcdClusterStateNew,
	}

	// start the routine to create an etcd cluster
	err = e.etcdStartService(lines)
	if err != nil {
		log.Errorf("Failed to start ETCD for startEtcdCluster")
		return err
	}

	// check if etcd cluster is successfully started, then start the watcher
	return e.checkLocalEtcd()
}

// joinEtcdCluster function is called by a non-leader swarm manager to join a ETCD cluster
func (e *EtcdKVS) joinEtcdCluster() error {
	nodeAddr := e.nodeAddr
	nodeID := e.nodeID
	etcdClientPort := e.etcdClientPort
	etcdPeerPort := e.etcdPeerPort
	log.Infof("joinEtcdCluster on node with nodeID %s and nodeAddr %s", nodeID, nodeAddr)

	leaderAddr, err := e.dockerOps.GetSwarmLeader()
	if err != nil {
		log.WithFields(
			log.Fields{
				"nodeID": nodeID,
				"error":  err},
		).Error("Failed to get swarm leader address ")
		return err
	}

	etcd, err := e.addrToEtcdClient(leaderAddr)
	if err != nil {
		log.WithFields(
			log.Fields{"nodeAddr": nodeAddr,
				"leaderAddr": leaderAddr,
				"nodeID":     nodeID},
		).Error("Failed to join ETCD cluster on manager ")
		return err
	}
	defer etcd.Close()

	// list all current ETCD members, check if this node is already added as a member
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	lresp, err := etcd.MemberList(ctx)
	cancel()
	if err != nil {
		log.WithFields(
			log.Fields{"leaderAddr": leaderAddr,
				"error":       err,
				"members len": len(lresp.Members)},
		).Error("Failed to list member for ETCD")
		return err
	}

	peerAddr := etcdScheme + nodeAddr + etcdPeerPort
	existing := false
	for _, member := range lresp.Members {
		// loop all current etcd members to find if there is already a member with the same peerAddr
		if member.PeerURLs[0] == peerAddr {
			if member.Name == "" {
				// same peerAddr already existing
				// empty name indicates this member is not started, continue the join process
				log.WithFields(
					log.Fields{"nodeID": nodeID,
						"peerAddr": peerAddr},
				).Info("Already joined as ETCD member but not started. ")

				existing = true
			} else {
				// same peerAddr already existing and started, need to remove before re-join
				// we need the remove since etcd data directory is not persistent
				// thus the node cannot re-join as the same member as before
				log.WithFields(
					log.Fields{"nodeID": nodeID,
						"peerAddr": peerAddr},
				).Info("Already joined as a ETCD member and started. Action: remove self before re-join ")

				ctx, cancel = context.WithTimeout(context.Background(), etcdRequestTimeout)
				_, err = etcd.MemberRemove(ctx, member.ID)
				cancel()
				if err != nil {
					log.WithFields(
						log.Fields{"peerAddr": peerAddr,
							"member.ID": member.ID},
					).Error("Failed to remove member for ETCD")
					return err
				}
			}
			// the same peerAddr can only join at once. no need to continue.
			break
		}
	}

	initCluster := ""
	if !existing {
		peerAddrs := []string{peerAddr}
		ctx, cancel = context.WithTimeout(context.Background(), etcdRequestTimeout)
		aresp, err := etcd.MemberAdd(ctx, peerAddrs)
		cancel()
		if err != nil {
			log.WithFields(
				log.Fields{"leaderAddr": leaderAddr,
					"error":       err,
					"members len": len(aresp.Members)},
			).Error("Failed to add member for ETCD")
			return err
		}
		for _, member := range aresp.Members {
			if member.Name != "" {
				initCluster += member.Name + "=" + member.PeerURLs[0] + ","
			}
		}
	} else {
		for _, member := range lresp.Members {
			if member.Name != "" {
				initCluster += member.Name + "=" + member.PeerURLs[0] + ","
			}
		}
	}

	// create new ETCD data-dir
	err = os.Mkdir(etcdDataDir, 0755)
	if err != nil {
		log.Errorf("Failed to create directory etcd-data: err %v", err)
		return err
	}

	lines := []string{
		"--name", nodeID,
		"--data-dir", etcdDataDir,
		"--advertise-client-urls", etcdScheme + nodeAddr + etcdClientPort,
		"--initial-advertise-peer-urls", etcdScheme + nodeAddr + etcdPeerPort,
		"--listen-client-urls", etcdScheme + etcdListenURL + etcdClientPort,
		"--listen-peer-urls", etcdScheme + etcdListenURL + etcdPeerPort,
		"--initial-cluster-token", etcdClusterToken,
		"--initial-cluster", initCluster + nodeID + "=" + etcdScheme + nodeAddr + etcdPeerPort,
		"--initial-cluster-state", etcdClusterStateExisting,
	}

	// start the routine for joining an etcd cluster
	err = e.etcdStartService(lines)
	if err != nil {
		log.Errorf("Failed to start ETCD for joinEtcdCluster")
		return err
	}

	// check if successfully joined the etcd cluster, then start the watcher
	return e.checkLocalEtcd()
}

// leaveEtcdCluster function is called when a manager is demoted
func (e *EtcdKVS) leaveEtcdCluster() error {
	etcd, err := e.addrToEtcdClient(e.nodeAddr)
	if err != nil {
		log.WithFields(
			log.Fields{"nodeAddr": e.nodeAddr,
				"nodeID": e.nodeID},
		).Error("Failed to create ETCD client from own address")
		return err
	}
	defer etcd.Close()

	// list all current ETCD members
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	lresp, err := etcd.MemberList(ctx)
	cancel()
	if err != nil {
		log.WithFields(
			log.Fields{"nodeAddr": e.nodeAddr,
				"error": err},
		).Error("Failed to list member for ETCD")
		return err
	}

	// create the peer URL for filtering ETCD member information
	// each ETCD member has a unique peer URL
	etcdPeerPort := e.etcdPeerPort
	peerAddr := etcdScheme + e.nodeAddr + etcdPeerPort
	for _, member := range lresp.Members {
		// loop all current etcd members to find if there is already a member with the same peerAddr
		if member.PeerURLs[0] == peerAddr {
			log.WithFields(
				log.Fields{"nodeID": e.nodeID,
					"peerAddr": peerAddr},
			).Info("Remove self from ETCD member due to demotion. ")

			ctx, cancel = context.WithTimeout(context.Background(), etcdRequestTimeout)
			_, err = etcd.MemberRemove(ctx, member.ID)
			cancel()
			if err != nil {
				log.WithFields(
					log.Fields{"peerAddr": peerAddr,
						"member.ID": member.ID},
				).Error("Failed to remove this node from ETCD ")
				return err
			}

			// the same peerAddr can only join at once. no need to continue.
			log.WithFields(
				log.Fields{"peerAddr": peerAddr,
					"member.ID": member.ID},
			).Info("Successfully removed self from ETCD ")
			break
		}
	}

	err = e.etcdStopService()
	return err
}

// etcdStartService function starts an ETCD process
func (e *EtcdKVS) etcdStartService(lines []string) error {
	cmd := exec.Command("/bin/etcd", lines...)
	err := cmd.Start()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err, "cmd": cmd},
		).Error("Failed to start ETCD command ")
		return err
	}

	e.etcdCMD = cmd
	return nil
}

// etcdStopService function stops the ETCD process
func (e *EtcdKVS) etcdStopService() error {
	// stop watcher
	e.watcher.Close()

	// stop ETCD process
	if err := e.etcdCMD.Process.Kill(); err != nil {
		log.Errorf("Failed to stop ETCD process. Error: %v", err)
		return err
	}

	// clean up ETCD data
	if err := os.RemoveAll(etcdDataDir); err != nil {
		log.Errorf("Failed to remove ETCD data directory. Error: %v", err)
		return err
	}

	log.Infof("Stopped ETCD service due to demotion")
	e.etcdCMD = nil
	return nil
}

// checkLocalEtcd function check if local ETCD endpoint is successfully started or not
// if yes, start the watcher for volume global refcount
func (e *EtcdKVS) checkLocalEtcd() error {
	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	timer := time.NewTimer(etcdUpdateTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			log.Infof("Checking ETCD client is started")
			cli, err := e.addrToEtcdClient(e.nodeAddr)
			if err != nil {
				log.WithFields(
					log.Fields{"nodeAddr": e.nodeAddr,
						"error": err},
				).Warningf("Failed to get ETCD client, retry before timeout ")
			} else {
				log.Infof("Local ETCD client is up successfully, start watcher")
				e.watcher = cli
				go e.etcdStartWatcher(cli)
				go e.etcdStopWatcher(cli)
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("Timeout reached; ETCD cluster is not started")
		}
	}
}

// etcdStartWatcher function sets up a watcher to monitor the change to StartTrigger in the KV store
func (e *EtcdKVS) etcdStartWatcher(cli *etcdClient.Client) {
	watchCh := cli.Watch(context.Background(), kvstore.VolPrefixStartTrigger,
		etcdClient.WithPrefix(), etcdClient.WithPrevKV())
	for wresp := range watchCh {
		for _, ev := range wresp.Events {
			e.etcdStartEventHandler(ev)
		}
	}
}

// etcdStopWatcher function sets up a watcher to monitor the change to StopTrigger in the KV store
func (e *EtcdKVS) etcdStopWatcher(cli *etcdClient.Client) {
	watchCh := cli.Watch(context.Background(), kvstore.VolPrefixStopTrigger,
		etcdClient.WithPrefix(), etcdClient.WithPrevKV())
	for wresp := range watchCh {
		for _, ev := range wresp.Events {
			e.etcdStopEventHandler(ev)
		}
	}
}

// etcdHelper: a helper thread which does the following tasks with time interval
// 1. clean up orphan services or orphan internal volumes
// 2. monitor the role of the node, start/shutdown etcd service accordingly
func (e *EtcdKVS) etcdHelper() {
	ticker := time.NewTicker(gcTicker)
	quit := make(chan struct{})

	for {
		select {
		case <-ticker.C:
			// check the role of this node
			err := e.etcdRoleCheck()
			if err != nil {
				log.Warningf("Failed to do role check")
			}

			if e.isManager {
				// find all the vFile volume services
				volumesToVerify, err := e.dockerOps.ListVolumesFromServices()
				if err != nil {
					log.Warningf("Failed to get vFile volumes according to docker services")
				} else {
					e.cleanOrphanService(volumesToVerify)
				}
			}
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func (e *EtcdKVS) etcdRoleCheck() error {
	nodeID, _, isManager, err := e.dockerOps.GetSwarmInfo()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to get swarm Info from docker client ")
		return err
	}

	if isManager {
		if !e.isManager {
			log.Infof("Node is promoted to manager, prepare to join ETCD cluster")
			err = e.joinEtcdCluster()
			if err != nil {
				log.WithFields(log.Fields{
					"nodeID": nodeID,
					"error":  err},
				).Error("Failed to join ETCD Cluster in etcdRoleCheck")
				return err
			}
			e.isManager = true
		}
	} else {
		if e.isManager {
			log.Infof("Node is demoted from manager to worker, prepare to leave ETCD cluster")
			err = e.leaveEtcdCluster()
			if err != nil {
				log.WithFields(log.Fields{
					"nodeID": nodeID,
					"error":  err},
				).Error("Failed to leave ETCD Cluster in etcdRoleCheck")
				return err
			}
			e.isManager = false
		}
	}

	return nil
}

// cleanOrphanService: stop orphan services
func (e *EtcdKVS) cleanOrphanService(volumesToVerify []string) {
	volStates, err := e.KvMapFromPrefix(string(kvstore.VolPrefixState))
	if err != nil {
		// if ETCD is not functionaing correctly, stop and return
		log.Warningf("Failed to get volume states from ETCD due to error %v.", err)
		return
	}

	for _, volName := range volumesToVerify {
		state, found := volStates[string(kvstore.VolPrefixState)+volName]
		if !found ||
			state == string(kvstore.VolStateDeleting) {
			log.Warningf("The service for vFile volume %s needs to be shutdown.", volName)
			e.dockerOps.StopSMBServer(volName)
		}
	}
}

// etcdStartEventHandler function handles the returned event from etcd watcher of the start trigger changes
func (e *EtcdKVS) etcdStartEventHandler(ev *etcdClient.Event) {
	log.WithFields(
		log.Fields{"type": ev.Type},
	).Info("Watcher on start trigger returns event ")

	// monitor PUT requests on start triggers, excluding the PUT for creation
	if ev.Type == etcdClient.EventTypePut && ev.PrevKv != nil {
		log.Debugf("Watcher got a increase event on watcher, new value is %s", string(ev.Kv.Value))
		volName := strings.TrimPrefix(string(ev.Kv.Key), kvstore.VolPrefixStartTrigger)

		// Compare the value of start marker, only one watcher will be able to successfully update the value to
		// the new value of start trigger
		success, err := e.CompareAndPutIfNotEqual(kvstore.VolPrefixStartMarker+volName, string(ev.Kv.Value))
		if err != nil || success == false {
			// watchers failed to update the marker just return
			return
		}

		log.Infof("Successfully update the start marker to the value of start trigger, proceed to next step")

		// Get the lock for changing the state and server
		lock, err := e.CreateLock(kvstore.VolPrefixState + volName)
		if err != nil {
			log.Errorf("Failed to create lock for state changing of %s", volName)
			return
		}

		err = lock.BlockingLockWithLease()
		if err != nil {
			log.Errorf("Failed to blocking wait lock for state changing of %s", volName)
			lock.ClearLock()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
		resp, err := e.watcher.Get(ctx, kvstore.VolPrefixState+volName)
		cancel()
		if err != nil {
			log.WithFields(
				log.Fields{"volName": volName,
					"error": err},
			).Error("Failed to Get state of volume from ETCD ")
			lock.ReleaseLock()
			return
		}

		if string(resp.Kvs[0].Value) == string(kvstore.VolStateMounted) {
			log.Infof("Volume is already in desired state %s", kvstore.VolStateMounted)
			lock.ReleaseLock()
			return
		}

		// start the SMB server
		port, servName, succeeded := e.dockerOps.StartSMBServer(volName)
		if succeeded {
			err = e.updateServerInfo(kvstore.VolPrefixInfo+volName, port, servName)
			if err != nil {
				// Leave the SMB server running but don't update the state
				log.Errorf("Failed to update metadata for %s", volName)
				lock.ReleaseLock()
				return
			}

			// server start succeed. Set desired state on volume.
			stateUpdateResult := e.CompareAndPut(kvstore.VolPrefixState+volName,
				string(kvstore.VolStateReady),
				string(kvstore.VolStateMounted))
			if stateUpdateResult == false {
				log.Errorf("Failed to update state of %s from Ready to Mounted", volName)
			}
		} else {
			// failed to start server
			log.Errorf("Failed to start SMB server for %s", volName)
		}
		lock.ReleaseLock()
	}

	return
}

// etcdStopEventHandler function handles the returned event from etcd watcher of the stop trigger changes
func (e *EtcdKVS) etcdStopEventHandler(ev *etcdClient.Event) {
	log.WithFields(
		log.Fields{"type": ev.Type},
	).Info("Watcher on stop trigger returns event ")

	// monitor PUT requests on stop triggers, excluding the PUT for creation
	if ev.Type == etcdClient.EventTypePut && ev.PrevKv != nil {
		log.Debugf("Watcher got a increase event on stop watcher, new value is %s", string(ev.Kv.Value))
		volName := strings.TrimPrefix(string(ev.Kv.Key), kvstore.VolPrefixStopTrigger)

		// Compare the value of stop marker, only one watcher will be able to successfully update the value to
		// the new value of stop trigger
		success, err := e.CompareAndPutIfNotEqual(kvstore.VolPrefixStopMarker+volName, string(ev.Kv.Value))
		if err != nil || success == false {
			// watchers failed to update the marker just return
			return
		}

		log.Infof("Successfully update the stop marker to the value of stop trigger, proceed to next step")

		// Get the lock for changing the state and server
		lock, err := e.CreateLock(kvstore.VolPrefixState + volName)
		if err != nil {
			log.Errorf("Failed to create lock for state changing of %s", volName)
			return
		}

		err = lock.BlockingLockWithLease()
		if err != nil {
			log.Errorf("Failed to blocking wait lock for state changing of %s", volName)
			lock.ClearLock()
			return
		}

		// stop event should check global refcount instead of state
		ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
		resp, err := e.watcher.Get(ctx, kvstore.VolPrefixGRef+volName)
		cancel()
		if err != nil {
			log.WithFields(
				log.Fields{"volName": volName,
					"error": err},
			).Error("Failed to Get global refcount of volume from ETCD ")
			lock.ReleaseLock()
			return
		}

		if string(resp.Kvs[0].Value) != etcdNoRef {
			log.Infof("Volume %s still has global users, cannot stop the server", volName)
			lock.ReleaseLock()
			return
		}

		// Change state back to Ready before stop the server
		_, err = e.CompareAndPutIfNotEqual(kvstore.VolPrefixState+volName, string(kvstore.VolStateReady))
		if err != nil {
			log.Errorf("Failed to update state of %s to Ready during server stop event", err)
			lock.ReleaseLock()
			return
		}

		// stop the SMB server
		port, servName, succeeded := e.dockerOps.StopSMBServer(volName)
		if succeeded {
			err = e.updateServerInfo(kvstore.VolPrefixInfo+volName, port, servName)
			if err != nil {
				log.Warningf("Failed to update metadata for %s after stopping the server", volName)
			}
		} else {
			log.Errorf("Failed to stop SMB server for %s", volName)
		}
		lock.ReleaseLock()
		return
	}
}

func (e *EtcdKVS) updateServerInfo(key string, port int, servName string) error {
	// Update volume metadata to reflect port number and file service name.
	var entries []kvstore.KvPair
	var writeEntries []kvstore.KvPair
	var volRecord VFileVolConnectivityData

	// Port, Server name, Samba username/password are in the same key.
	// Must fetch this key to know the value of other fields before rewriting them.
	keys := []string{key}
	entries, err := e.ReadMetaData(keys)
	if err != nil {
		// Failed to fetch existing metadata on the volume
		log.Warningf("Failed to read volume metadata before updating: %v", err)
		return err
	}
	err = json.Unmarshal([]byte(entries[0].Value), &volRecord)
	if err != nil {
		// Failed to unmarshal record from JSON
		log.Warningf("Failed to unmarshal JSON for reading existing metadata: %v", err)
		return err
	}

	// Check if current port number is already recorded in the meta data
	if volRecord.Port != port {
		// Rewrite the port number and service name
		// then marshal the data structure to JSON again.
		volRecord.Port = port
		volRecord.ServiceName = servName
		byteRecord, err := json.Marshal(volRecord)
		if err != nil {
			// Failed to marshal record as JSON
			log.Warningf("Failed to marshal JSON for writing metadata: %v", err)
			return err
		}
		writeEntries = append(writeEntries, kvstore.KvPair{
			Key:   key,
			Value: string(byteRecord)})

		log.Infof("Updating port and file service name for %s", key)
		err = e.WriteMetaData(writeEntries)
		if err != nil {
			// Failed to write metadata.
			log.Warningf("Failed to write metadata for volume %s", key)
			return err
		}
	} else {
		log.Warningf("Volume metadata already contains the correct port number, skip updating metadata")
	}
	return nil
}

// CreateLock: create a ETCD lock for a given key, only setup the client
// session and mutex should be create when start to use the lock
func (e *EtcdKVS) CreateLock(name string) (kvstore.KvLock, error) {
	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	if etcdAPI == nil {
		log.Warningf(etcdClientCreateError)
		return nil, errors.New(etcdClientCreateError)
	}

	var kvlock kvstore.KvLock
	elock := &EtcdLock{
		Key:     name + "-lock",
		lockCli: etcdAPI,
	}
	kvlock = elock
	return kvlock, nil
}

// TryLock: try to get a ETCD mutex lock
func (e *EtcdLock) TryLock() error {
	session, err := concurrency.NewSession(e.lockCli, concurrency.WithTTL(etcdLockLeaseTime))
	if err != nil {
		log.Errorf("Failed to create session before blocking wait lock of %s", e.Key)
		return err
	}
	mutex := concurrency.NewMutex(session, e.Key)
	err = mutex.Lock(context.TODO())
	if err != nil {
		log.Errorf("Failed to get TryLock for key %s", e.Key)
		session.Close()
		return err
	} else {
		log.Infof("TryLock %s successfully", e.Key)
		e.lockSession = session
		e.lockMutex = mutex
		return nil
	}
}

// BlockingLockWithLease: blocking wait to get a ETCD mutex on the given name until timeout
func (e *EtcdLock) BlockingLockWithLease() error {
	log.Debugf("BlockingLockWithLease: key=%s", e.Key)

	session, err := concurrency.NewSession(e.lockCli, concurrency.WithTTL(etcdLockLeaseTime))
	if err != nil {
		log.Errorf("Failed to create session before blocking wait lock of %s", e.Key)
		return err
	}

	ticker := time.NewTicker(etcdLockTicker)
	defer ticker.Stop()
	timer := time.NewTimer(dockerops.GetServiceStartTimeout() + etcdLockTimer)
	defer timer.Stop()

	mutex := concurrency.NewMutex(session, e.Key)

	for {
		select {
		case <-ticker.C:
			err := mutex.Lock(context.TODO())
			if err != nil {
				log.Warningf("Failed to get lock for key %s", e.Key)
			} else {
				log.Infof("Locked %s successfully", e.Key)
				e.lockSession = session
				e.lockMutex = mutex
				return nil
			}
		case <-timer.C:
			msg := fmt.Sprintf(etcdLockTimeoutErrMsg)
			log.Warningf(msg)
			session.Close()
			return errors.New(msg)
		}
	}
}

// ClearLock: stop the client/session with the lock
func (e *EtcdLock) ClearLock() {
	e.lockMutex = nil
	if e.lockSession != nil {
		e.lockSession.Close()
		e.lockSession = nil
	}
	if e.lockCli != nil {
		e.lockCli.Close()
		e.lockCli = nil
	}
}

// ReleaseLock: try to release a lock
func (e *EtcdLock) ReleaseLock() {
	err := e.lockMutex.Unlock(context.TODO())
	if err != nil {
		log.Warningf("Failed to release lock for %s, but will continue to clear the lock session", e.Key)
	}
	e.ClearLock()
	return
}

// CompareAndPut function: compare the value of the kay with oldVal
// if equal, replace with newVal and return true; or else, return false.
func (e *EtcdKVS) CompareAndPut(key string, oldVal string, newVal string) bool {
	log.Debugf("CompareAndPut: key=%s oldVal=%s newVal=%s", key, oldVal, newVal)
	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	if etcdAPI == nil {
		log.Warningf(etcdClientCreateError)
		return false
	}
	defer etcdAPI.Close()

	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	txresp, err := etcdAPI.Txn(ctx).If(
		etcdClient.Compare(etcdClient.Value(key), "=", oldVal),
	).Then(
		etcdClient.OpPut(key, newVal),
	).Commit()
	cancel()

	if err != nil {
		log.WithFields(
			log.Fields{"Key": key,
				"Value to compare": oldVal,
				"Value to replace": newVal,
				"Error":            err},
		).Errorf("Failed to compare and put ")
		return false
	}

	return txresp.Succeeded
}

// CompareAndPutIfNotEqual - Compare and put a new value of the key if the current value is not equal to the new value
func (e *EtcdKVS) CompareAndPutIfNotEqual(key string, newVal string) (bool, error) {
	log.Debugf("CompareAndPutIfNotEqual: key=%s, newVal=%s", key, newVal)
	var txresp *etcdClient.TxnResponse
	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	if etcdAPI == nil {
		return false, errors.New(etcdClientCreateError)
	}
	defer etcdAPI.Close()

	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	txresp, err := etcdAPI.Txn(ctx).If(
		etcdClient.Compare(etcdClient.Value(key), "!=", newVal),
	).Then(
		etcdClient.OpPut(key, newVal),
	).Commit()
	cancel()

	if err != nil {
		log.WithFields(
			log.Fields{"Key": key,
				"Value to compare": newVal,
				"Error":            err},
		).Errorf("Failed to compare and put if not equal")
	}
	return txresp.Succeeded, err
}

//CompareAndPutOrFetch - Compare and put or get the current value of the key
func (e *EtcdKVS) CompareAndPutOrFetch(key string,
	oldVal string,
	newVal string) (*etcdClient.TxnResponse, error) {

	log.Debugf("CompareAndPutOrFetch: key=%s oldVal=%s newVal=%s", key, oldVal, newVal)
	var txresp *etcdClient.TxnResponse
	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	if etcdAPI == nil {
		return txresp, errors.New(etcdClientCreateError)
	}
	defer etcdAPI.Close()

	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	txresp, err := etcdAPI.Txn(ctx).If(
		etcdClient.Compare(etcdClient.Value(key), "=", oldVal),
	).Then(
		etcdClient.OpPut(key, newVal),
	).Else(
		etcdClient.OpGet(key),
	).Commit()
	cancel()

	if err != nil {
		// There was some error
		log.WithFields(
			log.Fields{"Key": key,
				"Value to compare": oldVal,
				"Value to replace": newVal,
				"Error":            err},
		).Errorf("Failed to compare and put ")
	}
	return txresp, err
}

// CompareAndPutStateOrBusywait function: compare the volume state with oldVal
// if equal, replace with newVal and return true; or else, return false;
// waits if volume is in a state from where it can reach the ready state
func (e *EtcdKVS) CompareAndPutStateOrBusywait(key string, oldVal string, newVal string) bool {
	var txresp *etcdClient.TxnResponse
	var err error
	log.Debugf("CompareAndPutStateOrBusywait: key=%s oldVal=%s newVal=%s", key, oldVal, newVal)
	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	timer := time.NewTimer(etcdUpdateTimeout)
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			// Retry
			log.Infof("Attempting to change volume state to %s", newVal)
			txresp, err = e.CompareAndPutOrFetch(key, oldVal, newVal)
			if err != nil {
				return false
			}
			if txresp.Succeeded == false {
				resp := txresp.Responses[0].GetResponseRange()
				// Did we encounter states other than Unmounting or Creating?
				if (string(resp.Kvs[0].Value) != string(kvstore.VolStateUnmounting)) &&
					(string(resp.Kvs[0].Value) != string(kvstore.VolStateCreating)) {
					log.Infof("Volume not in proper state for the operation: %s",
						string(resp.Kvs[0].Value))
					return false
				}
			} else {
				return true
			}
		case <-timer.C:
			// Time out
			log.Warningf("Operation to change state from %s to %s timed out!",
				oldVal, newVal)
			return false
		}
	}
}

// createEtcdClient function creates an ETCD client according to swarm manager info
func (e *EtcdKVS) createEtcdClient() *etcdClient.Client {
	managers, err := e.dockerOps.GetSwarmManagers()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to get swarm managers ")
		return nil
	}

	for _, manager := range managers {
		etcd, err := e.addrToEtcdClient(manager.Addr)
		if err == nil {
			return etcd
		}
	}

	log.WithFields(
		log.Fields{"Swarm ID": e.nodeID,
			"IP Addr": e.nodeAddr},
	).Error("Failed to create ETCD client according to manager info ")
	return nil
}

// addrToEtcdClient function create a new Etcd client according to the input docker address
// it can be used by swarm worker to get a Etcd client on swarm manager
// or it can be used by swarm manager to get a Etcd client on swarm leader
func (e *EtcdKVS) addrToEtcdClient(addr string) (*etcdClient.Client, error) {
	// input address are RemoteManagers from docker info or ManagerStatus.Addr from docker inspect
	// in the format of [host]:[docker manager port]
	etcdClientPort := e.etcdClientPort
	s := strings.Split(addr, ":")
	endpoint := s[0] + etcdClientPort
	cfg := etcdClient.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: etcdRequestTimeout,
	}

	etcd, err := etcdClient.New(cfg)
	if err != nil {
		log.Debugf("Cannot get etcdClient for addr %s", addr)
		return nil, err
	}

	return etcd, nil
}

// List function lists all the different portion of keys with the given prefix
func (e *EtcdKVS) List(prefix string) ([]string, error) {
	var keys []string

	client := e.createEtcdClient()
	if client == nil {
		return keys, fmt.Errorf(etcdClientCreateError)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	resp, err := client.Get(ctx, prefix, etcdClient.WithPrefix(),
		etcdClient.WithSort(etcdClient.SortByKey, etcdClient.SortDescend))
	cancel()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err,
				"prefix": prefix},
		).Error("Failed to call ETCD Get for listing all keys with prefix ")
		return nil, err
	}

	for _, ev := range resp.Kvs {
		keys = append(keys, strings.TrimPrefix(string(ev.Key), prefix))
	}

	return keys, nil
}

// KvMapFromPrefix -  Create key-value pairs according to a given prefix
func (e *EtcdKVS) KvMapFromPrefix(prefix string) (map[string]string, error) {
	m := make(map[string]string)

	client := e.createEtcdClient()
	if client == nil {
		return m, errors.New(etcdClientCreateError)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	resp, err := client.Get(ctx, prefix, etcdClient.WithPrefix(),
		etcdClient.WithSort(etcdClient.SortByKey, etcdClient.SortDescend))
	cancel()
	if err != nil {
		return m, err
	}

	for _, ev := range resp.Kvs {
		m[string(ev.Key)] = string(ev.Value)
	}

	return m, nil
}

// WriteMetaData - Update or Create metadata in KV store
func (e *EtcdKVS) WriteMetaData(entries []kvstore.KvPair) error {

	var ops []etcdClient.Op
	var msg string
	var err error

	log.WithFields(
		log.Fields{"KvPair": entries},
	).Debug("WriteMetaData")

	// Create a client to talk to etcd
	client := e.createEtcdClient()
	if client == nil {
		return errors.New(etcdClientCreateError)
	}
	defer client.Close()

	// ops contain multiple operations that will be done to etcd
	// in a single revision
	for _, elem := range entries {
		ops = append(ops, etcdClient.OpPut(elem.Key, elem.Value))
	}

	// Lets write the metadata in a single transaction
	// Use a transaction if more than one entries are to be written
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	if len(entries) > 1 {
		_, err = client.Txn(ctx).Then(ops...).Commit()
	} else {
		_, err = client.Do(ctx, ops[0])
	}
	cancel()

	if err != nil {
		msg = fmt.Sprintf("Failed to write metadata: %v.", err)
		if err == context.DeadlineExceeded {
			msg += fmt.Sprintf(etcdUnhealthyErrorMsg)
		}
		log.Warningf(msg)
		return errors.New(msg)
	}
	return nil
}

// UpdateMetaData - Read/Write/Delete metadata according to given key-value pairs
func (e *EtcdKVS) UpdateMetaData(entries []kvstore.KvPair) ([]kvstore.KvPair, error) {
	var ops []etcdClient.Op
	//var result_entries []kvstore.KvPair
	log.WithFields(
		log.Fields{"KvPair": entries},
	).Debug("UpdateMetaData")

	// Create a client to talk to etcd
	client := e.createEtcdClient()
	if client == nil {
		return nil, errors.New(etcdClientCreateError)
	}
	defer client.Close()

	// Lets build the request which will be executed
	// in a single transaction
	// ops contain multiple operations
	for _, elem := range entries {
		if elem.OpType == kvstore.OpPut {
			ops = append(ops, etcdClient.OpPut(elem.Key, elem.Value))
		} else if elem.OpType == kvstore.OpGet {
			ops = append(ops, etcdClient.OpGet(elem.Key))
		} else if elem.OpType == kvstore.OpDelete {
			ops = append(ops, etcdClient.OpDelete(elem.Key))
		} else {
			msg := fmt.Sprintf("Unknown OpType for UpdateMetaData: %s", elem.OpType)
			log.Errorf(msg)
			return nil, errors.New(msg)
		}
	}

	// Post all requested operations in one transaction
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	resp, err := client.Txn(ctx).Then(ops...).Commit()
	cancel()
	if err != nil {
		msg := fmt.Sprintf("Transactional metadata update failed: %v.", err)
		if err == context.DeadlineExceeded {
			msg += fmt.Sprintf(etcdUnhealthyErrorMsg)
		}
		log.Warningf(msg)
		return nil, errors.New(msg)
	}

	for i, elem := range entries {
		resp := resp.Responses[i].GetResponseRange()
		if elem.OpType == kvstore.OpGet {
			// If any Get() didnt find a key, there wont be
			// an error returned. It will just return an empty resp
			if resp.Count == 0 {
				continue
			}
			entries[i].Value = string(resp.Kvs[0].Value)
		}
	}

	log.WithFields(
		log.Fields{"KvPair": entries},
	).Debug("UpdateMetaData succeeded")
	return entries, nil
}

// ReadMetaData - Read metadata in KV store
func (e *EtcdKVS) ReadMetaData(keys []string) ([]kvstore.KvPair, error) {
	var entries []kvstore.KvPair
	var ops []etcdClient.Op
	var missedCount int

	log.WithFields(
		log.Fields{"key": keys},
	).Debug("ReadMetaData")
	// Create a client to talk to etcd
	client := e.createEtcdClient()
	if client == nil {
		return entries, errors.New(etcdClientCreateError)
	}
	defer client.Close()

	// Lets build the request which will be executed
	// in a single transaction
	// ops contain multiple read operations
	for _, elem := range keys {
		ops = append(ops, etcdClient.OpGet(elem))
	}

	// Read all requested keys in one transaction
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	getresp, err := client.Txn(ctx).Then(ops...).Commit()
	cancel()
	if err != nil {
		msg := fmt.Sprintf("Transactional metadata read failed: %v.", err)
		if err == context.DeadlineExceeded {
			msg += fmt.Sprintf(etcdUnhealthyErrorMsg)
		}
		log.Warningf(msg)
		return entries, errors.New(msg)
	}

	// Check responses and append them in entries[]
	missedCount = 0
	for i, elem := range keys {
		resp := getresp.Responses[i].GetResponseRange()
		// If any Get() didnt find a key, there wont be
		// an error returned. It will just return an empty resp
		// Update the number of misses and carry on
		if resp.Count == 0 {
			missedCount++
			continue
		}
		entry := kvstore.KvPair{Key: elem, Value: string(resp.Kvs[0].Value)}
		entries = append(entries, entry)
	}

	if missedCount == len(keys) {
		// Volume does not exist
		return nil, errors.New(kvstore.VolumeDoesNotExistError)
	} else if missedCount > 0 {
		// This should not happen
		// There is a volume but we couldn't read all its keys
		msg := fmt.Sprintf("Failed to get volume. Couldn't find all keys!")
		log.Warningf(msg)
		panic(msg)
	}
	log.WithFields(
		log.Fields{"KvPair": entries},
	).Debug("ReadMetaData succeeded")
	return entries, nil
}

// DeleteMetaData - Delete volume metadata in KV store
func (e *EtcdKVS) DeleteMetaData(name string) error {

	var msg string
	var err error
	log.Debugf("DeleteMetaData: name=%s", name)
	// Create a client to talk to etcd
	client := e.createEtcdClient()
	if client == nil {
		return errors.New(etcdClientCreateError)
	}
	defer client.Close()

	// ops hold multiple operations that will be done to etcd
	// in a single revision. Add all keys for this volname.
	ops := []etcdClient.Op{
		etcdClient.OpDelete(kvstore.VolPrefixState + name),
		etcdClient.OpDelete(kvstore.VolPrefixGRef + name),
		etcdClient.OpDelete(kvstore.VolPrefixStartTrigger + name),
		etcdClient.OpDelete(kvstore.VolPrefixStopTrigger + name),
		etcdClient.OpDelete(kvstore.VolPrefixStartMarker + name),
		etcdClient.OpDelete(kvstore.VolPrefixStopMarker + name),
		etcdClient.OpDelete(kvstore.VolPrefixInfo + name),
	}

	// Delete the metadata in a single transaction
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	_, err = client.Txn(ctx).Then(ops...).Commit()
	cancel()
	if err != nil {
		msg = fmt.Sprintf("Failed to delete metadata for volume %s: %v", name, err)
		if err == context.DeadlineExceeded {
			msg += fmt.Sprintf(etcdUnhealthyErrorMsg)
		}
		log.Warningf(msg)
		return errors.New(msg)
	}
	return nil
}

// DeleteClientMetaData - Delete volume client metadata in KV store
func (e *EtcdKVS) DeleteClientMetaData(name string, nodeID string) error {

	var msg string
	var err error

	log.Debugf("DeleteClientMetaData: name=%s nodeID=%s", name, nodeID)
	// Create a client to talk to etcd
	client := e.createEtcdClient()
	if client == nil {
		return errors.New(etcdClientCreateError)
	}
	defer client.Close()

	// ops hold multiple operations that will be done to etcd
	// in a single revision. Add all keys for this volname.
	ops := []etcdClient.Op{
		etcdClient.OpDelete(kvstore.VolPrefixClient + name + "_" + nodeID),
	}

	// Delete the metadata in a single transaction
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	_, err = client.Txn(ctx).Then(ops...).Commit()
	cancel()
	if err != nil {
		msg = fmt.Sprintf("Failed to delete client metadata for volume %s on node %s: %v", name, nodeID, err)
		if err == context.DeadlineExceeded {
			msg += fmt.Sprintf(etcdUnhealthyErrorMsg)
		}
		log.Warningf(msg)
		return errors.New(msg)
	}
	return nil
}

// AtomicIncr - Increase a key value by 1
func (e *EtcdKVS) AtomicIncr(key string) error {
	// Create a client to talk to etcd
	client := e.createEtcdClient()
	if client == nil {
		return fmt.Errorf(etcdClientCreateError)
	}
	defer client.Close()

	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	timer := time.NewTimer(etcdUpdateTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
			resp, err := client.Get(ctx, key)
			cancel()
			if err != nil {
				log.WithFields(
					log.Fields{"key": key,
						"error": err},
				).Error("Failed to Get key-value from ETCD ")
				return err
			}

			if len(resp.Kvs) == 0 {
				return fmt.Errorf("AtomicIncr: no key found for %s", key)
			}

			oldVal := string(resp.Kvs[0].Value)
			num, _ := strconv.Atoi(oldVal)
			num++
			newVal := strconv.Itoa(num)
			if e.CompareAndPut(key, oldVal, newVal) {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("Timeout reached; AtomicIncr is not complete")
		}
	}
}

// AtomicDecr - Decrease a key value by 1
func (e *EtcdKVS) AtomicDecr(key string) error {
	// Create a client to talk to etcd
	client := e.createEtcdClient()
	if client == nil {
		return fmt.Errorf(etcdClientCreateError)
	}
	defer client.Close()

	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	timer := time.NewTimer(etcdUpdateTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
			resp, err := client.Get(ctx, key)
			cancel()
			if err != nil {
				log.WithFields(
					log.Fields{"key": key,
						"error": err},
				).Error("Failed to Get key-value from ETCD ")
				return err
			}

			if len(resp.Kvs) == 0 {
				return fmt.Errorf("AtomicIncr: no key found for %s", key)
			}

			oldVal := string(resp.Kvs[0].Value)
			num, _ := strconv.Atoi(oldVal)
			if num == 0 {
				return fmt.Errorf("Cannot decrease a value equal to 0")
			}
			num--
			newVal := strconv.Itoa(num)
			if e.CompareAndPut(key, oldVal, newVal) {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("Timeout reached; AtomicDecr is not complete")
		}
	}
}

// BlockingWaitAndGet - Blocking wait until a key value becomes equal to a specific value
// then read the value of another key
func (e *EtcdKVS) BlockingWaitAndGet(key string, value string, newKey string) (string, error) {
	// Create a client to talk to etcd
	log.Debugf("BlockingWaitAndGet: key=%s value=%s newKey=%s", key, value, newKey)
	client := e.createEtcdClient()
	if client == nil {
		return "", fmt.Errorf(etcdClientCreateError)
	}
	defer client.Close()

	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	// This call is used to block and wait for long
	// running functions. Larger timeout is justified.
	timer := time.NewTimer(dockerops.GetServiceStartTimeout() + etcdUpdateTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
			txresp, err := client.Txn(ctx).If(
				etcdClient.Compare(etcdClient.Value(key), "=", value),
			).Then(
				etcdClient.OpGet(newKey),
			).Commit()
			cancel()

			if err != nil {
				log.WithFields(
					log.Fields{"key": key,
						"value":   value,
						"new key": newKey,
						"error":   err},
				).Error("Failed to compare and get from ETCD ")
				return "", err
			}

			if txresp.Succeeded {
				resp := txresp.Responses[0].GetResponseRange()
				if len(resp.Kvs) == 0 {
					return "", fmt.Errorf("BlockingWaitAndGet: no key found for %s", newKey)
				}

				return string(resp.Kvs[0].Value), nil
			}
		case <-timer.C:
			return "", fmt.Errorf("Timeout reached; BlockingWait is not complete")
		}
	}
}
