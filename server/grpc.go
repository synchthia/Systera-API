package server

import (
	"log"
	"strings"
	"sync"

	"golang.org/x/net/context"

	pb "gitlab.com/Startail/Systera-API/apipb"
	"gitlab.com/Startail/Systera-API/database"
	"google.golang.org/grpc"
)

type Server interface {
	Announce(target, msg string)
	QuitStream(name string)

	InitPlayerProfile(playerUUID, playerName, ipAddress string) (bool, error)
	FetchPlayerProfile(playerUUID string) (string, string, map[string]bool, error)
	FetchPlayerProfileByName(playerName string) (string, string, map[string]bool, error)

	SetPlayerServer(playerUUID, serverName string) error
	SetPlayerSettings(playerUUID, key string, value bool) error

	GetPlayerPunish(playerUUID string, filterLevel pb.PunishLevel, includeExpired bool) []pb.PunishEntry
	SetPlayerPunish(remote bool, force bool, entry pb.PunishEntry) (bool, bool, bool, bool, error)

	FetchGroups(serverName string) []string
}

type grpcServer struct {
	server   Server
	mu       sync.RWMutex
	asrChans map[chan pb.ActionStreamResponse]struct{}
	psrChans map[chan pb.PunishStreamResponse]struct{}
}

func NewServer() *grpcServer {
	return &grpcServer{
		asrChans: make(map[chan pb.ActionStreamResponse]struct{}),
		psrChans: make(map[chan pb.PunishStreamResponse]struct{}),
	}
}

func NewGRPCServer() *grpc.Server {
	server := grpc.NewServer()
	pb.RegisterSysteraServer(server, NewServer())
	return server
}

func (s *grpcServer) Ping(ctx context.Context, e *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

func (s *grpcServer) ActionStream(r *pb.StreamRequest, as pb.Systera_ActionStreamServer) error {
	ech := make(chan pb.ActionStreamResponse)
	s.mu.Lock()
	s.asrChans[ech] = struct{}{}
	s.mu.Unlock()
	log.Printf("[Action/ST]: Added New Watcher: %s (%v)", r.Name, ech)
	log.Printf("[Action/ST]: -> Currently Watcher: %d", len(s.asrChans))

	defer func() {
		s.mu.Lock()
		delete(s.asrChans, ech)
		s.mu.Unlock()
		close(ech)
		log.Printf("[Action/ST]: Deleted Watcher: %v", ech)
	}()

	for e := range ech {
		if e.Target != "GLOBAL" && !strings.HasPrefix(e.Target, r.Name) {
			continue
		}

		log.Printf("[Action/ST]: Requested (%s / Target: %s) [%s >> %s]", e.Type, e.Target, r.Name, e.Cmd)
		if e.Type == pb.StreamType_QUIT {
			return nil
		}

		err := as.Send(&e)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *grpcServer) PunishStream(r *pb.StreamRequest, ps pb.Systera_PunishStreamServer) error {
	ech := make(chan pb.PunishStreamResponse)
	s.mu.Lock()
	s.psrChans[ech] = struct{}{}
	s.mu.Unlock()
	log.Printf("[Punish/ST]: Added New Watcher: %s", r.Name)
	log.Printf("[Punish/ST]: -> Currently Watcher: %d", len(s.psrChans))

	defer func() {
		s.mu.Lock()
		delete(s.psrChans, ech)
		s.mu.Unlock()
		close(ech)
		log.Printf("[Punish/ST]: Deleted Watcher: %v", ech)
	}()

	for e := range ech {
		if e.Target != "GLOBAL" && !strings.HasPrefix(e.Target, r.Name) {
			continue
		}

		log.Printf("[Punish/ST]: Requested (%s / Target: %s) [%s]", e.Type, e.Target, r.Name)
		if e.Type == pb.StreamType_QUIT {
			return nil
		}

		err := ps.Send(&e)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *grpcServer) Announce(ctx context.Context, e *pb.AnnounceRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for c := range s.asrChans {
		c <- pb.ActionStreamResponse{Type: pb.StreamType_DISPATCH, Target: e.Target, Cmd: e.Message}
	}
	return &pb.Empty{}, nil
}

func (s *grpcServer) QuitStream(ctx context.Context, e *pb.QuitStreamRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for c := range s.asrChans {
		c <- pb.ActionStreamResponse{Type: pb.StreamType_QUIT, Target: e.Name}
	}
	return &pb.Empty{}, nil
}

func (s *grpcServer) InitPlayerProfile(ctx context.Context, e *pb.InitPlayerProfileRequest) (*pb.InitPlayerProfileResponse, error) {
	h, err := database.InitPlayerProfile(e.PlayerUUID, e.PlayerName, e.PlayerIPAddress)
	return &pb.InitPlayerProfileResponse{HasProfile: h}, err
}

func (s *grpcServer) FetchPlayerProfile(ctx context.Context, e *pb.FetchPlayerProfileRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := database.Find(e.PlayerUUID)
	return &pb.FetchPlayerProfileResponse{
		PlayerUUID: playerData.UUID,
		PlayerName: playerData.Name,
		Groups:     playerData.Groups,
		Settings:   playerData.Settings,
	}, err
}

func (s *grpcServer) FetchPlayerProfileByName(ctx context.Context, e *pb.FetchPlayerProfileByNameRequest) (*pb.FetchPlayerProfileResponse, error) {
	playerData, err := database.FindByName(e.PlayerName)
	return &pb.FetchPlayerProfileResponse{
		PlayerUUID: playerData.UUID,
		PlayerName: playerData.Name,
		Groups:     playerData.Groups,
		Settings:   playerData.Settings,
	}, err
}

func (s *grpcServer) SetPlayerServer(ctx context.Context, e *pb.SetPlayerServerRequest) (*pb.Empty, error) {
	err := database.SetPlayerServer(e.PlayerUUID, e.ServerName)
	return &pb.Empty{}, err
}

func (s *grpcServer) RemovePlayerServer(ctx context.Context, e *pb.RemovePlayerServerRequest) (*pb.Empty, error) {
	playerData, err := database.Find(e.PlayerUUID)

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
		log.Printf("To: %s / Level: %s / Reason: %s", entry.PunishedTo.Name, pb.PunishLevel(entry.Level), entry.Reason)
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
	playerData, err := database.FindByName(entry.PunishedTo.Name)

	serverName := playerData.Stats.CurrentServer


	if e.Force && entry.PunishedTo.UUID == "" {
		targetUUID, err := database.NameToUUIDwithMojang(entry.PunishedTo.Name)
		if err != nil {
			return &pb.SetPlayerPunishResponse{}, err
		}
		entry.PunishedTo.UUID = targetUUID
	}

	from := database.PunishPlayerData{
		UUID: entry.PunishedFrom.UUID,
		Name: entry.PunishedFrom.Name,
	}

	to := database.PunishPlayerData{
		UUID: entry.PunishedTo.UUID,
		Name: entry.PunishedTo.Name,
	}

	noProfile, offline, duplicate, coolDown, err := database.SetPlayerPunishment(e.Force, from, to, level, entry.Reason, entry.Date, entry.Expire)

	if e.Remote && !noProfile && !offline && !duplicate && !coolDown && err == nil {
		log.Printf("DISPATCH: " + playerData.Name)
		// DISPATCH target
		s.mu.Lock()
		defer s.mu.Unlock()

		for c := range s.psrChans {
			c <- pb.PunishStreamResponse{Type: pb.StreamType_DISPATCH, Target: serverName, Entry: entry}
		}
	}

	return &pb.SetPlayerPunishResponse{Noprofile: noProfile, Offline: offline, Duplicate: duplicate, Cooldown: coolDown}, err
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
