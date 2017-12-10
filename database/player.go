package database

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"gitlab.com/Startail/Systera-API/util"
	"gopkg.in/mgo.v2/bson"
)

// PlayerData - PlayerProfile on Database
type PlayerData struct {
	ID                 bson.ObjectId              `bson:"_id,omitempty"`
	UUID               string                     `bson:"uuid" json:"id"`
	Name               string                     `bson:"name"`
	NameLower          string                     `bson:"name_lower"`
	Groups             []string                   `bson:"groups"`
	Stats              PlayerStats                `bson:"stats"`
	KnownUsernames     map[string]int64           `bson:"known_usernames"`
	KnownUsernameLower map[string]int64           `bson:"known_usernames_lower"`
	KnownAddresses     map[string]PlayerAddresses `bson:"known_addresses"`
	Settings           PlayerSettings             `bson:"settings"`
}

// PlayerStats - Stats in PlayerProfile
type PlayerStats struct {
	CurrentServer string `bson:"current_server"`
	FirstLogin    int64  `bson:"first_login"`
	LastLogin     int64  `bson:"last_login"`
}

// PlayerSettings - Player Personal Settings
type PlayerSettings struct {
	Vanish   bool `bson:"vanish" json:"vanish"`
	Japanize bool `bson:"japanize" json:"japanize"`
}

// PlayerAddresses - Player Address Entries
type PlayerAddresses struct {
	Address  string `bson:"address"`
	Hostname string `bson:"hostname"`
	Date     int64  `bson:"date"`
}

// UUIDToName - Get Player Name from UUID
func UUIDToName(uuid string) string {
	if _, err := GetMongoSession(); err != nil {
		return ""
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed UUIDToName(%s)", uuid)
		return ""
	}

	return playerData.Name
}

// NameToUUID - Get UUID from Player Name
func NameToUUID(name string) (string, error) {
	if _, err := GetMongoSession(); err != nil {
		return "", err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	err := coll.Find(bson.M{"name": name}).One(&playerData)
	if err != nil {
		uuid, err := NameToUUIDwithMojang(name)
		return uuid, err
	}

	return playerData.UUID, nil
}

// NameToUUIDwithMojang - Get UUID from Mojang API
func NameToUUIDwithMojang(name string) (string, error) {
	playerData := PlayerData{}
	err := util.GetFromJSONAPI("https://api.mojang.com/users/profiles/minecraft/"+name, &playerData)
	if err != nil {
		return "", err
	}

	return playerData.UUID, nil
}

// Find - Find PlayerProfile
func Find(uuid string) (PlayerData, error) {
	if _, err := GetMongoSession(); err != nil {
		return PlayerData{}, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")
	playerData := PlayerData{}

	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	return playerData, err
}

// FindByName - Find PlayerProfile from Name
func FindByName(name string) (PlayerData, error) {
	if _, err := GetMongoSession(); err != nil {
		return PlayerData{}, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")
	playerData := PlayerData{}

	nameLower := strings.ToLower(name)
	err := coll.Find(bson.M{"name_lower": nameLower}).Sort("-stats.last_login").One(&playerData)
	return playerData, err
}

// InitPlayerProfile - Initialize Player Profile
func InitPlayerProfile(uuid, name, ipAddress, hostname string) (int, error) {
	if _, err := GetMongoSession(); err != nil {
		return 0, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	nowtime := time.Now().UnixNano() / int64(time.Millisecond)

	profileCnt, err := coll.Find(bson.M{"uuid": uuid}).Count()
	if err != nil {
		logrus.WithError(err).Errorf("[Player] IPP: Failed Failed get profile %s(%s)", name, uuid)
		return 0, err
	}

	playerData := PlayerData{}

	if profileCnt == 0 {
		// Initialize
		playerData.KnownUsernames = make(map[string]int64)
		playerData.KnownUsernameLower = make(map[string]int64)
		playerData.KnownAddresses = make(map[string]PlayerAddresses)

		playerData.Stats.FirstLogin = nowtime
	} else {
		coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	}

	// User Profile
	playerData.UUID = uuid
	playerData.Name = name
	playerData.NameLower = strings.ToLower(name)
	playerData.Stats.LastLogin = nowtime

	// User Log (Address / Name)
	playerData.KnownAddresses[strings.NewReplacer(".", "_").Replace(ipAddress)] = PlayerAddresses{
		Address:  ipAddress,
		Hostname: hostname,
		Date:     nowtime,
	}
	playerData.KnownUsernames[name] = nowtime
	playerData.KnownUsernameLower[strings.ToLower(name)] = nowtime

	coll.Upsert(bson.M{"uuid": uuid}, playerData)

	logrus.WithFields(logrus.Fields{
		"name":    name,
		"uuid":    uuid,
		"address": ipAddress,
	}).Infof("[Player] InitPlayerProfile")

	return profileCnt, nil
}

// SetPlayerServer - Define Player Current Server
func SetPlayerServer(uuid, server string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	coll.Find(bson.M{"uuid": uuid}).One(&playerData)

	err := coll.Update(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"stats.current_server": server}})
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed Execute SetPlayerServer")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"server": server,
	}).Debugf("[Player] Set Server: %s > %s(%s)", server, playerData.Name, uuid)

	return nil
}

// PushPlayerSettings - Set Player Settings
func PushPlayerSettings(uuid, key string, value bool) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	_, err := coll.Upsert(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"settings." + key: value}})
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed Push Player Settings")
		return err
	}

	return nil
}
