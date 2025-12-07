package main

import (
	"flag"
	"fmt"
	"impl-raft-go/raft"
	"net"
	"os"
	"strconv"
	"strings"
)

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	id    = flag.Uint64("id", 0, "Node ID. It must be a non-zero integer.")
	peers = flag.String("peers", "", "1=127.0.0.1:8081,2=127.0.0.1:8082)")
)

func main() {

	file, err := os.OpenFile("config.log", os.O_CREATE|os.O_RDWR, 0644)
	handleError(err)
	defer file.Close()

	flag.Parse()
	if *id == 0 {
		handleError(fmt.Errorf("id must be a non-zero value"))
	}

	peerMap := make(map[uint64]net.Addr)
	if *peers != "" {
		peerList := strings.Split(*peers, ",")
		for _, p := range peerList {
			parts := strings.Split(p, "=")
			if len(parts) != 2 {
				handleError(fmt.Errorf("invalid peer format: %s", p))
			}
			peerID, err := strconv.ParseUint(parts[0], 10, 64)
			handleError(err)

			addr, err := net.ResolveTCPAddr("tcp", parts[1])
			handleError(err)
			peerMap[peerID] = addr
		}
	}

	addr, ok := peerMap[*id]
	if !ok {
		handleError(fmt.Errorf("node's own id %d not found in peers list", *id))
	}

	raftNode, err := raft.NewRaftNode(file, *id, addr)
	handleError(err)
	raftNode.SetPeers(peerMap)
}
