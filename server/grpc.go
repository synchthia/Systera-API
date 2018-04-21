package server

import (
	"sync"

	"github.com/sirupsen/logrus"

	"golang.org/x/net/context"

	"gitlab.com/Startail/Systera-API/database"
	"gitlab.com/Startail/Systera-API/stream"
	pb "gitlab.com/Startail/Systera-API/systerapb"
	"gitlab.com/Startail/Systera-API/util"
	"google.golang.org/grpc"
)

type Server interface {
	Announce(target, msg string)
	Dispatch(target, cmd string)

	InitPlayerProfile(playerUUID, playerName, ipAddress, hostname string) (bool, error)
	FetchPlayerProfile(playerUUID string) (string, string, map[string]bool, error)
	FetchPlayerProfileByName(playerName string) (string, string, map[string]bool, error)

	SetPlayerGroups(playerUUID string, groups []string) error
	SetPlayerServer(playerUUID, serverName string) error
	SetPlayerSettings(playerUUID, key string, value bool) error

	GetPlayerPunish(playerUUID string, filterLevel pb.PunishLevel, includeExpired bool) []pb.PunishEntry
	SetPlayerPunish(force bool, entry pb.PunishEntry) (bool, bool, bool, bool, error)

	Report(from pb.PlayerData, to pb.PlayerData, message string) error

	FetchGroups(serverName string) []string
}

type grpcServer struct {
	server Server
	mu     sync.RWMutex
}

func NewServer() *grpcServer {
	return &grpcServer{}
}

func NewGRPCServer() *grpc.Server {
	server := grpc.NewServer()
	pb.RegisterSysteraServer(server, NewServer())
	return server
}

func (s *grpcServer) Announce(ctx context.Context, e *pb.AnnounceRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := stream.PublishAnnounce(e.Target, e.Message)
	return &pb.Empty{}, err
}

func (s *grpcServer) Dispatch(ctx context.Context, e *pb.DispatchRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := stream.PublishCommand(e.Target, e.Cmd)
	return &pb.Empty{}, err
}

func (s *grpcServer) InitPlayerProfile(ctx context.Context, e *pb.InitPlayerProfileRequest) (*pb.InitPlayerProfileResponse, error) {
	hasProfile := false
	count, err := database.InitPlayerProfile(e.PlayerUUID, e.PlayerName, e.PlayerIPAddress, e.PlayerHostname)

	if count != 0 {
		hasProfile = true
	}
	return &pb.InitPlayerProfileResponse{HasProfile: hasProfile}, err
}

func (s *grpcServer) FetchPlayerProfile(ctx context.Context, e *pb.FetchPlayerProfileRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := database.FindPlayer(e.PlayerUUID)
	return &pb.FetchPlayerProfileResponse{
		Entry: s.PlayerData_DBtoPB(playerData),
	}, err
}

func (s *grpcServer) FetchPlayerProfileByName(ctx context.Context, e *pb.FetchPlayerProfileByNameRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := database.FindPlayerByName(e.PlayerName)
	return &pb.FetchPlayerProfileResponse{
		Entry: s.PlayerData_DBtoPB(playerData),
	}, err
}

func (s *grpcServer) SetPlayerGroups(ctx context.Context, e *pb.SetPlayerGroupsRequest) (*pb.Empty, error) {
	err := database.SetPlayerGroups(e.PlayerUUID, e.Groups)
	playerData, err := database.FindPlayer(e.PlayerUUID)

	if err != nil {
		return &pb.Empty{}, err
	}

	if playerData.Stats.CurrentServer != "" {
		stream.PublishPlayerGroups(playerData.Stats.CurrentServer,
			&pb.PlayerEntry{
				PlayerUUID: e.PlayerUUID,
				Groups:     e.Groups,
			},
		)
	}
	return &pb.Empty{}, err
}

func (s *grpcServer) SetPlayerServer(ctx context.Context, e *pb.SetPlayerServerRequest) (*pb.Empty, error) {
	err := database.SetPlayerServer(e.PlayerUUID, e.ServerName)
	return &pb.Empty{}, err
}

func (s *grpcServer) RemovePlayerServer(ctx context.Context, e *pb.RemovePlayerServerRequest) (*pb.Empty, error) {
	playerData, err := database.FindPlayer(e.PlayerUUID)

	if e.ServerName == playerData.Stats.CurrentServer {
		err = database.SetPlayerServer(e.PlayerUUID, "")
	}

	return &pb.Empty{}, err
}

func (s *grpcServer) SetPlayerSettings(ctx context.Context, e *pb.SetPlayerSettingsRequest) (*pb.Empty, error) {
	err := database.PushPlayerSettings(e.PlayerUUID, e.Key, e.Value)
	return &pb.Empty{}, err
}

func (s *grpcServer) GetPlayerPunish(ctx context.Context, e *pb.GetPlayerPunishRequest) (*pb.GetPlayerPunishResponse, error) {
	level := database.PunishLevel(e.FilterLevel)
	entries, err := database.GetPlayerPunishment(e.PlayerUUID, level, e.IncludeExpired)

	var punishEntry []*pb.PunishEntry
	for _, entry := range entries {
		logrus.Debugf("To: %s / Level: %s / Reason: %s", entry.PunishedTo.Name, pb.PunishLevel(entry.Level), entry.Reason)
		punishEntry = append(punishEntry, &pb.PunishEntry{
			Available: entry.Available,
			Level:     pb.PunishLevel(entry.Level),
			Reason:    entry.Reason,
			Date:      entry.Date,
			Expire:    entry.Expire,

			PunishedFrom: &pb.PlayerData{UUID: entry.PunishedFrom.UUID, Name: entry.PunishedFrom.Name},
			PunishedTo:   &pb.PlayerData{UUID: entry.PunishedTo.UUID, Name: entry.PunishedTo.Name},
		})
	}
	return &pb.GetPlayerPunishResponse{Entry: punishEntry}, err
}

func (s *grpcServer) SetPlayerPunish(ctx context.Context, e *pb.SetPlayerPunishRequest) (*pb.SetPlayerPunishResponse, error) {
	// if offline in the server, it should be input server name.
	entry := e.Entry
	level := database.PunishLevel(entry.Level)
	if e.Force || entry.PunishedTo.UUID == "" {
		targetUUID, err := database.NameToUUID(entry.PunishedTo.Name)
		if err != nil {
			logrus.WithError(err).Errorf("[MojangAPI] Failed Lookup Player UUID: %s", entry.PunishedTo.Name)
			return &pb.SetPlayerPunishResponse{}, err
		}
		entry.PunishedTo.UUID = targetUUID
	}

	from := database.PlayerIdentity{
		UUID: entry.PunishedFrom.UUID,
		Name: entry.PunishedFrom.Name,
	}

	to := database.PlayerIdentity{
		UUID: entry.PunishedTo.UUID,
		Name: entry.PunishedTo.Name,
	}

	success, result, err := database.SetPlayerPunishment(e.Force, from, to, level, entry.Reason, entry.Date, entry.Expire)

	if err == nil && success {
		stream.PublishPunish(entry)
		logrus.Printf("%s", entry)
	}

	response := &pb.SetPlayerPunishResponse{
		Noprofile: result.NoProfile,
		Offline:   result.Offline,
		Duplicate: result.Duplicate,
		Cooldown:  result.Cooldown,
	}

	return response, err
}

func (s *grpcServer) Report(ctx context.Context, e *pb.ReportRequest) (*pb.ReportResponse, error) {
	from := database.PlayerIdentity{
		UUID: e.From.UUID,
		Name: e.From.Name,
	}
	to := database.PlayerIdentity{
		UUID: e.To.UUID,
		Name: e.To.Name,
	}

	entry, err := database.SetReport(from, to, e.Message)
	if err == nil {
		stream.PublishReport(
			&pb.ReportEntry{
				From: &pb.PlayerData{
					UUID: entry.ReportedFrom.UUID,
					Name: entry.ReportedFrom.Name,
				},
				To: &pb.PlayerData{
					UUID: entry.ReportedTo.UUID,
					Name: entry.ReportedTo.Name,
				},
				Message: entry.Message,
				Date:    entry.Date,
				Server:  entry.Server,
			},
		)
	}

	return &pb.ReportResponse{}, err
}

func (s *grpcServer) FetchGroups(ctx context.Context, e *pb.FetchGroupsRequest) (*pb.FetchGroupsResponse, error) {
	groups, err := database.FindGroupData()
	var allGroups []*pb.GroupEntry
	for _, value := range groups {
		allGroups = append(allGroups, &pb.GroupEntry{
			GroupName:   value.Name,
			GroupPrefix: value.Prefix,
			GlobalPerms: value.Permissions["GLOBAL"],
			ServerPerms: value.Permissions[e.ServerName],
		})
	}
	return &pb.FetchGroupsResponse{Groups: allGroups}, err
}

func (s *grpcServer) PlayerData_DBtoPB(dbEntry database.PlayerData) *pb.PlayerEntry {
	return &pb.PlayerEntry{
		PlayerUUID: dbEntry.UUID,
		PlayerName: dbEntry.Name,
		Groups:     dbEntry.Groups,
		Settings:   util.StructToBoolMap(dbEntry.Settings),
	}
}
