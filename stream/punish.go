package stream

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/Startail/Systera-API/systerapb"
)

// PublishPunish - Publish Punish
func PublishPunish(data *systerapb.PunishEntry) {
	c := pool.Get()
	defer c.Close()

	d := &systerapb.PunishmentStream{
		Type:        systerapb.PunishmentStream_PUNISH,
		PunishEntry: data,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.punishment.global", string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish Punishment")
	}
}

// PublishReport - Publish Report
func PublishReport(data *systerapb.ReportEntry) {
	c := pool.Get()
	defer c.Close()

	d := &systerapb.PunishmentStream{
		Type:        systerapb.PunishmentStream_REPORT,
		ReportEntry: data,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.punishment.global", string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish Report")
	}
}
