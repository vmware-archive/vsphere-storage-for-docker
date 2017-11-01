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
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	etcdClient "github.com/coreos/etcd/clientv3"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/vfile/dockerops"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/vfile/kvstore"
)

/*
   etcdClientPort:             port for etcd clients to talk to the peers
   etcdPeerPort:               port for etcd peers talk to each other
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
   swarmUnhealthyErrorMsg:     Message indicating swarm cluster is unhealthy
   etcdSingleRef:              if global refcount 0 -> 1, start SMB server
   etcdNoRef:                  if global refcount 1 -> 0, shut down SMB server
*/
const (
	etcdDataDir              = "/etcd-data"
	etcdClientPort           = ":2379"
	etcdPeerPort             = ":2380"
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
	swarmUnhealthyErrorMsg   = "Swarm cluster maybe unhealthy"
	etcdSingleRef            = "1"
	etcdNoRef                = "0"
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

	e = &EtcdKVS{
		dockerOps: dockerOps,
		nodeID:    nodeID,
		nodeAddr:  addr,
		isManager: isManager,
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
	} else {
		// ETCD data-dir already exists, just re-join
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

// rejoinEtcdCluster function is called when a node need to rejoin a ETCD cluster
func (e *EtcdKVS) rejoinEtcdCluster() error {
	nodeID := e.nodeID
	nodeAddr := e.nodeAddr
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
	log.Infof("startEtcdCluster on node with nodeID %s and nodeAddr %s", nodeID, nodeAddr)

	// create ETCD data directory with 755 permission since only root needs full permission
	err := os.Mkdir(etcdDataDir, 0755)
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

	etcd, err := addrToEtcdClient(leaderAddr)
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
	etcd, err := addrToEtcdClient(e.nodeAddr)
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
			cli, err := addrToEtcdClient(e.nodeAddr)
			if err != nil {
				log.WithFields(
					log.Fields{"nodeAddr": e.nodeAddr,
						"error": err},
				).Warningf("Failed to get ETCD client, retry before timeout ")
			} else {
				log.Infof("Local ETCD client is up successfully, start watcher")
				e.watcher = cli
				go e.etcdWatcher(cli)
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("Timeout reached; ETCD cluster is not started")
		}
	}
}

// etcdWatcher function sets up a watcher to monitor all the changes to global refcounts in the KV store
func (e *EtcdKVS) etcdWatcher(cli *etcdClient.Client) {
	watchCh := cli.Watch(context.Background(), kvstore.VolPrefixGRef,
		etcdClient.WithPrefix(), etcdClient.WithPrevKV())
	for wresp := range watchCh {
		for _, ev := range wresp.Events {
			e.etcdEventHandler(ev)
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

// etcdEventHandler function handles the returned event from etcd watcher of global refcount changes
func (e *EtcdKVS) etcdEventHandler(ev *etcdClient.Event) {
	log.WithFields(
		log.Fields{"type": ev.Type},
	).Infof("Watcher on global refcount returns event ")

	nested := func(key string, fromState kvstore.VolStatus,
		toState kvstore.VolStatus, interimState kvstore.VolStatus,
		fn func(string) (int, string, bool)) {

		// watcher observes global refcount critical change
		// transactional edit state first
		volName := strings.TrimPrefix(key, kvstore.VolPrefixGRef)
		succeeded := e.CompareAndPutStateOrBusywait(kvstore.VolPrefixState+volName,
			string(fromState), string(interimState))
		if !succeeded {
			// this handler doesn't get the right to start/stop server
			return
		}

		port, servName, succeeded := fn(volName)
		if succeeded {
			// Either starting or stopping SMB
			// server succeeded.
			// Update volume metadata to reflect
			// port number and file service name.
			var entries []kvstore.KvPair
			var writeEntries []kvstore.KvPair
			var volRecord VFileVolConnectivityData

			// Port, Server name, Client list, Samba
			// username/password are in the same key.
			// Must fetch this key to know the value
			// of other fields before rewriting them.
			keys := []string{
				kvstore.VolPrefixInfo + volName,
			}
			entries, err := e.ReadMetaData(keys)
			if err != nil {
				// Failed to fetch existing metadata on the volume
				// Set volume state to error as we cannot
				// proceed
				log.Warningf("Failed to read volume metadata before updating port information: %v",
					err)
				e.CompareAndPut(kvstore.VolPrefixState+volName,
					string(interimState),
					string(kvstore.VolStateError))
				return
			}
			err = json.Unmarshal([]byte(entries[0].Value), &volRecord)
			if err != nil {
				// Failed to unmarshal record from JSON
				// Set volume state to error as we cannot
				// proceed
				log.Warningf("Failed to unmarshal JSON for reading existing metadata: %v",
					err)
				e.CompareAndPut(kvstore.VolPrefixState+volName,
					string(interimState),
					string(kvstore.VolStateError))
				return
			}
			// Rewrite the port number and service name
			// then marshal the data structure to JSON again.
			volRecord.Port = port
			volRecord.ServiceName = servName
			byteRecord, err := json.Marshal(volRecord)
			if err != nil {
				// Failed to marshal record as JSON
				// Set volume state to error as we cannot
				// proceed
				log.Warningf("Failed to marshal JSON for writing metadata: %v",
					err)
				e.CompareAndPut(kvstore.VolPrefixState+volName,
					string(interimState),
					string(kvstore.VolStateError))
				return
			}
			writeEntries = append(writeEntries, kvstore.KvPair{
				Key:   kvstore.VolPrefixInfo + volName,
				Value: string(byteRecord)})

			log.Infof("Updating port and file service name for %s", volName)
			err = e.WriteMetaData(writeEntries)
			if err != nil {
				// Failed to write metadata.
				// Set volume state to error as we cannot
				// proceed
				log.Warningf("Failed to write metadata for volume %s",
					volName)
				e.CompareAndPut(kvstore.VolPrefixState+volName,
					string(interimState),
					string(kvstore.VolStateError))
				return
			}

			// server start/stop succeed. Set desired state on volume.
			stateUpdateResult := e.CompareAndPut(kvstore.VolPrefixState+volName,
				string(interimState),
				string(toState))
			if stateUpdateResult == false {
				// Could not set desired state on volume
				// set to state Error
				e.CompareAndPut(kvstore.VolPrefixState+volName,
					string(interimState),
					string(kvstore.VolStateError))
			}
		} else {
			// failed to start/stop server, set to state Error
			e.CompareAndPut(kvstore.VolPrefixState+volName,
				string(interimState),
				string(kvstore.VolStateError))
		}
		return
	}

	// What we want to monitor are PUT requests on global refcount
	// Not delete, not get, not anything else
	if ev.Type == etcdClient.EventTypePut {
		if string(ev.Kv.Value) == etcdSingleRef &&
			ev.PrevKv != nil &&
			string(ev.PrevKv.Value) == etcdNoRef {
			// Refcount went 0 -> 1
			nested(string(ev.Kv.Key), kvstore.VolStateReady,
				kvstore.VolStateMounted, kvstore.VolStateMounting,
				e.dockerOps.StartSMBServer)
		} else if string(ev.Kv.Value) == etcdNoRef &&
			ev.PrevKv != nil &&
			string(ev.PrevKv.Value) == etcdSingleRef {
			// Refcount went 1 -> 0
			nested(string(ev.Kv.Key), kvstore.VolStateMounted,
				kvstore.VolStateReady, kvstore.VolStateUnmounting,
				e.dockerOps.StopSMBServer)
		}
	}
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

//CompareAndPutOrFetch - Compare and put of get the current value of the key
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
		etcd, err := addrToEtcdClient(manager.Addr)
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
func addrToEtcdClient(addr string) (*etcdClient.Client, error) {
	// input address are RemoteManagers from docker info or ManagerStatus.Addr from docker inspect
	// in the format of [host]:[docker manager port]
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
			msg += fmt.Sprintf(swarmUnhealthyErrorMsg)
		}
		log.Warningf(msg)
		return errors.New(msg)
	}
	return nil
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
			msg += fmt.Sprintf(swarmUnhealthyErrorMsg)
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
		etcdClient.OpDelete(kvstore.VolPrefixInfo + name),
	}

	// Delete the metadata in a single transaction
	ctx, cancel := context.WithTimeout(context.Background(), etcdRequestTimeout)
	_, err = client.Txn(ctx).Then(ops...).Commit()
	cancel()
	if err != nil {
		msg = fmt.Sprintf("Failed to delete metadata for volume %s: %v", name, err)
		if err == context.DeadlineExceeded {
			msg += fmt.Sprintf(swarmUnhealthyErrorMsg)
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
			msg += fmt.Sprintf(swarmUnhealthyErrorMsg)
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
