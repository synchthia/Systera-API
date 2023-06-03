package server

import (
	"github.com/synchthia/systera-api/database"
	sts "github.com/synchthia/systera-api/status"
	"github.com/synchthia/systera-api/stream"
	pb "github.com/synchthia/systera-api/systerapb"
	"golang.org/x/net/context"
)

func (s *grpcServer) GetPlayerIdentityByName(ctx context.Context, e *pb.GetPlayerIdentityByNameRequest) (*pb.GetPlayerIdentityByNameResponse, error) {
	r, err := s.mysql.GetIdentityByName(e.Name)

	if err != nil {
		if err == sts.ErrPlayerNotFound.Error {
			// If just not exists
			return &pb.GetPlayerIdentityByNameResponse{
				Exists: false,
			}, sts.ErrPlayerNotFound.ToGrpcError().Err()
		} else {
			// or If has error
			return &pb.GetPlayerIdentityByNameResponse{
				Exists: false,
			}, err
		}
	} else {
		return &pb.GetPlayerIdentityByNameResponse{
			Identity: r.ToProtobuf(),
			// Return true when non-nil / false when nil
			Exists: true,
		}, nil
	}
}

func (s *grpcServer) InitPlayerProfile(ctx context.Context, e *pb.InitPlayerProfileRequest) (*pb.InitPlayerProfileResponse, error) {
	r, err := s.mysql.InitPlayerProfile(e.Uuid, e.Name, e.IpAddress, e.Hostname)
	return &pb.InitPlayerProfileResponse{Entry: r.ToProtobuf()}, err
}

func (s *grpcServer) FetchPlayerProfile(ctx context.Context, e *pb.FetchPlayerProfileRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := s.mysql.FindPlayer(e.Uuid)
	return &pb.FetchPlayerProfileResponse{
		Entry: playerData.ToProtobuf(),
	}, err
}

func (s *grpcServer) FetchPlayerProfileByName(ctx context.Context, e *pb.FetchPlayerProfileByNameRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := s.mysql.FindPlayerByName(e.Name)
	return &pb.FetchPlayerProfileResponse{
		Entry: playerData.ToProtobuf(),
	}, err
}

func (s *grpcServer) SetPlayerGroups(ctx context.Context, e *pb.SetPlayerGroupsRequest) (*pb.Empty, error) {
	if err := s.mysql.SetPlayerGroups(e.Uuid, e.Groups); err != nil {
		return &pb.Empty{}, err
	}

	playerData, err := s.mysql.FindPlayer(e.Uuid)

	if err != nil {
		return &pb.Empty{}, err
	}

	if playerData.CurrentServer != "" {
		stream.PublishPlayerGroups(playerData.CurrentServer,
			&pb.PlayerEntry{
				Uuid:   e.Uuid,
				Groups: e.Groups,
			},
		)
	}
	return &pb.Empty{}, err
}

func (s *grpcServer) SetPlayerServer(ctx context.Context, e *pb.SetPlayerServerRequest) (*pb.Empty, error) {
	err := s.mysql.SetPlayerServer(false, e.Uuid, e.ServerName)
	return &pb.Empty{}, err
}

func (s *grpcServer) RemovePlayerServer(ctx context.Context, e *pb.RemovePlayerServerRequest) (*pb.Empty, error) {
	err := s.mysql.SetPlayerServer(true, e.Uuid, e.ServerName)
	return &pb.Empty{}, err
}

func (s *grpcServer) SetPlayerSettings(ctx context.Context, e *pb.SetPlayerSettingsRequest) (*pb.Empty, error) {
	err := s.mysql.SetPlayerSettings(e.Uuid, (&database.PlayerSettings{}).FromProtobuf(e.Settings))
	return &pb.Empty{}, err
}

func (s *grpcServer) AltLookup(ctx context.Context, e *pb.AltLookupRequest) (*pb.AltLookupResponse, error) {
	result, err := s.mysql.AltLookup(e.PlayerUuid)
	if err != nil {
		return &pb.AltLookupResponse{}, err
	}

	var entry []*pb.AltLookupEntry
	for _, r := range result {
		var aEntry []*pb.AddressesEntry
		for _, a := range r.Addresses {
			aEntry = append(aEntry, &pb.AddressesEntry{
				Address:   a.Address,
				Hostname:  a.Hostname,
				FirstSeen: a.FirstSeen.UnixMilli(),
				LastSeen:  a.LastSeen.UnixMilli(),
			})
		}

		entry = append(entry, &pb.AltLookupEntry{
			Uuid:      r.UUID,
			Name:      r.Name,
			Addresses: aEntry,
		})
	}

	return &pb.AltLookupResponse{Entries: entry}, err
}
