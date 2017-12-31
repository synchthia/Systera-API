package stream

import (
	"encoding/json"

	"gitlab.com/Startail/Systera-API/apipb"

	"github.com/sirupsen/logrus"
)

// PublishPlayerGroups - Publish Player Groups
func PublishPlayerGroups(target string, data *apipb.PlayerEntry) {
	c := pool.Get()
	defer c.Close()

	d := &apipb.PlayerEntryStream{
		Type:  apipb.PlayerEntryStream_GROUPS,
		Entry: data,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.player."+target, string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish Player")
	}
}
