package server

import (
	"context"
	"impl-raft-go/proto"
	"log"
)

// func (r *RaftNode) RequestVote(ctx context.Context, args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {

// }

// func (r *RaftNode) AppendEntries(ctx context.Context, args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {

// }

func (r *RaftNode) HealthCheck(_ context.Context, in *proto.HealthCheckArgs) (*proto.HealthCheckReply, error) {
	log.Println("Received from: ", in.Ping)
	return &proto.HealthCheckReply{Pong: "pong"}, nil
}
