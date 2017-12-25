package server

import (
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	pb "gitlab.com/Startail/Systera-API/apipb"
)

func (s *grpcServer) QuitStream(ctx context.Context, e *pb.QuitStreamRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for c := range s.actionChans {
		c <- pb.ActionStreamResponse{Type: pb.StreamType_QUIT, Target: e.Name}
	}

	for c := range s.playerChans {
		c <- pb.PlayerStreamResponse{Type: pb.StreamType_QUIT, Target: e.Name}
	}

	for c := range s.punishChans {
		c <- pb.PunishStreamResponse{Type: pb.StreamType_QUIT, Target: e.Name}
	}
	return &pb.Empty{}, nil
}

func (s *grpcServer) ActionStream(r *pb.StreamRequest, as pb.Systera_ActionStreamServer) error {
	ech := make(chan pb.ActionStreamResponse)
	s.mu.Lock()
	s.actionChans[ech] = struct{}{}
	s.mu.Unlock()

	clientLen := len(s.actionChans)

	logrus.WithFields(logrus.Fields{
		"from":    r.Name,
		"clients": clientLen,
	}).Infof("[ACTION] [>] Connect > %s", r.Name)

	defer func() {
		s.mu.Lock()
		delete(s.actionChans, ech)
		s.mu.Unlock()
		close(ech)
		logrus.WithFields(logrus.Fields{
			"from":    r.Name,
			"clients": clientLen,
		}).Infof("[ACTION] [x] CLOSED > %s", r.Name)
	}()

	for e := range ech {
		if e.Target != "GLOBAL" && !strings.HasPrefix(e.Target, r.Name) {
			continue
		}

		if e.Type == pb.StreamType_QUIT {
			return nil
		}

		logrus.WithFields(logrus.Fields{
			"type":   e.Type,
			"from":   r.Name,
			"target": e.Target,
			"cmd":    e.Cmd,
		}).Infof("[ACTION] [<->] %s > %s", e.Type, e.Target)

		err := as.Send(&e)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *grpcServer) PlayerStream(r *pb.StreamRequest, ps pb.Systera_PlayerStreamServer) error {
	ech := make(chan pb.PlayerStreamResponse)
	s.mu.Lock()
	s.playerChans[ech] = struct{}{}
	s.mu.Unlock()

	clientLen := len(s.playerChans)

	logrus.WithFields(logrus.Fields{
		"from":    r.Name,
		"clients": clientLen,
	}).Infof("[PLAYER] [>] Connect > %s", r.Name)

	defer func() {
		s.mu.Lock()
		delete(s.playerChans, ech)
		s.mu.Unlock()
		close(ech)
		logrus.WithFields(logrus.Fields{
			"from":    r.Name,
			"clients": clientLen,
		}).Infof("[PLAYER] [x] CLOSED > %s", r.Name)
	}()

	for e := range ech {
		if e.Target != "GLOBAL" && !strings.HasPrefix(e.Target, r.Name) {
			continue
		}

		if e.Type == pb.StreamType_QUIT {
			return nil
		}

		logrus.WithFields(logrus.Fields{
			"type":   e.Type,
			"from":   r.Name,
			"target": e.Target,
		}).Infof("[PLAYER] [<->] %s > %s", e.Type, e.Target)

		err := ps.Send(&e)
		if err != nil {
			logrus.WithError(err).Errorf("[gRPC]")
			return err
		}
	}
	return nil
}

func (s *grpcServer) PunishStream(r *pb.StreamRequest, ps pb.Systera_PunishStreamServer) error {
	ech := make(chan pb.PunishStreamResponse)
	s.mu.Lock()
	s.punishChans[ech] = struct{}{}
	s.mu.Unlock()

	clientLen := len(s.punishChans)

	logrus.WithFields(logrus.Fields{
		"from":    r.Name,
		"clients": clientLen,
	}).Infof("[PUNISH] [>] Connect > %s", r.Name)

	defer func() {
		s.mu.Lock()
		delete(s.punishChans, ech)
		s.mu.Unlock()
		close(ech)
		logrus.WithFields(logrus.Fields{
			"from":    r.Name,
			"clients": clientLen,
		}).Infof("[PUNISH] [x] CLOSED > %s", r.Name)
	}()

	for e := range ech {
		if e.Target != "GLOBAL" && !strings.HasPrefix(e.Target, r.Name) {
			continue
		}

		logrus.WithFields(logrus.Fields{
			"type":   e.Type,
			"from":   r.Name,
			"target": e.Target,
		}).Infof("[PUNISH] [<->] %s > %s", e.Type, e.Target)

		logrus.Debugf("[PUNISH_S]: PUNISH > Requested (%s / Target: %s) [%s]", e.Type, e.Target, r.Name)
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
