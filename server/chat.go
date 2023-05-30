package server

import (
	"context"

	"github.com/synchthia/systera-api/database"
	"github.com/synchthia/systera-api/status"
	"github.com/synchthia/systera-api/stream"
	"github.com/synchthia/systera-api/systerapb"
)

func (s *grpcServer) Chat(ctx context.Context, e *systerapb.ChatRequest) (*systerapb.Empty, error) {
	return &systerapb.Empty{}, stream.PublishChat(e.GetEntry())
}

func (s *grpcServer) AddChatIgnore(ctx context.Context, e *systerapb.AddChatIgnoreRequest) (*systerapb.ChatIgnoreResponse, error) {
	if e.Target.Uuid == "" {
		if res, err := s.mysql.GetIdentityByName(e.Target.Name); err != nil {
			if err == status.ErrPlayerNotFound.Error {
				return &systerapb.ChatIgnoreResponse{Result: systerapb.CallResult_NOT_FOUND}, nil
			} else {
				return &systerapb.ChatIgnoreResponse{}, err
			}
		} else {
			e.Target = res.ToProtobuf()
		}
	}

	err := s.mysql.AddIgnore(e.Uuid, &database.PlayerIdentity{UUID: e.Target.Uuid, Name: e.Target.Name})
	if err == status.ErrPlayerAlreadyExists.Error {
		return &systerapb.ChatIgnoreResponse{
			Result: systerapb.CallResult_DUPLICATED,
		}, nil
	} else {
		return &systerapb.ChatIgnoreResponse{
			Result:   systerapb.CallResult_SUCCESS,
			Identity: e.Target,
		}, err
	}
}

func (s *grpcServer) RemoveChatIgnore(ctx context.Context, e *systerapb.RemoveChatIgnoreRequest) (*systerapb.ChatIgnoreResponse, error) {
	if e.Target.Uuid == "" {
		if res, err := s.mysql.GetIdentityByName(e.Target.Name); err != nil {
			if err == status.ErrPlayerNotFound.Error {
				return &systerapb.ChatIgnoreResponse{Result: systerapb.CallResult_NOT_FOUND}, nil
			} else {
				return &systerapb.ChatIgnoreResponse{}, err
			}
		} else {
			e.Target = res.ToProtobuf()
		}
	}

	err := s.mysql.RemoveIgnore(e.Uuid, &database.PlayerIdentity{UUID: e.Target.Uuid, Name: e.Target.Name})
	return &systerapb.ChatIgnoreResponse{
		Result:   systerapb.CallResult_SUCCESS,
		Identity: e.Target,
	}, err
}
