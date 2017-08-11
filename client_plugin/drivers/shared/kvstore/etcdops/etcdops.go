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
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	etcdClient "github.com/coreos/etcd/clientv3"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/dockerops"
	"github.com/vmware/docker-volume-vsphere/client_plugin/drivers/shared/kvstore"
)

/*
   etcdClientPort:             port for etcd clients to talk to the peers
   etcdPeerPort:               port for etcd peers talk to each other
   etcdClusterToken:           ID of the cluster to create/join
   etcdListenURL:              etcd listening interface
   etcdScheme:                 Protocol used for communication
   etcdClusterStateNew:        Used to indicate the formation of a new
                               cluster
   etcdClusterStateExisting:   Used to indicate that this node is joining
                               an existing etcd cluster
   requestTimeout:             After how long should an etcd request timeout
   etcdClientCreateError:      Error indicating failure to create etcd client
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
	requestTimeout           = 5 * time.Second
	checkSleepDuration       = time.Second
	etcdClientCreateError    = "Failed to create etcd client"
	etcdSingleRef            = "1"
	etcdNoRef                = "0"
)

type EtcdKVS struct {
	dockerOps *dockerops.DockerOps
	nodeID    string
	nodeAddr  string
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
	}

	if !isManager {
		log.WithFields(
			log.Fields{"nodeID": nodeID},
		).Info("Swarm node role: worker. Return from NewKvStore ")
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
		return e
	}

	// if manager, first find out who's leader, then proceed to join ETCD cluster
	leaderAddr, err := dockerOps.GetSwarmLeader()
	if err != nil {
		log.WithFields(
			log.Fields{
				"nodeID": nodeID,
				"error":  err},
		).Error("Failed to get swarm leader address ")
		return nil
	}

	err = e.joinEtcdCluster(leaderAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"nodeID": nodeID,
			"error":  err},
		).Error("Failed to join ETCD Cluster")
		return nil
	}
	return e
}

// startEtcdCluster function is called by swarm leader to start a ETCD cluster
func (e *EtcdKVS) startEtcdCluster() error {
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
func (e *EtcdKVS) joinEtcdCluster(leaderAddr string) error {
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
func (e *EtcdKVS) checkLocalEtcd() error {
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
	// TODO: when the manager is demoted to worker, the watcher should be cancelled
	watchCh := cli.Watch(context.Background(), kvstore.VolPrefixGRef,
		etcdClient.WithPrefix(), etcdClient.WithPrevKV())
	for wresp := range watchCh {
		for _, ev := range wresp.Events {
			e.etcdEventHandler(ev)
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
		fn func(string) bool) {
		// watcher observes global refcount critical change
		// transactional edit state first
		volName := strings.TrimPrefix(key, kvstore.VolPrefixGRef)
		succeeded := e.CompareAndPutStateOrBusywait(kvstore.VolPrefixState+volName,
			string(fromState), string(interimState))
		if !succeeded {
			// this handler doesn't get the right to start server
			return
		}

		if fn(volName) {
			// server start/stop succeed, set desired state
			if e.CompareAndPut(kvstore.VolPrefixState+volName,
				string(interimState),
				string(toState)) == false {
				// Failed to set state to desired state
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
	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	if etcdAPI == nil {
		log.Warningf(etcdClientCreateError)
		return false
	}
	defer etcdAPI.Close()
	txresp, err := etcdAPI.Txn(context.TODO()).If(
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

//CompareAndPutOrFetch - Compare and put of get the current value of the key
func (e *EtcdKVS) CompareAndPutOrFetch(key string,
	oldVal string,
	newVal string) (*etcdClient.TxnResponse, error) {

	var txresp *etcdClient.TxnResponse
	// Create a client to talk to etcd
	etcdAPI := e.createEtcdClient()
	if etcdAPI == nil {
		return txresp, errors.New(etcdClientCreateError)
	}
	defer etcdAPI.Close()
	txresp, err := etcdAPI.Txn(context.TODO()).If(
		etcdClient.Compare(etcdClient.Value(key), "=", oldVal),
	).Then(
		etcdClient.OpPut(key, newVal),
	).Else(
		etcdClient.OpGet(key),
	).Commit()

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

	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	timer := time.NewTimer(2 * requestTimeout)
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

// List function lists all the different portion of keys with the given prefix
func (e *EtcdKVS) List(prefix string) ([]string, error) {
	var keys []string

	client := e.createEtcdClient()
	if client == nil {
		return keys, fmt.Errorf(etcdClientCreateError)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
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

// WriteMetaData - Update or Create metadata in KV store
func (e *EtcdKVS) WriteMetaData(entries []kvstore.KvPair) error {

	var ops []etcdClient.Op
	var msg string
	var err error

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
	if len(entries) > 1 {
		_, err = client.Txn(context.TODO()).Then(ops...).Commit()
	} else {
		_, err = client.Do(context.TODO(), ops[0])
	}

	if err != nil {
		msg = fmt.Sprintf("Failed to write metadata. Reason: %v", err)
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
	getresp, err := client.Txn(context.TODO()).Then(ops...).Commit()
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
	return entries, nil
}

// DeleteMetaData - Delete volume metadata in KV store
func (e *EtcdKVS) DeleteMetaData(name string) error {

	var msg string
	var err error

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
	_, err = client.Txn(context.TODO()).Then(ops...).Commit()
	if err != nil {
		msg = fmt.Sprintf("Failed to delete metadata for volume %s. Reason: %v", name, err)
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
	timer := time.NewTimer(requestTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
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
	timer := time.NewTimer(requestTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
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
	client := e.createEtcdClient()
	if client == nil {
		return "", fmt.Errorf(etcdClientCreateError)
	}
	defer client.Close()

	ticker := time.NewTicker(checkSleepDuration)
	defer ticker.Stop()
	timer := time.NewTimer(2 * requestTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			txresp, err := client.Txn(context.TODO()).If(
				etcdClient.Compare(etcdClient.Value(key), "=", value),
			).Then(
				etcdClient.OpGet(newKey),
			).Commit()

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
