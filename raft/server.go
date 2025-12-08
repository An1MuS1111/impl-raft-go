package server

import (
	"encoding/gob"
	"fmt"
	"impl-raft-go/proto"
	"log"
	"net"
	"os"
	"sync"
)

type ServerState uint64

const (
	Leader ServerState = iota
	Candidate
	Follower
)

type Log struct {
	Index   uint64 // first index has to be 1
	Term    uint64
	Command string
}

type RaftNode struct {
	proto.UnimplementedRaftServiceServer
	mu sync.Mutex

	//! persistence state
	currentTerm  uint64 // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor     uint64 // candidateId that received vote in current term (or null if none)
	log          []Log  // log entries; each entry contains command for state machine, and term when entry was received by leader (first index is 1)
	commitLength uint64

	//! read-only state
	id    uint64
	addr  net.Addr
	peers map[uint64]net.Addr

	//! volatile state
	currentRole ServerState
	// currentLeader unique
	// votesReceived uint64
	// sentLength    uint64
	// ackedLength   uint64

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
	if err := decoder.Decode(&r.commitLength); err != nil {
		return fmt.Errorf("failed to decode commitLength: %v", err)

	}

	return nil
}

func NewRaftNode(file *os.File, id uint64, addr net.Addr, addrs map[uint64]net.Addr) (*RaftNode, error) {
	// A node can be of three states (Leader, Candidate, Follower)
	// When a node first starts running, or when it crashes and recovers,
	// it starts up in the follower state and awaits messages from other nodes.
	raftNode := &RaftNode{
		id:          id,
		addr:        addr,
		currentRole: Follower,
		peers:       make(map[uint64]net.Addr),
	}

	raftNode.setPeers(addrs)

	// extract currentTerm, votedFor, log, and commitLength
	// from the disk for initialization
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats of file: %v", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to beginning: %v", err)
	}

	// if the size == 0 that means the the node is newly added to the quorum
	// other wise we think the node has crashed or
	// for some reason it was shutdown to apply some updates
	configSize := stat.Size()
	if configSize == 0 {
		log.Printf("starting a new server")

		raftNode.currentTerm = 0
		raftNode.votedFor = 0
		raftNode.log = make([]Log, 0)
		raftNode.commitLength = 0

		return raftNode, nil

	}

	// sync the node and retrieve the persisted states
	log.Printf("server restarted. config size: %v", configSize)
	log.Printf("retrieving persisted state")
	if err := raftNode.restoreState(file); err != nil {
		return nil, fmt.Errorf("failed to restore state from persistence: %v", err)
	}

	return raftNode, nil
}

func (r *RaftNode) setPeers(addrs map[uint64]net.Addr) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// remove self from peers
	delete(addrs, r.id)
	r.peers = addrs

	fmt.Println(r.id, r.addr, r.peers)
}

func (r *RaftNode) StartServer() error {

	// If it receives no messages from a leader or candidate for some period of time,
	// the follower suspects that the leader is unavailable, and it may attempt to become leader itself.

	return nil
}

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
