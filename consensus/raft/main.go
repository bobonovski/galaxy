package main

import (
	"flag"
	"strings"

	"github.com/coreos/etcd/raft/raftpb"
)

var (
	peers  = flag.String("peers", "http://127.0.0.1:9001", "comma separated cluster peers")
	nodeId = flag.Int("node_id", 1, "node id")
	kvPort = flag.Int("kv_port", 9002, "http port of kv store")
	join   = flag.Bool("join", false, "whether to join an existing cluster")
)

func main() {
	flag.Parse()

	proposeChan := make(chan string)
	confChangeChan := make(chan raftpb.ConfChange)
	defer func() {
		close(proposeChan)
		close(confChangeChan)
	}()

	commitChan, errorChan := newRaftNode(*nodeId,
		strings.Split(*peers, ","), *join, proposeChan, confChangeChan)

	kvs := newKVStore(proposeChan, commitChan, errorChan)
}
