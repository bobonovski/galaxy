package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/proto"

	pb "github.com/bobonovski/galaxy/networking/protos"
)

func md5Hash(key string) string {
	v := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", v)
}

func runTCPClient(serverAddr, peerAddr string) {
	time.Sleep(5 * time.Second)
	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				conn, err := net.Dial("tcp", peerAddr)
				if err != nil {
					log.Fatal(err)
				}
				// serialize message in protobuf
				ledger := pb.Ledger{
					Hash:     md5Hash(fmt.Sprintf("%s current hash %d", serverAddr, time.Now().Unix())),
					PrevHash: md5Hash(fmt.Sprintf("%s previous hash %d", serverAddr, time.Now().Unix())),
				}
				ledgerBytes, err := proto.Marshal(&ledger)
				if err != nil {
					log.Fatal(err)
				}
				conn.Write(ledgerBytes)
				conn.Close()
			}
		}
	}()
	time.Sleep(60 * time.Second)
	ticker.Stop()
	log.Printf("client %s stopped\n", serverAddr)
}

func RunTCPServer(serverAddr, peerAddr string) {
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	// client sends message to server periodically
	go runTCPClient(serverAddr, peerAddr)
	// listen for connection
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept connection error: %s\n", err.Error())
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	var ledger pb.Ledger
	b, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Printf("read data from connection failed: %s\n", err.Error())
		return
	}
	// unmarshal ledger protobuf data
	err = proto.Unmarshal(b, &ledger)
	if err != nil {
		log.Printf("unmarshal ledger failed: %s\n", err.Error())
		return
	}
	log.Printf("got message: %+v", ledger)
}

var (
	serverAddr = flag.String("server_addr", "127.0.0.1:19001", "server address")
	peerAddr   = flag.String("peer_addr", "127.0.0.1:19002", "peer address")
)

func main() {
	flag.Parse()
	go RunTCPServer(*serverAddr, *peerAddr)
	go RunTCPServer(*peerAddr, *serverAddr)
	time.Sleep(1 * time.Hour)
}
