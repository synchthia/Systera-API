package database

import (
	"errors"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/sirupsen/logrus"
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

	logrus.Printf("[MongoDB] Connected!")
	session = s

	genCollWithIndex("players", []string{"-uuid", "-name", "-name_lower"})
	genCollWithIndex("groups", []string{"name"})
	genCollWithIndex("punishments", []string{"punished_to.uuid"})
}

func genCollWithIndex(collName string, keys []string) {
	session, err := GetMongoSession()
	if err != nil {
		panic(err)
	}

	session = session.Copy()
	defer session.Close()

	if coll := session.DB("systera").C(collName); err == nil {
		// Index
		err := coll.EnsureIndex(mgo.Index{
			Key:    keys,
			Unique: true,
		})

		if err != nil {
			logrus.WithError(err).Errorf("[Index] Failed Index on %s", collName)
		}
	}
}

func GetMongoSession() (*mgo.Session, error) {
	if session == nil {
		return nil, errors.New("mongo session is not establish")
	}

	return session, nil
}
