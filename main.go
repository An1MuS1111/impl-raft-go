package main

import (
	"fmt"
	"impl-raft-go/config"
	"impl-raft-go/server"
	"log"
)

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cfg, err := config.NewConfig("config.dat")
	handleError(err)
	defer cfg.File.Close()

	raftNode, err := server.NewRaftNode(cfg)
	handleError(err)

	// customized log settings displaying ServerID, ServerAddr, and currentTerm
	log.SetFlags(log.Ltime)
	prefixString := fmt.Sprintf("\n[%d; %s] term: %d ~ ", cfg.ServerID, cfg.ServerAddr, raftNode.GetCurrentTerm())
	log.SetPrefix(prefixString)

	err = raftNode.StartServer()
	handleError(err)

}
