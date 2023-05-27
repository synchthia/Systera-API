package server

import (
	"github.com/synchthia/systera-api/database"
	"github.com/synchthia/systera-api/stream"
	pb "github.com/synchthia/systera-api/systerapb"
	"golang.org/x/net/context"
)

func (s *grpcServer) Report(ctx context.Context, e *pb.ReportRequest) (*pb.ReportResponse, error) {
	from := database.PlayerIdentity{
		UUID: e.From.Uuid,
		Name: e.From.Name,
	}
	to := database.PlayerIdentity{
		UUID: e.To.Uuid,
		Name: e.To.Name,
	}

	entry, err := s.mysql.SetReport(from, to, e.ServerName, e.Message)
	if err == nil {
		stream.PublishReport(
			&pb.ReportEntry{
				From: &pb.PlayerIdentity{
					Uuid: entry.ReporterPlayerUUID,
					Name: entry.ReporterPlayerName,
				},
				To: &pb.PlayerIdentity{
					Uuid: entry.TargetPlayerUUID,
					Name: entry.TargetPlayerName,
				},
				Message: entry.Message,
				Date:    entry.Date.UnixMilli(),
				Server:  entry.Server,
			},
		)
	}

	return &pb.ReportResponse{}, err
}
