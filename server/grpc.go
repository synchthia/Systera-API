package server

import (
	"sync"

	"github.com/sirupsen/logrus"

	"golang.org/x/net/context"

	"github.com/synchthia/systera-api/database"
	"github.com/synchthia/systera-api/stream"
	"github.com/synchthia/systera-api/systerapb"
	pb "github.com/synchthia/systera-api/systerapb"
	"google.golang.org/grpc"
)

type Server interface {
	Announce(target, msg string)
	Dispatch(target, cmd string)

	InitPlayerProfile(playerUUID, playerName, ipAddress, hostname string) (*systerapb.PlayerEntry, error)
	FetchPlayerProfile(playerUUID string) (string, string, map[string]bool, error)
	FetchPlayerProfileByName(playerName string) (string, string, map[string]bool, error)

	SetPlayerGroups(playerUUID string, groups []string) error
	SetPlayerServer(playerUUID, serverName string) error
	SetPlayerSettings(playerUUID, settings *systerapb.PlayerSettings) error

	AltLookup(playerUUID string) ([]pb.AltLookupEntry, error)

	GetPlayerPunish(playerUUID string, filterLevel pb.PunishLevel, includeExpired bool) []pb.PunishEntry
	SetPlayerPunish(force bool, entry pb.PunishEntry) (bool, bool, bool, bool, error)

	Report(from pb.PlayerIdentity, to pb.PlayerIdentity, message string) error

	FetchGroups(serverName string) ([]pb.GroupEntry, error)

	CreateGroup(groups pb.GroupEntry) error
	RemoveGroup(groupName string) error

	AddPermission(groupName, target, permission []string) error
	RemovePermission(groupName, target, permission []string) error
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
	r, err := database.InitPlayerProfile(e.Uuid, e.Name, e.IpAddress, e.Hostname)
	return &pb.InitPlayerProfileResponse{Entry: r.ToProtobuf()}, err
}

func (s *grpcServer) FetchPlayerProfile(ctx context.Context, e *pb.FetchPlayerProfileRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := database.FindPlayer(e.Uuid)
	return &pb.FetchPlayerProfileResponse{
		Entry: playerData.ToProtobuf(),
	}, err
}

func (s *grpcServer) FetchPlayerProfileByName(ctx context.Context, e *pb.FetchPlayerProfileByNameRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := database.FindPlayerByName(e.Name)
	return &pb.FetchPlayerProfileResponse{
		Entry: playerData.ToProtobuf(),
	}, err
}

func (s *grpcServer) SetPlayerGroups(ctx context.Context, e *pb.SetPlayerGroupsRequest) (*pb.Empty, error) {
	err := database.SetPlayerGroups(e.Uuid, e.Groups)
	playerData, err := database.FindPlayer(e.Uuid)

	if err != nil {
		return &pb.Empty{}, err
	}

	if playerData.Stats.CurrentServer != "" {
		stream.PublishPlayerGroups(playerData.Stats.CurrentServer,
			&pb.PlayerEntry{
				Uuid:   e.Uuid,
				Groups: e.Groups,
			},
		)
	}
	return &pb.Empty{}, err
}

func (s *grpcServer) SetPlayerServer(ctx context.Context, e *pb.SetPlayerServerRequest) (*pb.Empty, error) {
	err := database.SetPlayerServer(e.Uuid, e.ServerName)
	return &pb.Empty{}, err
}

func (s *grpcServer) RemovePlayerServer(ctx context.Context, e *pb.RemovePlayerServerRequest) (*pb.Empty, error) {
	playerData, err := database.FindPlayer(e.Uuid)

	if e.ServerName == playerData.Stats.CurrentServer {
		err = database.SetPlayerServer(e.Uuid, "")
	}

	return &pb.Empty{}, err
}

func (s *grpcServer) SetPlayerSettings(ctx context.Context, e *pb.SetPlayerSettingsRequest) (*pb.Empty, error) {
	err := database.SetPlayerSettings(e.Uuid, (&database.PlayerSettings{}).FromProtobuf(e.Settings))
	return &pb.Empty{}, err
}

func (s *grpcServer) AltLookup(ctx context.Context, e *pb.AltLookupRequest) (*pb.AltLookupResponse, error) {
	result, err := database.AltLookup(e.PlayerUuid)
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
				FirstSeen: a.FirstSeen,
				LastSeen:  a.LastSeen,
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

func (s *grpcServer) GetPlayerPunish(ctx context.Context, e *pb.GetPlayerPunishRequest) (*pb.GetPlayerPunishResponse, error) {
	level := database.PunishLevel(e.FilterLevel)
	entries, err := database.GetPlayerPunishment(e.Uuid, level, e.IncludeExpired)

	var punishEntry []*pb.PunishEntry
	for _, entry := range entries {
		logrus.Debugf("To: %s / Level: %s / Reason: %s", entry.PunishedTo.Name, pb.PunishLevel(entry.Level), entry.Reason)
		punishEntry = append(punishEntry, entry.ToProtobuf())
	}
	return &pb.GetPlayerPunishResponse{Entry: punishEntry}, err
}

func (s *grpcServer) SetPlayerPunish(ctx context.Context, e *pb.SetPlayerPunishRequest) (*pb.SetPlayerPunishResponse, error) {
	// if offline in the server, it should be input server name.
	entry := e.Entry
	level := database.PunishLevel(entry.Level)
	if e.Force || entry.PunishedTo.Uuid == "" {
		targetUUID, err := database.NameToUUID(entry.PunishedTo.Name)
		if err != nil {
			if err.Error() == "unable to GetAPIProfile: user not found" {
				response := &pb.SetPlayerPunishResponse{
					NoProfile: true,
				}
				return response, nil
			}

			logrus.WithError(err).Errorf("[MojangAPI] Failed Lookup Player UUID: %s", entry.PunishedTo.Name)
			return &pb.SetPlayerPunishResponse{}, err
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

	success, result, err := database.SetPlayerPunishment(e.Force, from, to, level, entry.Reason, entry.Date, entry.Expire)

	if err == nil && success {
		stream.PublishPunish(entry)
		logrus.Printf("%s", entry)
	}

	response := &pb.SetPlayerPunishResponse{
		NoProfile: result.NoProfile,
		Offline:   result.Offline,
		Duplicate: result.Duplicate,
		Cooldown:  result.Cooldown,
	}

	return response, err
}

func (s *grpcServer) Report(ctx context.Context, e *pb.ReportRequest) (*pb.ReportResponse, error) {
	from := database.PlayerIdentity{
		UUID: e.From.Uuid,
		Name: e.From.Name,
	}
	to := database.PlayerIdentity{
		UUID: e.To.Uuid,
		Name: e.To.Name,
	}

	entry, err := database.SetReport(from, to, e.Message)
	if err == nil {
		stream.PublishReport(
			&pb.ReportEntry{
				From: &pb.PlayerIdentity{
					Uuid: entry.ReportedFrom.UUID,
					Name: entry.ReportedFrom.Name,
				},
				To: &pb.PlayerIdentity{
					Uuid: entry.ReportedTo.UUID,
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
	groups, err := database.GetAllGroup()

	var allGroups []*pb.GroupEntry
	for _, group := range groups {
		allGroups = append(allGroups, group.ToProtobuf(e.ServerName))
	}

	return &pb.FetchGroupsResponse{Groups: allGroups}, err
}

func (s *grpcServer) CreateGroup(ctx context.Context, e *pb.CreateGroupRequest) (*pb.Empty, error) {
	d := database.GroupData{}
	d.Name = e.GroupName
	d.Prefix = e.GroupPrefix

	var dbPerms map[string][]string
	for _, v := range e.PermsEntry {
		dbPerms[v.ServerName] = v.Permissions
	}
	d.Permissions = dbPerms

	err := database.CreateGroup(d)
	if err != nil {
		return &pb.Empty{}, err
	}

	if dbPerms != nil {
		for sv := range dbPerms {
			stream.PublishGroup(d.ToProtobuf(sv))
		}
	} else {
		stream.PublishGroup(d.ToProtobuf(""))
	}

	return &pb.Empty{}, err
}

func (s *grpcServer) RemoveGroup(ctx context.Context, e *pb.RemoveGroupRequest) (*pb.Empty, error) {
	err := database.RemoveGroup(e.GroupName)
	return &pb.Empty{}, err
}

func (s *grpcServer) AddPermission(ctx context.Context, e *pb.AddPermissionRequest) (*pb.Empty, error) {
	err := database.AddPermission(e.GroupName, e.Target, e.Permissions)
	data, err := database.GetGroupData(e.GroupName)
	stream.PublishPerms(e.Target, data.ToProtobuf(e.Target))

	return &pb.Empty{}, err
}

func (s *grpcServer) RemovePermission(ctx context.Context, e *pb.RemovePermissionRequest) (*pb.Empty, error) {
	err := database.RemovePermission(e.GroupName, e.Target, e.Permissions)
	data, err := database.GetGroupData(e.GroupName)
	stream.PublishPerms(e.Target, data.ToProtobuf(e.Target))

	return &pb.Empty{}, err
}
