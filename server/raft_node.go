package server

import (
	"encoding/gob"
	"fmt"
	"impl-raft-go/config"
	"impl-raft-go/proto"
	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type State uint64

const (
	Leader State = iota
	Candidate
	Follower
)

type Log struct {
	Index   uint64 // first index has to be 1 as mentioned in the raft paper
	Term    uint64
	Command string
}

//***commitIndex***
// commitIndex: This represents the index of the highest log entry that is known to be committed to the cluster.
// A log entry is considered committed when it has been replicated to a majority of servers in the Raft cluster
// and the leader has acknowledged its commitment.
// The commitIndex is a volatile variable on all servers
// and is updated by the leader and communicated to followers via AppendEntries RPCs.
// Once an entry is committed, it is guaranteed to be durable
// and will eventually be applied to all state machines.
//? so why commitIndex isn't stored in persistence
// article: https://www.linkedin.com/pulse/raft-commitindex-lastapplied-migo-lee-5jgjc
// The only purpose of commitIndex is to bound how far state machines can advance.
// It's ok to hold back state machines for brief periods of time,
// especially when a new leader comes online,
// since state machines are briefly stalled anyways with respect to new entries then.
// The source of truth for which entries are committed (guaranteed to persist forever) is the cluster's logs.
//  Servers keep track of the latest index they know is committed,
// but there may always be more entries committed (in the guaranteed to persist forever sense) than servers' commit Index values reflect.
// For example, the moment an entry is sufficiently replicated,
// it is committed, but it takes the leader receiving a bunch of acknowledgements to realize this and update its commitIndex.
// If you think of commitment that way, it's not a big stretch
// to accept the idea that servers' commitIndex values may sometimes reset to 0 (when they reboot) and nothing bad comes of it.

type RaftNode struct {
	proto.UnimplementedRaftServiceServer
	mu sync.Mutex

	//! read-only state
	ServerID   uint64
	ServerAddr net.Addr
	Peers      map[uint64]net.Addr

	//! persistence state
	currentTerm uint64 // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor    uint64 // candidateId that received vote in current term (or null if none)
	log         []Log  // log entries; each entry contains command for state machine, and term when entry was received by leader (first index is 1)

	commitIndex uint64 // index of highest log entry known to be committed (initialized at 0, increases monotonically)
	lastApplied uint64 //? not entirely sure about this! how to retrieve the commit index on recovery?

	//! volatile state on leader (reintialized after election)
	nextIndex  map[uint64]uint64 // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex map[uint64]uint64 // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	//--- Above are the properties of state mentioned in the raft paper

	//! volatile state
	currentRole State
	// currentLeader unique
	// votesReceived uint64
	// sentLength    uint64
	// ackedLength   uint64

	servers map[uint64]proto.RaftServiceClient

	electionReset chan struct{}
}

// Helper to establish GRPC connections to all peers
func (r *RaftNode) dialPeers() {

	for id, peer := range r.Peers {

		peerStr := peer.String()
		conn, err := grpc.NewClient(peerStr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if conn != nil {
			fmt.Printf("connected to: %v", peerStr)
		}
		// allow partial connectivity; handle in production
		if err != nil {
			fmt.Printf("failed to connect to peer %d (%s): %v\n", id, peerStr, err)
		} else {
			r.servers[id] = proto.NewRaftServiceClient(conn)
		}

	}
}

func (r *RaftNode) restoreState(file *os.File) error {
	decoder := gob.NewDecoder(file)

	if err := decoder.Decode(&r.currentTerm); err != nil {
		return fmt.Errorf("failed to decode currentTerm: %v", err)
	}
	if err := decoder.Decode(&r.votedFor); err != nil {
		return fmt.Errorf("failed to decode votedFor: %v", err)
	}
	if err := decoder.Decode(&r.log); err != nil {
		return fmt.Errorf("failed to decode log: %v", err)
	}
	// if err := decoder.Decode(&r.commitIndex); err != nil {
	// 	return fmt.Errorf("failed to decode commitIndex: %v", err)

	// }

	return nil
}

func (r *RaftNode) storeState(file *os.File) error {

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning: %v", err)
	}
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(&r.currentTerm); err != nil {
		return fmt.Errorf("failed to Encode currentTerm: %v", err)
	}
	if err := encoder.Encode(&r.votedFor); err != nil {
		return fmt.Errorf("failed to Encode votedFor: %v", err)
	}
	if err := encoder.Encode(&r.log); err != nil {
		return fmt.Errorf("failed to Encode log: %v", err)
	}
	// if err := encoder.Encode(&r.commitIndex); err != nil {
	// 	return fmt.Errorf("failed to decode commitIndex: %v", err)

	// }

	return nil
}

func NewRaftNode(cfg *config.Config) (*RaftNode, error) {

	// A node can be of three states (Leader, Candidate, Follower)
	// When a node first starts running, or when it crashes and recovers,
	// it starts up in the follower state and awaits messages from other nodes.
	raftNode := &RaftNode{
		// read-only
		ServerID:   cfg.ServerID,
		ServerAddr: cfg.ServerAddr,
		Peers:      cfg.Peers,
		// persistence
		currentTerm: 0,
		votedFor:    0,
		log:         make([]Log, 0),
		// volatile
		commitIndex: 0,
		lastApplied: 0,
		// volatile state on leader
		nextIndex:  make(map[uint64]uint64, 0),
		matchIndex: make(map[uint64]uint64, 0),

		currentRole:   Follower,
		servers:       make(map[uint64]proto.RaftServiceClient),
		electionReset: make(chan struct{}),
	}

	// raftNode.setPeers(addrs)
	// [incomplete]
	// in dynamic state this should be handled as hotswappable
	raftNode.dialPeers()

	// extract currentTerm, votedFor, log, and commitIndex
	// from the disk for initialization

	cfgFile := cfg.File

	if _, err := cfgFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to beginning: %v", err)
	}

	stat, err := cfgFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats of file: %v", err)
	}
	// if the size == 0 that means the the node is newly added to the quorum
	// other wise we think the node has crashed or
	// for some reason it was shutdown to apply some updates
	cfgFileSize := stat.Size()
	if cfgFileSize != 0 {
		// sync the node and retrieve the persisted states
		log.Printf("server restarted. config size: %v", cfgFileSize)
		log.Printf("retrieving persisted state")
		if err := raftNode.restoreState(cfgFile); err != nil {
			return nil, fmt.Errorf("failed to restore state from persistence: %v", err)
		}
	}

	// //***need to evaluate this thing, and where to put it***
	// if raftNode.currentRole == Leader {
	// 	go func() {
	// 		ctx := context.Background()

	// 		ticker := time.Tick(time.Millisecond * 300)

	// 		for {
	// 			<-ticker

	// 			for id, client := range raftNode.servers {
	// 				reply, err := client.HealthCheck(ctx, &proto.HealthCheckArgs{Ping: "Ping"})
	// 				if reply == nil {
	// 					log.Printf("no reply from: %v", raftNode.peers[id])
	// 				}

	// 				if err != nil {
	// 					log.Printf("reconnecting: waiting for connection: %v", raftNode.peers[id])
	// 				}

	// 				log.Printf("sent to: %v, reply: %v", raftNode.peers[id], reply)
	// 			}
	// 		}
	// 	}()
	// }

	return raftNode, nil
}

func (r *RaftNode) GetCurrentTerm() uint64 {
	return r.currentTerm
}
