package database

import (
	"time"

	"github.com/sirupsen/logrus"
)

// ReportData - Report Data on Database
type Report struct {
	ID                 uint      `gorm:"primary_key;AutoIncrement;"`
	Date               time.Time `gorm:"type:datetime"`
	Message            string
	Server             string
	ReporterPlayerUUID string `gorm:"index;"`
	ReporterPlayerName string
	TargetPlayerUUID   string `gorm:"index;"`
	TargetPlayerName   string
}

// SetReport - Set Report Data
func (s *Mysql) SetReport(from, to PlayerIdentity, server, message string) (Report, error) {
	report := &Report{
		Date:               time.Now().UTC(),
		Message:            message,
		Server:             server,
		ReporterPlayerUUID: from.UUID,
		ReporterPlayerName: from.Name,
		TargetPlayerUUID:   to.UUID,
		TargetPlayerName:   to.Name,
	}
	result := s.client.Create(report)

	if result.Error != nil {
		logrus.WithError(result.Error).Errorf("[Report] Error @ SetReport")
		return Report{}, result.Error
	}

	logrus.WithFields(logrus.Fields{
		"from":    from,
		"to":      to,
		"message": message,
	}).Infof("[Report] Reported")

	return *report, nil
}
