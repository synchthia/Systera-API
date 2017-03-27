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
	FetchPlayerProfile(playerUUID string) (map[string]bool, error)

	SetPlayerServer(playerUUID, serverName string) error
}

type grpcServer struct {
	server   Server
	mu       sync.RWMutex
	asrChans map[chan pb.ActionStreamResponse]struct{}
}

func NewServer() *grpcServer {
	return &grpcServer{
		asrChans: make(map[chan pb.ActionStreamResponse]struct{}),
	}
}

func NewGRPCServer() *grpc.Server {
	server := grpc.NewServer()
	pb.RegisterSysteraServer(server, NewServer())
	return server
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
	//settings, err := database.FetchPlayerSettings(e.PlayerUUID)
	playerData, err := database.Find(e.PlayerUUID)
	return &pb.FetchPlayerProfileResponse{Settings: playerData.Settings}, err
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
