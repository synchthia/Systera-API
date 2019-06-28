package stream

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/synchthia/Systera-API/systerapb"
)

// PublishAnnounce - Publish Server Announce
func PublishAnnounce(target, msg string) error {
	c := pool.Get()
	defer c.Close()

	d := &systerapb.SystemStream{
		Type: systerapb.SystemStream_ANNOUNCE,
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

	d := &systerapb.SystemStream{
		Type: systerapb.SystemStream_DISPATCH,
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
