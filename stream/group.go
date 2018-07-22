package stream

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/Startail/Systera-API/systerapb"
)

// PublishGroup - Publish Group
func PublishGroup(data *systerapb.GroupEntry) {
	c := pool.Get()
	defer c.Close()

	d := &systerapb.GroupStream{
		Type:       systerapb.GroupStream_GROUP,
		GroupEntry: data,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.group.global", string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish group")
	}
}

// PublishPerms - Publish Permissions
func PublishPerms(target string, data *systerapb.GroupEntry) {
	c := pool.Get()
	defer c.Close()

	d := &systerapb.GroupStream{
		Type:       systerapb.GroupStream_PERMISSIONS,
		GroupEntry: data,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.group."+target, string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish permissions")
	}
}
