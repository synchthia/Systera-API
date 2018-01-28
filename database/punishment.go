package database

import (
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
)

// PunishLevel - Punishment level
type PunishLevel int32

const (
	//WARN - Warning
	WARN PunishLevel = iota

	//KICK - Kick from Server
	KICK

	//TBAN - Temporary BAN
	TBAN

	//PBAN - Permanently BAN
	PBAN
)

// PunishmentData - PunishData on Database
type PunishmentData struct {
	ID           bson.ObjectId  `bson:"_id,omitempty"`
	Available    bool           `bson:"available"`
	Level        PunishLevel    `bson:"level"`
	Reason       string         `bson:"reason"`
	Date         int64          `bson:"date"`
	Expire       int64          `bson:"expire,omitempty"`
	PunishedFrom PlayerIdentity `bson:"punished_from"`
	PunishedTo   PlayerIdentity `bson:"punished_to"`
}

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

// GetPlayerPunishment - Get Player Punishment History
func GetPlayerPunishment(playerUUID string, filterLevel PunishLevel, includeExpired bool) ([]PunishmentData, error) {
	if _, err := GetMongoSession(); err != nil {
		return nil, err
	}

	session := session.Copy()
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
			logrus.WithError(err).Errorf("[Punish] Failed GetPlayerPunishment(%s)", playerUUID)
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
			logrus.WithError(err).Errorf("[Punish] Failed GetPlayerPunishment(%s)", playerUUID)
			return nil, err
		}
	}
	return lookup, nil
}

// SetPlayerPunishment - Punish Player
func SetPlayerPunishment(force bool, from, to PlayerIdentity, level PunishLevel, reason string, date, expire int64) (bool, bool, bool, bool, error) {
	if _, err := GetMongoSession(); err != nil {
		return false, false, false, false, err
	}

	session := session.Copy()
	defer session.Close()

	playerColl := session.DB("systera").C("players")
	punishColl := session.DB("systera").C("punishments")

	//Error
	noProfile := false
	offline := false

	profile := playerColl.Find(bson.M{"name_lower": strings.ToLower(to.Name)}).Sort("-stats.last_login")

	//If not found Player Profile without Force option, return NoProfile
	p, err := profile.Count()
	if !force && p == 0 {
		noProfile = true
		logrus.Debugf("[Punishment] %s has not Profile", to.Name)
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
	logrus.Debugf("[Punishment] PBAN: = %d", len(availableBans))
	if len(availableBans) != 0 {
		return noProfile, offline, true, false, err
	}

	//全処罰で、Expireが来ていないAvailableな奴をすべて取得する
	//1つでもあればクールダウンを理由でreturn
	availableTempBan, err := GetPlayerPunishment(to.Name, TBAN, false)
	logrus.Debugf("[Punishment] Non Expired Punishments: %d", len(availableTempBan))
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
		logrus.WithError(err).Errorf("[Punish] Failed Punish Player")
		return noProfile, offline, false, false, err
	}

	var expireDate string
	if expire != 0 {
		expireDate = time.Unix(expire/1000, 0).String()
	}

	logrus.WithFields(logrus.Fields{
		"level":  level,
		"reason": reason,
		"expire": expireDate,
	}).Infof("[Punishment] %s -> %s", from.Name, to.Name)

	return noProfile, offline, false, false, nil
}
