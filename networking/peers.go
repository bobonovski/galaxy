package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"time"
)

func runTCPClient(serverAddr, peerAddr string) {
	time.Sleep(5 * time.Second)
	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for t := range ticker.C {
			conn, err := net.Dial("tcp", peerAddr)
			if err != nil {
				log.Fatal(err)
			}
			io.WriteString(conn,
				fmt.Sprintf("peer %s send message %v\n", serverAddr, t))
			conn.Close()
		}
	}()
	time.Sleep(60 * time.Second)
	ticker.Stop()
	log.Printf("client %s stopped\n", serverAddr)
}

func runUDPClient(serverAddr, peerAddr string) {
	time.Sleep(5 * time.Second)
	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for t := range ticker.C {
			conn, err := net.Dial("udp", peerAddr)
			if err != nil {
				log.Fatal(err)
			}
			io.WriteString(conn,
				fmt.Sprintf("peer %s send message %v", serverAddr, t))
			conn.Close()
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

func RunUDPServer(serverAddr, peerAddr string) {
	conn, err := net.ListenPacket("udp", serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	// client sends message to server periodically
	go runUDPClient(serverAddr, peerAddr)
	// listen for connection
	for {
		b := make([]byte, 1024)
		_, _, err := conn.ReadFrom(b)
		if err != nil {
			log.Printf("read data from UDP connection failed: %s\n", err.Error())
			continue
		}
		log.Printf("got message: %s", string(b))
	}
}

func handleConnection(conn net.Conn) {
	b, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Printf("read data from connection failed: %s\n", err.Error())
		return
	}
	log.Printf("got message: %s", string(b))
}

var (
	serverAddr = flag.String("server_addr", "127.0.0.1:19001", "server address")
	peerAddr   = flag.String("peer_addr", "127.0.0.1:19002", "peer address")
	protocol   = flag.String("protocol", "tcp", "protocol for testing")
)

func main() {
	flag.Parse()
	if *protocol == "tcp" {
		go RunTCPServer(*serverAddr, *peerAddr)
		go RunTCPServer(*peerAddr, *serverAddr)
	} else if *protocol == "udp" {
		go RunUDPServer(*serverAddr, *peerAddr)
		go RunUDPServer(*peerAddr, *serverAddr)
	}
	time.Sleep(1 * time.Hour)
}
