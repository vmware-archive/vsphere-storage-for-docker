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
	"fmt"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
	etcdClient "github.com/coreos/etcd/clientv3"
	"github.com/docker/engine-api/types"
)

const (
	etcdClientPort           = ":2379"
	etcdPeerPort             = ":2380"
	etcdClusterToken         = "vsphere-shared-etcd-cluster"
	etcdListenURL            = "0.0.0.0"
	etcdScheme               = "http://"
	etcdClusterStateNew      = "new"
	etcdClusterStateExisting = "existing"
)

// initEtcd start or join ETCD cluster depending on the role of the node
func (d *VolumeDriver) initEtcd() error {
	ctx := context.Background()
	cli := d.dockerd

	// get NodeID from docker client
	info, err := cli.Info(ctx)
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to get Info from docker client ")
		return err
	}

	// worker just returns
	nodeID := info.Swarm.NodeID
	if info.Swarm.ControlAvailable == false {
		log.WithFields(
			log.Fields{"nodeID": nodeID},
		).Info("Swarm node role: worker. Action: return from InitEtcd ")
		return nil
	}

	// check my local role
	node, _, err := cli.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		log.WithFields(log.Fields{"nodeID": nodeID,
			"error": err}).Error("Failed to inspect node ")
		return err
	}

	// get the IP address of current node
	addr := info.Swarm.NodeAddr

	// if leader, proceed to start ETCD cluster
	if node.ManagerStatus.Leader {
		log.WithFields(
			log.Fields{"nodeID": nodeID},
		).Info("Swarm node role: leader, start etcd cluster")
		err = startEtcdCluster(addr, nodeID)
		if err != nil {
			log.WithFields(log.Fields{"nodeID": nodeID,
				"error": err}).Error("Failed to start ETCD Cluster")
			return err
		}
		return nil
	}

	// if manager, first find out who's leader, then proceed to join ETCD cluster
	nodes, err := cli.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		log.WithFields(log.Fields{"nodeID": nodeID,
			"error": err}).Error("Failed to get NodeList from swarm manager")
		return err
	}
	for _, n := range nodes {
		if n.ManagerStatus != nil && n.ManagerStatus.Leader == true {
			log.WithFields(
				log.Fields{"leader ID": n.ID,
					"manager ID": nodeID},
			).Info("Swarm node role: manager. Action: find leader ")

			joinEtcdCluster(addr, n.ManagerStatus.Addr, nodeID)
			return nil
		}
	}

	err = fmt.Errorf("Failed to get leader for swarm manager %s", nodeID)
	return err
}

// startEtcdCluster function is called by swarm leader to start a ETCD cluster
func startEtcdCluster(nodeAddr string, nodeID string) error {
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
	go etcdService(lines)

	return nil
}

// joinEtcdCluster function is called by a non-leader swarm manager to join a ETCD cluster
func joinEtcdCluster(nodeAddr string, leaderAddr string, nodeID string) error {
	cli, err := addrToEtcdClient(leaderAddr)
	if err != nil {
		log.WithFields(
			log.Fields{"nodeAddr": nodeAddr,
				"leaderAddr": leaderAddr,
				"nodeID":     nodeID},
		).Error("Failed to join ETCD cluster on manager ")
	}

	// list all current ETCD members, check if this node is already added as a member
	lresp, err := cli.MemberList(context.Background())
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
				).Info("Already joined as etcd member but not started. ")

				existing = true
			} else {
				// same peerAddr already existing and started, need to remove before re-join
				// we need the remove since etcd data directory is not persistent
				// thus the node cannot re-join as the same member as before
				log.WithFields(
					log.Fields{"nodeID": nodeID,
						"peerAddr": peerAddr},
				).Info("Already joined as a etcd member and started. Action: remove self before re-join ")

				_, err = cli.MemberRemove(context.Background(), member.ID)
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
		aresp, err := cli.MemberAdd(context.Background(), peerAddrs)
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

	go etcdService(lines)

	return nil
}

func etcdService(cmd []string) {
	_, err := exec.Command("/bin/etcd", cmd...).Output()
	if err != nil {
		log.WithFields(
			log.Fields{"error": err, "cmd": cmd},
		).Error("Failed to start etcd command ")
	}
}

func (d *VolumeDriver) createEtcdClient() *etcdClient.Client {
	dclient := d.dockerd

	info, err := dclient.Info(context.Background())
	if err != nil {
		log.WithFields(
			log.Fields{"error": err},
		).Error("Failed to get Info from docker client ")
		return nil
	}

	for _, manager := range info.Swarm.RemoteManagers {
		cli, err := addrToEtcdClient(manager.Addr)
		if err == nil {
			return cli
		}
	}

	log.WithFields(
		log.Fields{"Swarm ID": info.Swarm.NodeID,
			"IP Addr": info.Swarm.NodeAddr},
	).Error("Failed to create etcd client according to manager info ")
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

	cli, err := etcdClient.New(cfg)
	if err != nil {
		log.WithFields(
			log.Fields{"endpoint": endpoint,
				"error": err},
		).Error("Failed to create ETCD Client ")
		return nil, err
	}

	return cli, nil
}
