package raft

import (
	"encoding/gob"
	"fmt"
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

type Message struct {
}

type Log struct {
	term uint64
	msg  Message
}

type RaftNode struct {
	mu sync.Mutex

	//! persistence state
	currentTerm  uint64
	votedFor     uint64
	log          []Log
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

// func (r *RaftNode) generateUniqueId() uint64 {

// 	for {
// 		uniqueID := rand.Uint64N(100)
// 		if _, ok := r.nodes[uniqueID]; ok {
// 			continue
// 		}
// 		return uniqueID
// 	}
// }

func (r *RaftNode) SetPeers(peers map[uint64]net.Addr) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// remove self from peers
	delete(peers, r.id)
	r.peers = peers

	fmt.Println(r.id, r.addr, r.peers)
}

func NewRaftNode(file *os.File, id uint64, addr net.Addr) (*RaftNode, error) {

	raftNode := &RaftNode{
		id:          id,
		addr:        addr,
		currentRole: Follower,
		peers:       make(map[uint64]net.Addr),
	}

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

	decoder := gob.NewDecoder(file)

	if err := decoder.Decode(&raftNode.currentTerm); err != nil {
		return nil, fmt.Errorf("failed to decode currentTerm: %v", err)
	}
	if err := decoder.Decode(&raftNode.votedFor); err != nil {
		return nil, fmt.Errorf("failed to decode votedFor: %v", err)
	}
	if err := decoder.Decode(&raftNode.log); err != nil {
		return nil, fmt.Errorf("failed to decode log: %v", err)
	}
	if err := decoder.Decode(&raftNode.commitLength); err != nil {
		return nil, fmt.Errorf("failed to decode commitLength: %v", err)

	}
	// log.Printf("currentTerm: %v", raftNode.currentTerm)
	// log.Printf("votedFor: %v", raftNode.votedFor)
	// log.Printf("log: %v", raftNode.log)
	// log.Printf("commitLength: %v", raftNode.commitLength)

	return raftNode, nil
}

func (r *RaftNode) BecomeLeader() {

}

func (r *RaftNode) BecomeFollower() {

}

func (r *RaftNode) BecomeCandidate() {

}

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
// If not enough votes are received within some period of time, the election times out, and the candidate restarts the election with a higher term.
