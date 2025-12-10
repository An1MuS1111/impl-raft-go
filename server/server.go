package server

import (
	"context"
	"fmt"
	"impl-raft-go/proto"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
)

// start Server -> wait for heartbeat(append entries rpc with empty entries []) ?
// else convert to candidate and start election
func (r *RaftNode) StartServer() error {

	addr := r.addr.String()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterRaftServiceServer(grpcServer, r)

	log.Printf("server no.%v listening at: %v", r.id, r.addr)
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to server grpc: %v", err)
	}

	// If it receives no messages from a leader or candidate for some period of time,
	// the follower suspects that the leader is unavailable, and it may attempt to become leader itself.

	return nil
}

func (r *RaftNode) startElection() {
	r.mu.Lock()
	r.currentRole = Candidate
	r.currentTerm++
	r.votedFor = r.id
	log.Printf("[%v] attempting an election at term %v", r.id, r.currentTerm)
	currentTerm := r.currentTerm
	r.mu.Unlock()

	for _, client := range r.clients {
		go func(client proto.RaftServiceClient, currentTerm uint64) {
			voteGranted := r.callRequestVote(client)
			if !voteGranted {
				return
			}
			// ....tally the votes
			r.mu.Lock()
			// If a candidate receives votes from a majority of servers for the same term, it becomes leader.
			// If a candidate receives vote from a peer, it means that the peer has acknowledged the candidate as a legitimate leader for that term.
			// So, the candidate should count the vote.
			// If the candidate receives votes from a majority of servers, it transitions to leader.
			// If another server claims to be leader (AppendEntries RPC) or
			// if a candidate with a higher term is discovered, the candidate reverts to follower state.
			r.mu.Unlock()

		}(client, currentTerm)
	}
}

// [incomplete]
func (r *RaftNode) callRequestVote(client proto.RaftServiceClient) bool {

}

// [incomplete]
// runElectionTimer handles election timeouts and starts elections.
func (r *RaftNode) runElectionTimer() {
	timeout := time.Duration(150+rand.Intn(150)) * time.Millisecond // Random timeout (150-300ms)
	timer := time.NewTimer(timeout)                                 // Create timer
	defer timer.Stop()                                              // Ensure timer is stopped on exit
	for {
		select {

		case <-r.electionReset: // Received heartbeat or vote
			timer.Reset(timeout) // Reset timer with new random duration
		case <-timer.C: // Timeout expired
			r.mu.Lock() // Lock to check state
			if r.currentRole == Leader {
				r.mu.Unlock() // Leader doesn't start elections
				continue
			}
			r.mu.Unlock()
			r.startElection() // Start election as Candidate
			// timeout = time.Duration(150+rand.Intn(150)) * time.Millisecond // New random timeout
			// timer.Reset(timeout)                                           // Reset timer
		}
	}
}

func convertToProtoEntries(entries []Log) []*proto.LogEntry {
	protoEntries := make([]*proto.LogEntry, len(entries))
	for i, entry := range entries {
		protoEntries[i] = &proto.LogEntry{
			Term:    entry.Term,
			Index:   entry.Index,
			Command: entry.Command,
		}
	}
	return protoEntries
}

// [incomplete]
func (r *RaftNode) StartHeartBeat() {
	ticker := time.NewTicker(time.Millisecond * 50) // 50 ms interval per heartbeat
	defer ticker.Stop()
	switch r.currentRole {
	case Leader:
		for {
			<-ticker.C

			for _, client := range r.clients {

				r.mu.Lock()
				args := &proto.AppendEntriesArgs{
					Term:         0,                              // Leader's term
					LeaderId:     0,                              // Leader's ID
					PrevLogIndex: 0,                              // Previous log index
					PrevLogTerm:  0,                              // Previous log term
					Entries:      convertToProtoEntries([]Log{}), // Convert entries
					LeaderCommit: 0,                              // Leader's commit index
				}
				r.mu.Unlock()
				_, err := client.AppendEntries(context.Background(), args)
				// reply, err := client.AppendEntries(context.Background(), args)
				if err != nil {
					return // Failed to contact peer
				}
			}
		}

	}

}
