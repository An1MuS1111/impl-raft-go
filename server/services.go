package server

import (
	"context"
	"impl-raft-go/proto"
)

func (r *RaftNode) RequestVote(ctx context.Context, args *proto.RequestVoteArgs) (*proto.RequestVoteReply, error) {

}

func (r *RaftNode) AppendEntries(ctx context.Context, args *proto.AppendEntriesArgs) (*proto.AppendEntriesReply, error) {

}
