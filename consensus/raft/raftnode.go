package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/coreos/etcd/etcdserver/api/rafthttp"
	stats "github.com/coreos/etcd/etcdserver/api/v2stats"
	"github.com/coreos/etcd/pkg/types"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"

	"go.uber.org/zap"
)

type raftNode struct {
	proposeChan    <-chan string
	confChangeChan <-chan raftpb.ConfChange
	commitChan     chan<- *string
	errorChan      chan<- error

	nodeId int      // node id
	peers  []string // raft peers
	join   bool     // whether node is joining and existing cluster

	confState raftpb.ConfState

	appliedIndex uint64

	node        raft.Node // internal raft node
	raftStorage *raft.MemoryStorage

	transport *rafthttp.Transport

	stopChan     chan struct{} // notify proposal channel closing
	httpStopChan chan struct{} // notify http server to shutdown
	httpDoneChan chan struct{} // notify http server shutdown complete
}

func newRaftNode(nodeId int, peers []string, join bool,
	proposeChan <-chan string,
	confChangeChan <-chan raftpb.ConfChange) (<-chan *string, <-chan error) {
	commitChan := make(chan *string)
	errorChan := make(chan error)

	rn := raftNode{
		proposeChan:    proposeChan,
		confChangeChan: confChangeChan,
		commitChan:     commitChan,
		errorChan:      errorChan,
		nodeId:         nodeId,
		peers:          peers,
		join:           join,
		appliedIndex:   uint64(0),
		raftStorage:    raft.NewMemoryStorage(),
		stopChan:       make(chan struct{}),
		httpStopChan:   make(chan struct{}),
		httpDoneChan:   make(chan struct{}),
	}

	go rn.startRaft()

	return commitChan, errorChan
}

func (rn *raftNode) startRaft() {
	var peers []raft.Peer
	for i, _ := range rn.peers {
		peers = append(peers, raft.Peer{ID: uint64(i + 1)})
	}
	// raft config for node
	config := &raft.Config{
		ID:              uint64(rn.nodeId),
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         rn.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
	}
	// start a new node
	startPeers := peers
	if rn.join {
		startPeers = nil
	}
	rn.node = raft.StartNode(config, startPeers)
	// init raft http transport
	rn.transport = &rafthttp.Transport{
		Logger:      zap.NewExample(),
		ID:          types.ID(rn.nodeId),
		ClusterID:   0x1000,
		Raft:        rn, // raft state machine
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(strconv.Itoa(rn.nodeId)),
	}
	rn.transport.Start()
	for i, peer := range rn.peers {
		if i+1 != rn.nodeId {
			rn.transport.AddPeer(types.ID(i+1), []string{peer})
		}
	}

	go rn.serveRaft()
	go rn.serveEvent()
}

// stop raft node
func (rn *raftNode) stop() {
	rn.transport.Stop()
	close(rn.httpStopChan)
	<-rn.httpDoneChan
	close(rn.commitChan)
	close(rn.errorChan)
	rn.node.Stop()
}

// raft http server
func (rn *raftNode) serveRaft() {
	log.Printf("start to serve raft\n")
	url, err := url.Parse(rn.peers[rn.nodeId-1])
	if err != nil {
		log.Fatalf("parse url failed: %v", err)
	}
	server := http.Server{
		Addr:    url.Host,
		Handler: rn.transport.Handler(),
	}
	go func() {
		if err = server.ListenAndServe(); err != nil {
			log.Fatalf("listen and serve failed: %v", err)
		}
	}()

	<-rn.httpStopChan

	server.Shutdown(nil)
	close(rn.httpDoneChan)
}

// serve raft event loop
func (rn *raftNode) serveEvent() {
	log.Printf("start to serve event\n")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// send proposals over raft
	go func() {
		confChangeCount := uint64(0)
		for rn.proposeChan != nil && rn.confChangeChan != nil {
			select {
			case prop, ok := <-rn.proposeChan:
				if !ok {
					rn.proposeChan = nil
				} else {
					log.Printf("raft got proposal %x\n", prop)
					rn.node.Propose(context.TODO(), []byte(prop))
				}
			case cc, ok := <-rn.confChangeChan:
				if !ok {
					rn.confChangeChan = nil
				} else {
					confChangeCount += 1
					cc.ID = confChangeCount
					rn.node.ProposeConfChange(context.TODO(), cc)
				}
			}
		}
		close(rn.stopChan)
	}()
	// raft event loop
	for {
		select {
		case <-ticker.C:
			rn.node.Tick()
		case ready := <-rn.node.Ready():
			rn.raftStorage.Append(ready.Entries)
			rn.transport.Send(ready.Messages)
			filteredEnts := rn.filterEntries(ready.CommittedEntries)
			if ok := rn.publishEntries(filteredEnts); !ok {
				rn.stop()
				return
			}
			rn.node.Advance()
		}
	}
}

// filter applied entries
func (rn *raftNode) filterEntries(ents []raftpb.Entry) []raftpb.Entry {
	var filteredEnts []raftpb.Entry
	if len(ents) == 0 {
		return filteredEnts
	}
	firstIdx := ents[0].Index
	if firstIdx > rn.appliedIndex+1 {
		log.Fatalf("local entries falls behind committed entries")
	}
	if rn.appliedIndex < firstIdx+uint64(len(ents)-1) {
		filteredEnts = ents[rn.appliedIndex-firstIdx+1:]
	}
	return filteredEnts
}

// publish valid entries to commit
func (rn *raftNode) publishEntries(ents []raftpb.Entry) bool {
	for i, _ := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				break
			}
			s := string(ents[i].Data)
			select {
			case rn.commitChan <- &s:
			case <-rn.stopChan:
				return false
			}
		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			cc.Unmarshal(ents[i].Data)
			rn.confState = *rn.node.ApplyConfChange(cc)
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				if len(cc.Context) > 0 {
					rn.transport.AddPeer(types.ID(cc.NodeID), []string{string(cc.Context)})
				}
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == uint64(rn.nodeId) {
					log.Println("I've been removed from the cluster, shutdown")
					return false
				}
				rn.transport.RemovePeer(types.ID(cc.NodeID))
			}
		}
		rn.appliedIndex = ents[i].Index
	}
	return true
}

// implement the Raft interface methods
func (rn *raftNode) Process(ctx context.Context, m raftpb.Message) error {
	return rn.node.Step(ctx, m)
}
func (rn *raftNode) IsIDRemoved(id uint64) bool                           { return false }
func (rn *raftNode) ReportUnreachable(id uint64)                          {}
func (rn *raftNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {}
