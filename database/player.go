package database

import (
	"strings"
	"time"

	"github.com/minotar/minecraft"

	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
)

// PlayerData - PlayerProfile on Database
type PlayerData struct {
	ID                 bson.ObjectId     `bson:"_id,omitempty"`
	UUID               string            `bson:"uuid" json:"id"`
	Name               string            `bson:"name"`
	NameLower          string            `bson:"name_lower"`
	Groups             []string          `bson:"groups"`
	Stats              PlayerStats       `bson:"stats"`
	KnownUsernames     map[string]int64  `bson:"known_usernames"`
	KnownUsernameLower map[string]int64  `bson:"known_usernames_lower"`
	KnownAddresses     []PlayerAddresses `bson:"known_addresses"`
	Settings           PlayerSettings    `bson:"settings"`
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

// PlayerIdentity - Player Data Set (Used from ex. punishment, report...)
type PlayerIdentity struct {
	UUID string `bson:"uuid"`
	Name string `bson:"name"`
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
	err := coll.Find(bson.M{"name_lower": strings.ToLower(name)}).One(&playerData)
	if err != nil {
		uuid, err := NameToUUIDwithMojang(name)
		return uuid, err
	}

	return playerData.UUID, nil
}

// NameToUUIDwithMojang - Get UUID from Mojang API
func NameToUUIDwithMojang(name string) (string, error) {
	uuid, err := minecraft.NewMinecraft().GetUUID(name)
	if err != nil {
		return "", err
	}

	return uuid, nil
}

// FindPlayer - Find PlayerProfile
func FindPlayer(uuid string) (PlayerData, error) {
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

// FindPlayerByName - Find PlayerProfile from Name
func FindPlayerByName(name string) (PlayerData, error) {
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

// Migrate
func Migrate() {
	logrus.Infof("[MIGRATE] Migrate Initializing...")

	if _, err := GetMongoSession(); err != nil {
		return
	}

	playerData := []PlayerData{}
	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")
	coll.Find(bson.M{}).All(&playerData)

	// for _, e := range playerData {
	// 	var pas []PlayerAddresses
	// 	for _, v := range e.KnownAddresses {
	// 		logrus.WithFields(logrus.Fields{
	// 			"Address":  v.Address,
	// 			"Hostname": v.Hostname,
	// 			"Date":     time.Unix(v.Date/1000, 0).Format("2006-01-02 15:04:05"),
	// 		}).Infof("[%s]", e.Name)
	// 		pa := PlayerAddresses{
	// 			Address:  v.Address,
	// 			Hostname: v.Hostname,
	// 			Date:     v.Date,
	// 		}
	// 		pas = append(pas, pa)
	// 	}
	// 	coll.Update(bson.M{"uuid": e.UUID}, bson.M{"$set": bson.M{"new_known_addresses": pas}})
	// }

	// for _, e := range playerData {
	// 	var pas []PlayerAddresses
	// 	for _, v := range e.NewKnownAddresses {
	// 		logrus.WithFields(logrus.Fields{
	// 			"Address":  v.Address,
	// 			"Hostname": v.Hostname,
	// 			"Date":     time.Unix(v.Date/1000, 0).Format("2006-01-02 15:04:05"),
	// 		}).Infof("[%s]", e.Name)
	// 		pa := PlayerAddresses{
	// 			Address:  v.Address,
	// 			Hostname: v.Hostname,
	// 			Date:     v.Date,
	// 		}
	// 		pas = append(pas, pa)
	// 	}
	// 	coll.Update(bson.M{"uuid": e.UUID}, bson.M{"$set": bson.M{"known_addresses": pas}})
	// }

	coll.Update(bson.M{}, bson.M{"$unset": bson.M{"new_known_addresses": 1}})
	// Test (Altlookup)
	// coll.Find(bson.M{"uuid": e.UUID})
	playerData2 := []PlayerData{}
	coll.Find(bson.M{"known_addresses.address": "172.18.0.1"}).All(&playerData2)
	for _, v := range playerData2 {
		logrus.Printf("[Result] %s", v.Name)
	}
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
		// playerData.KnownAddresses = make([]PlayerAddresses)

		playerData.Stats.FirstLogin = nowtime
	} else {
		coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	}

	// User Profile
	playerData.UUID = uuid
	playerData.Name = name
	playerData.NameLower = strings.ToLower(name)
	playerData.Stats.LastLogin = nowtime

	// Player Groups
	if len(playerData.Groups) == 0 {
		playerData.Groups = []string{"default"}
	}

	// User Log (Address / Name)
	var newPlayerAddresses []PlayerAddresses
	for _, playerAddress := range playerData.KnownAddresses {
		if ipAddress == playerAddress.Address {
			newPlayerAddresses = append(newPlayerAddresses, PlayerAddresses{
				Address:  ipAddress,
				Hostname: hostname,
				Date:     nowtime,
			})
		} else {
			newPlayerAddresses = append(newPlayerAddresses, playerAddress)
		}
	}

	playerData.KnownAddresses = newPlayerAddresses
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

// SetPlayerGroups - Define Player Groups
func SetPlayerGroups(uuid string, groups []string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	err := coll.Update(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"groups": groups}})
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed Execute SetPlayerGroups")
		return err
	}

	return nil
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

// AltLookup - AltLookup player accounts
func AltLookup() {

}

// AltLookupByName - AltLookup player's Name
func AltLookupByName(playerName string) {

}
