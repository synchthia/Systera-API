package database

import (
	"log"

	"gopkg.in/mgo.v2/bson"
)

type GroupData struct {
	ID          bson.ObjectId       `bson:"_id,omitempty"`
	Name        string              `bson:"name"`
	Prefix      string              `bson:"prefix"`
	Permissions map[string][]string `bson:"permissions"`
}

type GroupPerms struct {
	Name        string
	Prefix      string
	GlobalPerms []string
	ServerPerms []string
}

func FindGroupData() ([]GroupData, error) {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("groups")

	var groups []GroupData
	m := bson.M{}
	err := coll.Find(m).All(&groups)
	if err != nil {
		log.Printf("[!!!]: Failed Find GroupData from MongoDB: %s", err)
		return nil, err
	}

	return groups, nil
}
