package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"sync"
)

// internal key-value store
type kvStore struct {
	mu          sync.RWMutex
	proposeChan chan<- string
	store       map[string]string
}

// key-value pair
type kvPair struct {
	Key   string
	Value string
}

func newKVStore(proposeChan chan<- string,
	commitChan <-chan *string,
	errorChan <-chan error) *kvStore {
	kvs := &kvStore{
		proposeChan: proposeChan,
		store:       make(map[string]string),
	}
	go kvs.readCommits(commitChan, errorChan)
	return kvs
}

func (kvs *kvStore) Lookup(key string) (string, bool) {
	kvs.mu.RLock() // lock for read
	v, ok := kvs.store[key]
	kvs.mu.RUnlock()
	return v, ok
}

func (kvs *kvStore) Propose(key string, val string) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(kvPair{Key: key, Value: val}); err != nil {
		log.Fatal(err)
	}
	kvs.proposeChan <- buf.String()
}

func (kvs *kvStore) readCommits(commitChan <-chan *string, errorChan <-chan error) {
	for {
		select {
		case c := <-commitChan:
			var kv kvPair
			decoder := gob.NewDecoder(bytes.NewBufferString(*c))
			if err := decoder.Decode(&kv); err != nil {
				log.Fatalf("cannot decode message: %v", err)
			}
			kvs.mu.Lock() // lock for write
			kvs.store[kv.Key] = kv.Value
			kvs.mu.Unlock()
		case err := <-errorChan:
			log.Fatal(err)
		}
	}
}
