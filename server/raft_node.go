package server

import (
	"encoding/gob"
	"fmt"
	"impl-raft-go/proto"
	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServerRole uint64

const (
	Leader ServerRole = iota
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
	id    uint64
	addr  net.Addr
	peers map[uint64]net.Addr

	//! persistence state
	currentTerm uint64 // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor    uint64 // candidateId that received vote in current term (or null if none)
	log         []Log  // log entries; each entry contains command for state machine, and term when entry was received by leader (first index is 1)

	commitIndex uint64 // index of highest log entry known to be committed (initialized at 0, increases monotonically)
	lastApplied uint64 //? not entirely sure about this! how to retrieve the commit index on recovery?

	//! volatile state on leader (reintialized after election)
	nextIndex  []uint64 // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex []uint64 // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	//--- Above are the properties of state mentioned in the raft paper

	//! volatile state
	currentRole ServerRole
	// currentLeader unique
	// votesReceived uint64
	// sentLength    uint64
	// ackedLength   uint64

	clients map[uint64]proto.RaftServiceClient

	electionReset chan struct{}
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

// Helper to establish GRPC connections to all peers
func (r *RaftNode) dialPeers() {
	peers := r.peers

	for id, peer := range peers {

		peerStr := peer.String()
		conn, err := grpc.NewClient(peerStr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if conn != nil {
			fmt.Printf("connected to: %v", peerStr)
		}
		// allow partial connectivity; handle in production
		if err != nil {
			fmt.Printf("failed to connect to peer %d (%s): %v\n", id, peerStr, err)
		} else {
			r.clients[id] = proto.NewRaftServiceClient(conn)
		}

	}
}

func (r *RaftNode) setPeers(addrs map[uint64]net.Addr) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// remove self from peers
	delete(addrs, r.id)
	r.peers = addrs

	fmt.Println(r.id, r.addr, r.peers)
}

func NewRaftNode(file *os.File, id uint64, addr net.Addr, addrs map[uint64]net.Addr) (*RaftNode, error) {
	// A node can be of three states (Leader, Candidate, Follower)
	// When a node first starts running, or when it crashes and recovers,
	// it starts up in the follower state and awaits messages from other nodes.
	raftNode := &RaftNode{
		// read-only
		id:    id,
		addr:  addr,
		peers: make(map[uint64]net.Addr),
		// persistence
		currentTerm: 0,
		votedFor:    0,
		log:         make([]Log, 0),
		// volatile
		commitIndex: 0,
		lastApplied: 0,
		// volatile state on leader
		nextIndex:  make([]uint64, 0),
		matchIndex: make([]uint64, 0),

		currentRole:   Follower,
		clients:       make(map[uint64]proto.RaftServiceClient),
		electionReset: make(chan struct{}),
	}

	raftNode.setPeers(addrs)

	// extract currentTerm, votedFor, log, and commitIndex
	// from the disk for initialization

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to beginning: %v", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats of file: %v", err)
	}
	// if the size == 0 that means the the node is newly added to the quorum
	// other wise we think the node has crashed or
	// for some reason it was shutdown to apply some updates
	configSize := stat.Size()
	if configSize != 0 {
		// sync the node and retrieve the persisted states
		log.Printf("server restarted. config size: %v", configSize)
		log.Printf("retrieving persisted state")
		if err := raftNode.restoreState(file); err != nil {
			return nil, fmt.Errorf("failed to restore state from persistence: %v", err)
		}
	}

	// raftNode.dialPeers()

	// //***need to evaluate this thing, and where to put it***
	// if raftNode.currentRole == Leader {
	// 	go func() {
	// 		ctx := context.Background()

	// 		ticker := time.Tick(time.Millisecond * 300)

	// 		for {
	// 			<-ticker

	// 			for id, client := range raftNode.clients {
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

// ***implement heartbeat***
// article: https://www.linkedin.com/pulse/raft-replication-happy-case-migo-lee-7mezc/?trackingId=0LrOxmbGRLaV5ZQ%2BooYMqQ%3D%3D&trk=article-ssr-frontend-pulse_little-text-block
// Periodic heartbeat
// The leader sends periodic heartbeat messages approximately every 300–500 milliseconds.
// Each heartbeat message contains:
// The current term (a logical clock value).
// The leader’s identity to confirm leadership status.
// Each valid term is associated with a single leader, ensuring that only one node holds leadership at any given time.

// func (r *RaftNode) startElection() {
// 	ticker := time.NewTicker(time.Millisecond * 300)
// 	defer ticker.Stop()
// }

// func (r *RaftNode) BecomeLeader() {

// }

// func (r *RaftNode) BecomeFollower() {

// }

// func (r *RaftNode) BecomeCandidate() {

// }

// A node can be of three states (Leader, Candidate, Follower)
// When a node first starts running, or when it crashes and recovers,
// it starts up in the follower state and awaits messages from other nodes.
// If it receives no messages from a leader or candidate for some period of time,
// the follower suspects that the leader is unavailable, and it may attempt to become leader itself.
// The timeout for detecting leader failure is randomised,
// to reduce the probability of several nodes becoming candidates concurrently and competing to become leader.
// When a node suspects the leader to have failed,
// it transitions to the candidate state, increments the term number, and starts a leader election in that term. !term increment
// During this election, if the node hears from another candidate or leader with a higher term,
// it moves back into the follower state.
// But if the election succeeds and it receives votes from a quorum of nodes, the candidate transitions to leader state.
// If not enough votes are received within some period of time, the election times out, and the candidate restarts the election with a higher term. !important
