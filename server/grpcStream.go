package server

import (
	"strings"

	"github.com/sirupsen/logrus"

	pb "gitlab.com/Startail/Systera-API/apipb"
)

func (s *grpcServer) ActionStream(r *pb.StreamRequest, as pb.Systera_ActionStreamServer) error {
	ech := make(chan pb.ActionStreamResponse)
	s.mu.Lock()
	s.asrChans[ech] = struct{}{}
	s.mu.Unlock()

	clientLen := len(s.asrChans)

	logrus.WithFields(logrus.Fields{
		"from":    r.Name,
		"clients": clientLen,
	}).Infof("[ACTION] [>] Connect > %s", r.Name)

	defer func() {
		s.mu.Lock()
		delete(s.asrChans, ech)
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

func (s *grpcServer) PunishStream(r *pb.StreamRequest, ps pb.Systera_PunishStreamServer) error {
	ech := make(chan pb.PunishStreamResponse)
	s.mu.Lock()
	s.psrChans[ech] = struct{}{}
	s.mu.Unlock()

	clientLen := len(s.psrChans)

	logrus.WithFields(logrus.Fields{
		"from":    r.Name,
		"clients": clientLen,
	}).Infof("[PUNISH] [>] Connect > %s", r.Name)

	defer func() {
		s.mu.Lock()
		delete(s.psrChans, ech)
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
