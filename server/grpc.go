package server

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"

	pb "gitlab.com/Startail/Systera-API/apipb"
	"gitlab.com/Startail/Systera-API/database"
	"google.golang.org/grpc"
)

type Server interface {
	Announce(msg string)
	InitPlayerProfile(playerUUID, playerName, ipAddress string) (bool, error)
	FetchPlayerProfile(playerUUID string) (map[string]bool, error)
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
	fmt.Println("Added New Watcher", ech)

	defer func() {
		s.mu.Lock()
		delete(s.asrChans, ech)
		s.mu.Unlock()
		close(ech)
		fmt.Println("Deleted Watcher", ech)
	}()

	fmt.Println("sending")

	for e := range ech {
		/*if !strings.HasPrefix(e.Message, r.Name) {
			continue
		}*/
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
		c <- pb.ActionStreamResponse{Type: "dispatch", Cmd: e.Message}
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
