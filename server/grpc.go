package server

import (
	"sync"

	"golang.org/x/net/context"

	"github.com/synchthia/systera-api/database"
	"github.com/synchthia/systera-api/stream"
	"github.com/synchthia/systera-api/systerapb"
	pb "github.com/synchthia/systera-api/systerapb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server interface {
	Announce(target, msg string)
	Dispatch(target, cmd string)

	Chat(entry pb.ChatEntry) error
	AddChatIgnore(identity pb.PlayerIdentity) error
	RemoveChatIgnore(identity pb.PlayerIdentity) error

	InitPlayerProfile(playerUUID, playerName, ipAddress, hostname string) (*systerapb.PlayerEntry, error)
	FetchPlayerProfile(playerUUID string) (string, string, map[string]bool, error)
	FetchPlayerProfileByName(playerName string) (string, string, map[string]bool, error)

	SetPlayerGroups(playerUUID string, groups []string) error
	SetPlayerServer(playerUUID, serverName string) error
	SetPlayerSettings(playerUUID, settings *systerapb.PlayerSettings) error

	AltLookup(playerUUID string) ([]pb.AltLookupEntry, error)

	GetPlayerPunish(playerUUID string, filterLevel pb.PunishLevel, includeExpired bool) []pb.PunishEntry
	SetPlayerPunish(force bool, entry pb.PunishEntry) (bool, bool, bool, bool, error)
	UnBan(target pb.PlayerIdentity) error

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
	mysql  *database.Mysql
}

func NewServer(mysql *database.Mysql) *grpcServer {
	return &grpcServer{
		mysql: mysql,
	}
}

func NewGRPCServer(mysql *database.Mysql) *grpc.Server {
	server := grpc.NewServer()
	reflection.Register(server)
	pb.RegisterSysteraServer(server, NewServer(mysql))
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
