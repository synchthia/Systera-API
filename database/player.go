package database

import (
	"log"
	"strings"
	"time"

	"gitlab.com/Startail/Systera-API/util"
	"gopkg.in/mgo.v2/bson"
)

type PlayerData struct {
	ID                 bson.ObjectId    `bson:"_id,omitempty"`
	UUID               string           `bson:"uuid" json:"id"`
	Name               string           `bson:"name"`
	NameLower          string           `bson:"name_lower"`
	Groups             []string         `bson:"groups"`
	Stats              PlayerStats      `bson:"stats"`
	KnownUsernames     map[string]int64 `bson:"known_usernames"`
	KnownUsernameLower map[string]int64 `bson:"known_usernames_lower"`
	KnownAddresses     map[string]int64 `bson:"known_addresses"`
	Settings           map[string]bool  `bson:"settings"`
}

type PlayerStats struct {
	CurrentServer string `bson:"current_server"`
	FirstLogin    int64  `bson:"first_login,omitempty"`
	LastLogin     int64  `bson:"last_login"`
}

type PlayerSettings struct {
	Vanish   bool `bson:"vanish"`
	Japanize bool `bson:"japanize"`
}

func UUIDToName(uuid string) string {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	if err != nil {
		log.Printf("[!!!]: Failed UUIDToName(%s): %s", uuid, err.Error())
		return ""
	}

	return playerData.Name
}

func NameToUUID(name string) string {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	err := coll.Find(bson.M{"name": name}).One(&playerData)
	if err != nil {
		util.GetFromJSONAPI("https://api.mojang.com/users/profiles/minecraft/"+name, &playerData)
	}

	return playerData.UUID
}

func CheckHasProfile(uuid string) (bool, error) {
	session := GetMongoSession().Copy()
	defer session.Close()

	count, err := session.DB("systera").C("players").Find(bson.M{"uuid": uuid}).Count()
	if err != nil {
		log.Printf("[!!!]: Error occurred during CheckHasProfile: %s", err.Error())
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

func Find(uuid string) (PlayerData, error) {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")
	playerData := PlayerData{}

	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	return playerData, err
}

func FindByName(name string) (PlayerData, error) {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")
	playerData := PlayerData{}

	nameLower := strings.ToLower(name)
	err := coll.Find(bson.M{"name_lower": nameLower}).Sort("-stats.last_login").One(&playerData)
	return playerData, err
}

func InitPlayerProfile(uuid, name, ipaddress string) (bool, error) {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	nowtime := time.Now().UnixNano() / int64(time.Millisecond)
	playerData := PlayerData{}

	hasProfile, err := CheckHasProfile(uuid)
	if err != nil {
		log.Printf("[!!!]: Profile get failed @ InitPlayerProfile // %s(%s) from %s", name, uuid, ipaddress)
		return false, err
	}

	if !hasProfile {
		// First
		playerData.UUID = uuid
		playerData.Name = name
		playerData.NameLower = strings.ToLower(playerData.Name)
		playerData.Stats.FirstLogin = nowtime
		playerData.Stats.LastLogin = nowtime
	} else {
		// Not First
		coll.Find(bson.M{"uuid": uuid}).One(&playerData)
		playerData.Stats.LastLogin = nowtime
	}

	if playerData.KnownAddresses == nil {
		playerData.KnownAddresses = make(map[string]int64)
	}
	if playerData.KnownUsernames == nil {
		playerData.KnownUsernames = make(map[string]int64)
		playerData.KnownUsernameLower = make(map[string]int64)
	}
	playerData.KnownAddresses[strings.NewReplacer(".", "_").Replace(ipaddress)] = nowtime
	playerData.KnownUsernames[playerData.Name] = nowtime
	playerData.KnownUsernameLower[playerData.NameLower] = nowtime

	log.Printf("[InitPlayerProfile]: %s(%s) from %s", name, uuid, ipaddress)
	coll.Upsert(bson.M{"uuid": uuid}, bson.M{"$set": &playerData})
	return hasProfile, nil
}

func SetPlayerServer(uuid, server string) error {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	coll.Find(bson.M{"uuid": uuid}).One(&playerData)

	err := coll.Update(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"stats.current_server": server}})
	if err != nil {
		log.Printf("[!!!]: failed execute SetPlayerServer from MongoDB %s", err)
		return err
	}

	return nil
}

func RemovePlayerServer(uuid, server string) error {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)

	if playerData.Stats.CurrentServer == server {
		err = coll.Update(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"stats.current_server": ""}})
	}

	if err != nil {
		log.Printf("[!!!]: failed execute RemovePlayerServer from MongoDB %s", err)
		return err
	}

	return nil
}

func PushPlayerSettings(uuid, key string, value bool) error {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	coll.Find(bson.M{"uuid": uuid}).One(&playerData)

	err := coll.Update(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"settings." + key: value}})
	if err != nil {
		log.Printf("[!!!]: failed execute PushPlayerSettings from MongoDB: %s", err)
		return err
	}

	return nil
}
