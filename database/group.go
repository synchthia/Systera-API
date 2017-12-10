package database

import (
	"errors"

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

// AddGroup - Create New Group
func AddGroup(name string, prefix string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("groups")

	groupLen, findErr := coll.Find(bson.M{"name": name}).Count()
	if groupLen != 0 {
		return errors.New("group already exists")
	}
	if findErr != nil {
		return findErr
	}

	group := GroupData{
		Name:   name,
		Prefix: prefix,
	}

	err := coll.Insert(&group)

	return err
}
