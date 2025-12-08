package main

import (
	"flag"
	"fmt"
	"impl-raft-go/server"
	"log"
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
	id    uint64
	addrs string
)

func init() {
	flag.Uint64Var(&id, "id", 0, "Node ID. It must be a non-zero integer.")
	flag.StringVar(&addrs, "addrs", "", "1=127.0.0.1:8081,2=127.0.0.1:8082)")
}

func main() {

	log.SetFlags(log.Ltime)
	log.SetPrefix("raft: ")

	file, err := os.OpenFile("config.dat", os.O_CREATE|os.O_RDWR, 0644)
	handleError(err)
	defer file.Close()

	flag.Parse()

	if id == 0 {
		handleError(fmt.Errorf("id must be a non-zero value"))
	}

	addrMap := make(map[uint64]net.Addr)
	if addrs != "" {
		for p := range strings.SplitSeq(addrs, ",") {
			parts := strings.Split(p, "=")
			if len(parts) != 2 {
				handleError(fmt.Errorf("invalid addr format: %s", p))
			}
			addrID, err := strconv.ParseUint(parts[0], 10, 64)
			handleError(err)

			addr, err := net.ResolveTCPAddr("tcp", parts[1])
			handleError(err)
			addrMap[addrID] = addr
		}
	}

	addr, ok := addrMap[id]
	if !ok {
		handleError(fmt.Errorf("node's own id %d not found in addrs list", id))
	}

	raftNode, err := server.NewRaftNode(file, id, addr, addrMap)
	handleError(err)

	err = raftNode.StartServer()
	handleError(err)

}
