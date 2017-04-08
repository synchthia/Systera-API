package database

import (
	"log"

	"time"

	"gopkg.in/mgo.v2"
	"strings"
)

var session *mgo.Session

func NewMongoSession(address string) {
	addresses := strings.Split(address, ",")
	log.Printf("[MongoDB]: Connecting to: %s...", addresses)

	di := &mgo.DialInfo{
		Addrs:   addresses,
		Timeout: 5 * time.Second,
	}

	s, err := mgo.DialWithInfo(di)
	if err != nil {
		log.Fatalf("[!!!]: Error @ during Connecting Mongo: %s", err)
		return
	}

	log.Printf("[MongoDB]: Connected!")
	session = s
}

func GetMongoSession() *mgo.Session {
	return session
}
