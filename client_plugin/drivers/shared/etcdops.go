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

package shared

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	etcdClient "github.com/coreos/etcd/clientv3"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/swarm"
)

/*
   etcdClientPort:             On which port do etcd clients talk to
                               the peers?
   etcdPeerPort:               On which port do etcd peers talk to
                               each other?
   etcdClusterToken:           ID of the cluster to create/join
   etcdListenURL:              On which interface is etcd listening?
   etcdScheme:                 Protocol used for communication
   etcdClusterStateNew:        Used to indicate the formation of a new
                               cluster
   etcdClusterStateExisting:   Used to indicate that this node is joining
                               an existing etcd cluster
   etcdPrefixState:            Each volume has three metadata keys. Each such
                               key terminates in the name of the volume, but
                               is preceded by a prefix. This is the prefix
                               for "State" key
   etcdPrefixGRef:             The prefix for GRef key (Global refcount)
   etcdPrefixInfo:             The prefix for info key. This key holds all
                               other metadata fields squashed into one
   requestTimeout:             After how long should an etcd request timeout
   etcdClientCreateError:      Error indicating failure to create etcd client
   VolumeDoesNotExistError:    Error indicating that there is no such volume
   etcdSingleRef:              if global refcount 0 -> 1, start SMB server
   etcdNoRef:                  if global refcount 1 -> 0, shut down SMB server
*/
const (
	etcdClientPort           = ":2379"
	etcdPeerPort             = ":2380"
	etcdClusterToken         = "vsphere-shared-etcd-cluster"
	etcdListenURL            = "0.0.0.0"
	etcdScheme               = "http://"
	etcdClusterStateNew      = "new"
	etcdClusterStateExisting = "existing"
	etcdPrefixState          = "SVOLS_stat_"
	etcdPrefixGRef           = "SVOLS_gref_"
	etcdPrefixInfo           = "SVOLS_info_"
	requestTimeout           = 5 * time.Second
	checkSleepDuration       = time.Second
	etcdClientCreateError    = "Failed to create etcd client"
	VolumeDoesNotExistError  = "No such volume"
	etcdSingleRef            = "1"
	etcdNoRef                = "0"
)

// kvPair : Key Value pair holder
type kvPair struct {
	key   string
	value string
}

type etcdKVS struct {
	driver   *VolumeDriver
	nodeID   string
	nodeAddr string
	client   *etcdClient.Client
}

// NewKvStore function: start or join ETCD cluster depending on the role of the node
func NewKvStore(driver *VolumeDriver) *etcdKVS {
	var e *etcdKVS

	ctx := context.Background()
	dclient := driver.dockerd

	// get NodeID from docker client
	info, err := dclient.Info(ctx)
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to get Info from docker client ")
		return nil
	}

	// get the swarmID and IP address of current node
	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive {
		log.WithFields(
			log.Fields{"LocalNodeState": string(info.Swarm.LocalNodeState)},
		).Errorf("Swarm node state is not active ")
		return nil
	}

	nodeID := info.Swarm.NodeID
	addr := info.Swarm.NodeAddr

	e = &etcdKVS{
		driver:   driver,
		nodeID:   nodeID,
		nodeAddr: addr,
	}

	// worker just returns
	if info.Swarm.ControlAvailable == false {
		log.WithFields(
			log.Fields{"nodeID": nodeID},
		).Info("Swarm node role: worker. No further action needed, return from NewKvStore ")
		return e
	}

	// check my local role
	node, _, err := dclient.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		log.WithFields(log.Fields{"nodeID": nodeID,
			"error": err}).Error("Failed to inspect node ")
		return nil
	}

	// if leader, proceed to start ETCD cluster
	if node.ManagerStatus.Leader {
		log.WithFields(
			log.Fields{"nodeID": nodeID},
		).Info("Swarm node role: leader, start ETCD cluster")
		err = e.startEtcdCluster()
		if err != nil {
			log.WithFields(log.Fields{"nodeID": nodeID,
				"error": err}).Error("Failed to start ETCD Cluster")
			return nil
		}
		return e
	}

	// if manager, first find out who's leader, then proceed to join ETCD cluster
	nodes, err := dclient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		log.WithFields(log.Fields{"nodeID": nodeID,
			"error": err}).Error("Failed to get NodeList from swarm manager")
		return nil
	}
	for _, n := range nodes {
		if n.ManagerStatus != nil && n.ManagerStatus.Leader == true {
			log.WithFields(
				log.Fields{"leader ID": n.ID,
					"manager ID": nodeID},
			).Info("Swarm node role: manager. Action: find leader ")

			err = e.joinEtcdCluster(n.ManagerStatus.Addr)
			if err != nil {
				log.WithFields(log.Fields{"nodeID": nodeID,
					"error": err}).Error("Failed to join ETCD Cluster")
				return nil
			}
			return e
		}
	}

	log.Errorf("Failed to get leader for swarm manager %s", nodeID)
	return nil
}

// startEtcdCluster function is called by swarm leader to start a ETCD cluster
func (e *etcdKVS) startEtcdCluster() error {
	nodeID := e.nodeID
	nodeAddr := e.nodeAddr
	lines := []string{
		"--name", nodeID,
		"--advertise-client-urls", etcdScheme + nodeAddr + etcdClientPort,
		"--initial-advertise-peer-urls", etcdScheme + nodeAddr + etcdPeerPort,
		"--listen-client-urls", etcdScheme + etcdListenURL + etcdClientPort,
		"--listen-peer-urls", etcdScheme + etcdListenURL + etcdPeerPort,
		"--initial-cluster-token", etcdClusterToken,
		"--initial-cluster", nodeID + "=" + etcdScheme + nodeAddr + etcdPeerPort,
		"--initial-cluster-state", etcdClusterStateNew,
	}

	// start the routine to create an etcd cluster
	go etcdService(lines)

	// check if etcd cluster is successfully started, then start the watcher
	return e.checkLocalEtcd()
}

// joinEtcdCluster function is called by a non-leader swarm manager to join a ETCD cluster
func (e *etcdKVS) joinEtcdCluster(leaderAddr string) error {
	nodeAddr := e.nodeAddr
	nodeID := e.nodeID

	etcd, err := addrToEtcdClient(leaderAddr)
	if err != nil {
		log.WithFields(
			log.Fields{"nodeAddr": nodeAddr,
				"leaderAddr": leaderAddr,
				"nodeID":     nodeID},
		).Error("Failed to join ETCD cluster on manager ")
	}

	// list all current ETCD members, check if this node is already added as a member
	lresp, err := etcd.MemberList(context.Background())
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

				_, err = etcd.MemberRemove(context.Background(), member.ID)
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
		aresp, err := etcd.MemberAdd(context.Background(), peerAddrs)
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

	lines := []string{
		"--name", nodeID,
		"--advertise-client-urls", etcdScheme + nodeAddr + etcdClientPort,
		"--initial-advertise-peer-urls", etcdScheme + nodeAddr + etcdPeerPort,
		"--listen-client-urls", etcdScheme + etcdListenURL + etcdClientPort,
		"--listen-peer-urls", etcdScheme + etcdListenURL + etcdPeerPort,
		"--initial-cluster-token", etcdClusterToken,
		"--initial-cluster", initCluster + nodeID + "=" + etcdScheme + nodeAddr + etcdPeerPort,
		"--initial-cluster-state", etcdClusterStateExisting,
	}

	// start the routine for joining an etcd cluster
	go etcdService(lines)

	// check if successfully joined the etcd cluster, then start the watcher
	return e.checkLocalEtcd()
}

// etcdService function starts a routine of etcd
func etcdService(cmd []string) {
	_, err := exec.Command("/bin/etcd", cmd...).Output()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err, "cmd": cmd},
		).Error("Failed to start ETCD command ")
	}
}

// checkLocalEtcd function check if local ETCD endpoint is successfully started or not
// if yes, start the watcher for volume global refcount
func (e *etcdKVS) checkLocalEtcd() error {
	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	timer := time.NewTimer(requestTimeout)
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
				e.client = cli
				go e.etcdWatcher()
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("Timeout reached; ETCD cluster is not started")
		}
	}
}

// etcdWatcher function sets up a watcher to monitor all the changes to global refcounts in the KV store
func (e *etcdKVS) etcdWatcher() {
	// TODO: when the manager is demoted to worker, the watcher should be cancelled
	watchCh := e.client.Watch(context.Background(), etcdPrefixGRef,
		etcdClient.WithPrefix(), etcdClient.WithPrevKV())
	for wresp := range watchCh {
		for _, ev := range wresp.Events {
			e.etcdEventHandler(ev)
		}
	}
}

// etcdEventHandler function handles the returned event from etcd watcher of global refcount changes
func (e *etcdKVS) etcdEventHandler(ev *etcdClient.Event) {
	log.WithFields(
		log.Fields{"type": ev.Type},
	).Infof("Watcher on global refcount returns event ")

	nested := func(key string, fromState volStatus, toState volStatus, fn func(string) bool) {
		// watcher observes global refcount critical change
		// transactional edit state first
		volName := strings.TrimPrefix(key, etcdPrefixGRef)
		succeeded := e.CompareAndPut(etcdPrefixState+volName,
			string(fromState), string(volStateIntermediate))
		if !succeeded {
			// this handler doesn't get the right to start server
			return
		}

		if fn(volName) {
			// server start/stop succeed, set desired state
			if e.CompareAndPut(etcdPrefixState+volName,
				string(volStateIntermediate),
				string(toState)) == false {
				// Failed to set state to desired state
				// set to state Error
				e.CompareAndPut(etcdPrefixState+volName,
					string(volStateIntermediate),
					string(volStateError))
			}
		} else {
			// failed to start/stop server, set to state Error
			e.CompareAndPut(etcdPrefixState+volName,
				string(volStateIntermediate),
				string(volStateError))
		}
		return
	}

	if ev.Type == etcdClient.EventTypePut {
		if string(ev.Kv.Value) == etcdSingleRef &&
			ev.PrevKv != nil &&
			string(ev.PrevKv.Value) == etcdNoRef {
			nested(string(ev.Kv.Key), volStateReady, volStateMounted, e.driver.startSMBServer)
			return
		}

		if string(ev.Kv.Value) == etcdNoRef &&
			ev.PrevKv != nil &&
			string(ev.PrevKv.Value) == etcdSingleRef {
			nested(string(ev.Kv.Key), volStateMounted, volStateReady, e.driver.stopSMBServer)
		}
	}

	return
}

// CompareAndPut function: compare the value of the kay with oldVal
// if equal, replace with newVal and return true; or else, return false.
func (e *etcdKVS) CompareAndPut(key string, oldVal string, newVal string) bool {
	txresp, err := e.client.Txn(context.TODO()).If(
		etcdClient.Compare(etcdClient.Value(key), "=", oldVal),
	).Then(
		etcdClient.OpPut(key, newVal),
	).Commit()

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

func (e *etcdKVS) createEtcdClient() *etcdClient.Client {
	dclient := e.driver.dockerd

	info, err := dclient.Info(context.Background())
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to get Info from docker client ")
		return nil
	}

	for _, manager := range info.Swarm.RemoteManagers {
		etcd, err := addrToEtcdClient(manager.Addr)
		if err == nil {
			return etcd
		}
	}

	log.WithFields(
		log.Fields{"Swarm ID": info.Swarm.NodeID,
			"IP Addr": info.Swarm.NodeAddr},
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
		Endpoints: []string{endpoint},
	}

	etcd, err := etcdClient.New(cfg)
	if err != nil {
		log.WithFields(
			log.Fields{"endpoint": endpoint,
				"error": err},
		).Error("Failed to create ETCD Client ")
		return nil, err
	}

	return etcd, nil
}

// ListVolumeName function lists all the volume names associated with this KV store
func (e *etcdKVS) ListVolumeName() ([]string, error) {
	var volumes []string

	etcd := e.createEtcdClient()
	if etcd == nil {
		return nil, fmt.Errorf(etcdClientCreateError)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	resp, err := etcd.Get(ctx, etcdPrefixState, etcdClient.WithPrefix(),
		etcdClient.WithSort(etcdClient.SortByKey, etcdClient.SortDescend))
	cancel()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to call ETCD Get for listing all volumes ")
		return nil, err
	}

	for _, ev := range resp.Kvs {
		volumes = append(volumes, strings.TrimPrefix(string(ev.Key), etcdPrefixState))
	}

	return volumes, nil
}

// WriteVolMetadata - Update or Create volume metadata in KV store
func (e *etcdKVS) WriteVolMetadata(entries []kvPair) error {

	var ops []etcdClient.Op
	var msg string
	var err error

	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	defer etcdAPI.Close()

	// ops contain multiple operations that will be done to etcd
	// in a single revision
	for _, elem := range entries {
		ops = append(ops, etcdClient.OpPut(elem.key, elem.value))
	}

	// Lets write the metadata in a single transaction
	// Use a transaction if more than one entries are to be written
	if len(entries) > 1 {
		_, err = etcdAPI.Txn(context.TODO()).Then(ops...).Commit()
	} else {
		_, err = etcdAPI.Do(context.TODO(), ops[0])
	}

	if err != nil {
		msg = fmt.Sprintf("Failed to write metadata. Reason: %v", err)
		log.Warningf(msg)
		return errors.New(msg)
	}
	return nil
}

// ReadVolMetadata - Read volume metadata in KV store
func (e *etcdKVS) ReadVolMetadata(keys []string) ([]kvPair, error) {
	var entries []kvPair
	var ops []etcdClient.Op
	var missedCount int

	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	defer etcdAPI.Close()

	// Lets build the request which will be executed
	// in a single transaction
	// ops contain multiple read operations
	for _, elem := range keys {
		ops = append(ops, etcdClient.OpGet(elem))
	}

	// Read all requested keys in one transaction
	getresp, err := etcdAPI.Txn(context.TODO()).Then(ops...).Commit()
	if err != nil {
		log.Warningf("Transactional metadata read failed: %v", err)
		return entries, err
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
		entry := kvPair{key: elem, value: string(resp.Kvs[0].Value)}
		entries = append(entries, entry)
	}

	if missedCount == len(keys) {
		// Volume does not exist
		return nil, errors.New(VolumeDoesNotExistError)
	} else if missedCount > 0 {
		// This should not happen
		// There is a volume but we couldn't read all its keys
		msg := fmt.Sprintf("Failed to get volume. Couldn't find all keys!")
		log.Warningf(msg)
		panic(msg)
	}
	return entries, nil
}

// DeleteVolMetadata - Delete volume metadata in KV store
func (e *etcdKVS) DeleteVolMetadata(name string) error {

	var msg string
	var err error

	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	defer etcdAPI.Close()

	// ops hold multiple operations that will be done to etcd
	// in a single revision. Add all keys for this volname.
	ops := []etcdClient.Op{
		etcdClient.OpDelete(etcdPrefixState + name),
		etcdClient.OpDelete(etcdPrefixGRef + name),
		etcdClient.OpDelete(etcdPrefixInfo + name),
	}

	// Delete the metadata in a single transaction
	_, err = etcdAPI.Txn(context.TODO()).Then(ops...).Commit()
	if err != nil {
		msg = fmt.Sprintf("Failed to delete metadata for volume %s. Reason: %v", name, err)
		log.Warningf(msg)
		return errors.New(msg)
	}
	return nil
}
