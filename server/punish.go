package server

import (
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/synchthia/systera-api/database"
	"github.com/synchthia/systera-api/stream"
	pb "github.com/synchthia/systera-api/systerapb"
	"golang.org/x/net/context"
)

func (s *grpcServer) GetPlayerPunish(ctx context.Context, e *pb.GetPlayerPunishRequest) (*pb.GetPlayerPunishResponse, error) {
	level := database.PunishLevel(e.FilterLevel)
	entries, err := s.mysql.GetPlayerPunishment(e.Uuid, level, e.IncludeExpired)

	var punishEntry []*pb.PunishEntry
	for _, entry := range entries {
		logrus.Debugf("To: %s / Level: %s / Reason: %s", entry.TargetPlayerName, pb.PunishLevel(entry.Level), entry.Reason)
		punishEntry = append(punishEntry, entry.ToProtobuf())
	}
	return &pb.GetPlayerPunishResponse{Entry: punishEntry}, err
}

func (s *grpcServer) SetPlayerPunish(ctx context.Context, e *pb.SetPlayerPunishRequest) (*pb.SetPlayerPunishResponse, error) {
	// if offline in the server, it should be input server name.
	entry := e.Entry
	level := database.PunishLevel(entry.Level)
	if e.Force || entry.PunishedTo.Uuid == "" {
		targetUUID, err := s.mysql.NameToUUID(entry.PunishedTo.Name)
		if err != nil {
			logrus.WithError(err).Errorf("[MojangAPI] Failed Lookup Player UUID: %s", entry.PunishedTo.Name)
			return &pb.SetPlayerPunishResponse{}, err
		} else if targetUUID == "" {
			response := &pb.SetPlayerPunishResponse{
				NoProfile: true,
			}
			return response, nil
		}
		entry.PunishedTo.Uuid = targetUUID
	}

	from := database.PlayerIdentity{
		UUID: entry.PunishedFrom.Uuid,
		Name: entry.PunishedFrom.Name,
	}

	to := database.PlayerIdentity{
		UUID: entry.PunishedTo.Uuid,
		Name: entry.PunishedTo.Name,
	}

	success, result, err := s.mysql.SetPlayerPunishment(e.Force, from, to, level, entry.Reason, entry.Date, entry.Expire)

	if err == nil && success {
		stream.PublishPunish(e.Remote, entry)
	}

	response := &pb.SetPlayerPunishResponse{
		NoProfile: result.NoProfile,
		Offline:   result.Offline,
		Duplicate: result.Duplicate,
		Cooldown:  result.Cooldown,
	}

	return response, err
}

func (s *grpcServer) UnBan(ctx context.Context, e *pb.UnBanRequest) (*pb.UnBanResponse, error) {
	targetUUID := e.Target.Uuid

	if e.Target.Name == "" && e.Target.Uuid == "" {
		return &pb.UnBanResponse{}, errors.New("target does not have name / uuid")
	}

	if targetUUID == "" {
		if u, err := s.mysql.NameToUUID(e.Target.Name); err != nil {
			return &pb.UnBanResponse{}, err
		} else {
			targetUUID = u
		}
	}

	return &pb.UnBanResponse{}, s.mysql.UnBan(targetUUID)
}
