package main

import (
	"fmt"
	"impl-raft-go/config"
	"log"
	"os"
)

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	file, err := os.OpenFile("config.dat", os.O_CREATE|os.O_RDWR, 0644)
	handleError(err)
	defer file.Close()

	cfg, err := config.NewConfig()
	handleError(err)

	// log formatting
	prefixString := fmt.Sprintf("\n%d - [%s] ", cfg.ServerID, cfg.ServerAddr)
	log.SetFlags(log.Ltime)
	log.SetPrefix(prefixString)
	log.Printf("Hello there")

	// raftNode, err := server.NewRaftNode(file, id, addr, addrMap)
	// handleError(err)

	// err = raftNode.StartServer()
	// handleError(err)

}
