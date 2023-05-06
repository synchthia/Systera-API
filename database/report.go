package database

import (
	"time"

	"github.com/sirupsen/logrus"
)

// ReportData - Report Data on Database
type Report struct {
	ID           int32 `gorm:"primary_key;AutoIncrement;"`
	Message      string
	Data         int64
	Server       string
	ReportedFrom PlayerIdentity `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	ReportedTo   PlayerIdentity `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// SetReport - Set Report Data
func (s *Mysql) SetReport(from, to PlayerIdentity, message string) (Report, error) {
	fromUser, findErr := s.FindPlayer(from.UUID)
	if findErr != nil {
		logrus.WithError(findErr).Errorf("[Report] Error @ SetReport")
		return Report{}, findErr
	}

	nowtime := time.Now().UnixNano() / int64(time.Millisecond)
	report := &Report{
		Message:      message,
		Data:         nowtime,
		Server:       fromUser.Stats.CurrentServer,
		ReportedFrom: from,
		ReportedTo:   to,
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
