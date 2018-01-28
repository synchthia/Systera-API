package stream

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/Startail/Systera-API/systerapb"
)

// PublishReport - Publish Report
func PublishReport(data *systerapb.ReportEntry) {
	c := pool.Get()
	defer c.Close()

	d := &systerapb.ReportEntryStream{
		Type:  systerapb.ReportEntryStream_REPORT,
		Entry: data,
	}
	serialized, _ := json.Marshal(&d)
	logrus.Debugln(d)

	_, err := c.Do("PUBLISH", "systera.report.global", string(serialized))
	if err != nil {
		logrus.WithError(err).Errorf("[Publish] Failed Publish Report")
	}
}
