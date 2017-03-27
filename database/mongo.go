package database

import (
	"log"

	"time"

	"gopkg.in/mgo.v2"
)

var session *mgo.Session

func NewMongoSession() {
	address := []string{"192.168.99.100:27017"}
	log.Printf("[MongoDB]: Connecting to: %s...", address)

	di := &mgo.DialInfo{
		Addrs:   address,
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
