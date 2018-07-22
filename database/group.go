package database

import (
	"errors"

	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
)

// GroupData - Permission Group Data
type GroupData struct {
	ID          bson.ObjectId       `bson:"_id,omitempty"`
	Name        string              `bson:"name"`
	Prefix      string              `bson:"prefix"`
	Permissions map[string][]string `bson:"permissions"`
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

// CreateGroup - Create New Group
func CreateGroup(groupData GroupData) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("groups")

	groupLen, findErr := coll.Find(bson.M{"name": groupData.Name}).Count()
	if findErr != nil {
		return findErr
	}
	if groupLen != 0 {
		return errors.New("group already exists")
	}

	err := coll.Insert(&groupData)

	return err
}

// RemoveGroup - Remove Group
func RemoveGroup(groupName string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("groups")

	groupLen, findErr := coll.Find(bson.M{"name": groupName}).Count()
	if findErr != nil {
		return findErr
	}
	if groupLen == 0 {
		return errors.New("group not exists")
	}

	err := coll.Remove(bson.M{"name": groupName})

	return err
}

// AddPermission - Add Permission
func AddPermission(groupName, target string, permissions []string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("groups")

	cnt, cntErr := coll.Find(bson.M{"name": groupName}).Count()
	if cntErr != nil {
		return cntErr
	}
	if cnt == 0 {
		return errors.New("group not exists")
	}

	err := coll.Update(
		bson.M{"name": groupName},
		bson.M{"$addToSet": bson.M{"permissions." + target: bson.M{"$each": permissions}}},
	)
	return err
}

// RemovePermission - Remove Permission
func RemovePermission(groupName, target string, permissions []string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("groups")

	query := coll.Find(bson.M{"name": groupName})
	cnt, cntErr := query.Count()
	if cntErr != nil {
		return cntErr
	}
	if cnt == 0 {
		return errors.New("group not exists")
	}

	err := coll.Update(bson.M{"name": groupName}, bson.M{"$pull": bson.M{"permissions." + target: bson.M{"$in": permissions}}})
	return err
}
