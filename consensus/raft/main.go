package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

	serveKVStore(kvs, *kvPort, errorChan)
}

func serveKVStore(kvs *kvStore, port int, errorChan <-chan error) {
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: &kvStoreHandler{store: kvs},
	}
	go func() {
		log.Printf("start to listen on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	if err, ok := <-errorChan; ok {
		log.Fatal(err)
	}
}

// define custom hander for kv store
type kvStoreHandler struct {
	store *kvStore
}

func (h *kvStoreHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.RequestURI
	switch {
	case r.Method == "POST": // set key-value
		log.Printf("received key-value to set: key=%s\n", key)
		v, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("failed to put key-value: %v", err)
			http.Error(w, "failed to put key-value", http.StatusBadRequest)
		}
		h.store.Propose(key, string(v))
		// TODO wait until receive commit message
		w.WriteHeader(http.StatusNoContent)
	case r.Method == "GET": // get value of key
		if v, ok := h.store.Lookup(key); ok {
			w.Write([]byte(v))
		} else {
			http.Error(w, "Failed to get value", http.StatusNotFound)
		}
	default:
		w.Header().Set("Allow", "PUT")
		w.Header().Add("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
