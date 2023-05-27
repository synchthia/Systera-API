package stream

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/synchthia/systera-api/systerapb"
)

// PublishChat - Publish Chat
func PublishChat(entry *systerapb.ChatEntry) error {
	c := pool.Get()
	defer c.Close()

	d := &systerapb.ChatStream{
		Type:      systerapb.ChatStream_CHAT,
		ChatEntry: entry,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.chat.global", string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish Chat")
		return err
	}

	return nil
}
