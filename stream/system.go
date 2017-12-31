package stream

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/Startail/Systera-API/apipb"
)

// PublishAnnounce - Publish Server Announce
func PublishAnnounce(target, msg string) error {
	c := pool.Get()
	defer c.Close()

	d := &apipb.SystemStream{
		Type: apipb.SystemStream_ANNOUNCE,
		Msg:  msg,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.system."+target, string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish Announce")
		return err
	}
	return nil
}

// PublishCommand - Publish Server Command
func PublishCommand(target, cmd string) error {
	c := pool.Get()
	defer c.Close()

	d := &apipb.SystemStream{
		Type: apipb.SystemStream_DISPATCH,
		Msg:  cmd,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.system."+target, string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish Command")
		return err
	}
	return nil
}
