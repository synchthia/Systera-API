package server

import (
	"context"

	"github.com/synchthia/systera-api/stream"
	"github.com/synchthia/systera-api/systerapb"
)

func (s *grpcServer) Chat(ctx context.Context, e *systerapb.ChatRequest) (*systerapb.Empty, error) {
	return &systerapb.Empty{}, stream.PublishChat(e.GetEntry())
}

func (s *grpcServer) AddChatIgnore(ctx context.Context, e *systerapb.AddChatIgnoreRequest) (*systerapb.Empty, error) {
	return &systerapb.Empty{}, nil
}

func (s *grpcServer) RemoveChatIgnore(ctx context.Context, e *systerapb.RemoveChatIgnoreRequest) (*systerapb.Empty, error) {
	return &systerapb.Empty{}, nil
}
