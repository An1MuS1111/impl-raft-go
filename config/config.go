package config

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Config struct {
	ServerID   uint64
	ServerAddr net.Addr
	Peers      map[uint64]net.Addr
}

func NewConfig() (*Config, error) {

	config := &Config{}

	var (
		id         uint64
		addrString string
	)

	flag.Uint64Var(&id, "id", 0, "serverID must be a non-zero integer.")
	flag.StringVar(&addrString, "addrs", "", "1=127.0.0.1:8081,2=127.0.0.1:8082)")
	flag.Parse()

	if id == 0 {
		return nil, fmt.Errorf("serverID must be a non-zero value, found: %d", id)
	}

	// create address map from the string flag
	addrMap := make(map[uint64]net.Addr)
	if addrString != "" {
		for p := range strings.SplitSeq(addrString, ",") {
			parts := strings.Split(p, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid addr format: %s", p)
			}
			addrID, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				return nil, err
			}

			addr, err := net.ResolveTCPAddr("tcp", parts[1])
			if err != nil {
				return nil, err
			}
			addrMap[addrID] = addr
		}
	}

	if addr, ok := addrMap[id]; ok {
		config.ServerAddr = addr
		delete(addrMap, config.ServerID)
	} else {
		return nil, fmt.Errorf("ServerID: %d not found in addrs list", id)
	}
	config.ServerID = id
	config.Peers = addrMap

	return config, nil
}
