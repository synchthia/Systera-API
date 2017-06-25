package database

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

// GroupData - Permission Group Data
type GroupData struct {
	ID          bson.ObjectId       `bson:"_id,omitempty"`
	Name        string              `bson:"name"`
	Prefix      string              `bson:"prefix"`
	Permissions map[string][]string `bson:"permissions"`
}

// GroupPerms - Group Permission
type GroupPerms struct {
	Name        string
	Prefix      string
	GlobalPerms []string
	ServerPerms []string
}

// FindGroupData - Find Group Entry
func FindGroupData() ([]GroupData, error) {
	if _, err := GetMongoSession(); err != nil {
		return nil, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("groups")

	var groups []GroupData
	err := coll.Find(bson.M{}).All(&groups)
	if err != nil {
		logrus.WithError(err).Errorf("[Group] Failed Find GroupData: %s", err)
		return nil, err
	}

	return groups, nil
}
