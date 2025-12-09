package server

import (
	"context"
	"impl-raft-go/proto"
	"log"
)

// func (r *RaftNode) RequestVote(ctx context.Context, args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {

// }

func (r *RaftNode) AppendEntries(ctx context.Context, args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {

	select {
	case <-ctx.Done():
		return &proto.AppendEntriesReply{Term: r.currentTerm, Success: false}, nil

	default:
		if args.Term < r.currentTerm {
			return &proto.AppendEntriesReply{Term: r.currentTerm, Success: false}, nil
		}

	}
	return &proto.AppendEntriesReply{Term: r.currentTerm, Success: false}, nil
}

func (r *RaftNode) HealthCheck(_ context.Context, in *proto.HealthCheckArgs) (*proto.HealthCheckReply, error) {
	log.Println("Received from: ", in.Ping)
	return &proto.HealthCheckReply{Pong: "pong"}, nil
}
