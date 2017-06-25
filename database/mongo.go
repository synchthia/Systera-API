package database

import (
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"strings"

	"gopkg.in/mgo.v2"
)

var session *mgo.Session

// NewMongoSession - Connect to MongoDB
func NewMongoSession(address string) {
	addresses := strings.Split(address, ",")
	logrus.WithFields(logrus.Fields{
		"servers": addresses,
	}).Infof("[MongoDB] Connecting...")

	di := &mgo.DialInfo{
		Addrs:    addresses,
		FailFast: true,
		Timeout:  5 * time.Second,
	}

	s, err := mgo.DialWithInfo(di)
	if err != nil {
		logrus.WithError(err).Errorf("[MongoDB] Failed Connection")
		NewMongoSession(address)

		//session = nil
		return
	}

	//log.Printf("[MongoDB] Connected!")
	logrus.Printf("[MongoDB] Connected!")

	session = s
}

/*func (m *mongoSession) GetMongoSession() *mgo.Session {
	return m.mSession
}*/
func GetMongoSession() (*mgo.Session, error) {
	if session == nil {
		return nil, errors.New("mongo session is not establish")
	}

	return session, nil
}
