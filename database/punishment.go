package database

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"log"
	"strings"
)

type PunishLevel int32

type PunishmentData struct {
	ID           bson.ObjectId `bson:"_id,omitempty"`
	Available    bool          `bson:"available"`
	Level        PunishLevel   `bson:"level"`
	Reason       string        `bson:"reason"`
	Date         int64         `bson:"date"`
	Expire       int64         `bson:"expire,omitempty"`
	PunishedFrom PunishPlayerData `bson:"punished_from"`
	PunishedTo   PunishPlayerData `bson:"punished_to"`
}

type PunishPlayerData struct {
	UUID string `bson:"uuid"`
	Name string `bson:"name"`
}

const (
	WARN PunishLevel = iota
	KICK 
	TBAN 
	PBAN 
)

func (i PunishLevel) String() string {
	switch i {
	case WARN:
		return "WARN"
	case KICK:
		return "KICK"
	case TBAN:
		return "TBAN"
	case PBAN:
		return "PBAN"
	}
	return ""
}

func GetPlayerPunishment(playerUUID string, filterLevel PunishLevel, includeExpired bool) ([]PunishmentData, error) {
	session := GetMongoSession().Copy()
	defer session.Close()
	coll := session.DB("systera").C("punishments")

	nowtime := time.Now().UnixNano() / int64(time.Millisecond)

	var lookup []PunishmentData

	if includeExpired == true {
		err := coll.Find(
			bson.M{
				"punished_to.uuid": playerUUID,
				"level":            bson.M{"$gte": filterLevel},
			}).All(&lookup)
		if err != nil {
			log.Printf("[!!!]: Error @ GetPlayerPunishment(%s) from MongoDB: %s", playerUUID, err)
			return nil, err
		}
	} else {
		err := coll.Find(
			bson.M{
				"punished_to.uuid": playerUUID,
				"level":            bson.M{"$gte": filterLevel},
				"available":        true,
				"$or":              []bson.M{{"expire": bson.M{"$exists": false}}, {"expire": bson.M{"$gte": nowtime}}},
			}).All(&lookup)
		if err != nil {
			log.Printf("[!!!]: Error @ GetPlayerPunishment(%s) from MongoDB: %s", playerUUID, err)
			return nil, err
		}
	}
	return lookup, nil
}

func SetPlayerPunishment(force bool, from, to PunishPlayerData, level PunishLevel, reason string, date, expire int64) (bool, bool, bool, bool, error) {
	session := GetMongoSession().Copy()
	defer session.Close()
	playerColl := session.DB("systera").C("players")
	punishColl := session.DB("systera").C("punishments")

	/* Error */
	noProfile := false
	offline := false

	profile := playerColl.Find(bson.M{"name_lower": strings.ToLower(to.Name)}).Sort("-stats.last_login")

	//If not found Player Profile without Force option, return NoProfile
	p, err := profile.Count()
	if !force && p == 0 {
		noProfile = true
		log.Printf("[!!!]: %s is has not Profile", to.Name)
		return noProfile, offline, false, false, nil
	}

	if !force {
		playerData := &PlayerData{}
		profile.One(&playerData)

		// Convert true Name
		to.UUID = playerData.UUID
		to.Name = playerData.Name

		// if Offline
		if level != TBAN && level != PBAN && playerData.Stats.CurrentServer == "" {
			offline = true
			return noProfile, offline, false, false, nil
		}
	}

	//If already Permanently Banned, return Duplicate
	availableBans, err := GetPlayerPunishment(to.UUID, PBAN, false)
	log.Printf("[]: PBAN: = %d", len(availableBans))
	if len(availableBans) != 0 {
		return noProfile, offline, true, false, err
	}

	//全処罰で、Expireが来ていないAvailableな奴をすべて取得する
	//1つでもあればクールダウンを理由でreturn
	availableTempBan, err := GetPlayerPunishment(to.Name, TBAN, false)
	log.Printf("[Punishment]: Non Expired punishments: %d", len(availableTempBan))
	if level >= TBAN && len(availableTempBan) != 0 {
		return noProfile, offline, false, true, err
	}

	punishData := PunishmentData{
		Available:    true,
		Level:        level,
		Reason:       reason,
		Date:         date,
		Expire:       expire,
		PunishedFrom: from,
		PunishedTo:   to,
	}

	err = punishColl.Insert(&punishData)
	if err != nil {
		log.Printf("[!!!]: Failed Punish player: %s", err)
		return noProfile, offline, false, false, err
	}

	var expireDate string
	if expire != 0 {
		expireDate = time.Unix(expire/1000, 0).String()
	}

	log.Printf("[Punishment]: %s -> %s (Level: %s / Reason: %s) %s",
		from.Name, to.Name, level, reason, expireDate)
	return noProfile, offline, false, false, nil
}
