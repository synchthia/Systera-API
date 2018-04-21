package database

import (
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
)

// ReportData - Report Data on Database
type ReportData struct {
	ID           bson.ObjectId  `bson:"_id,omitempty"`
	Message      string         `bson:"message"`
	Date         int64          `bson:"date"`
	Server       string         `bson:"server"`
	ReportedFrom PlayerIdentity `bson:"reported_from"`
	ReportedTo   PlayerIdentity `bson:"reported_to"`
}

// SetReport - Set Report Data
func SetReport(from, to PlayerIdentity, message string) (ReportData, error) {
	if _, err := GetMongoSession(); err != nil {
		return ReportData{}, err
	}

	session := session.Copy()
	defer session.Close()

	coll := session.DB("systera").C("reports")

	nowtime := time.Now().UnixNano() / int64(time.Millisecond)

	fromUser, findErr := FindPlayer(from.UUID)
	if findErr != nil {
		logrus.WithError(findErr).Errorf("[Report] Error @ SetReport")
		return ReportData{}, findErr
	}

	reportData := &ReportData{
		Message:      message,
		Date:         nowtime,
		Server:       fromUser.Stats.CurrentServer,
		ReportedFrom: from,
		ReportedTo:   to,
	}

	err := coll.Insert(&reportData)
	if err != nil {
		logrus.WithError(err).Errorf("[Report] Error @ SetReport")
		return ReportData{}, err
	}

	logrus.WithFields(logrus.Fields{
		"from":    from,
		"to":      to,
		"message": message,
	}).Infof("[Report] Reported")
	return *reportData, nil
}
