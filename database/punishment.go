package database

import (
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

	//TEMPBAN - Temporary BAN
	TEMPBAN

	//PERMBAN - Permanently BAN
	PERMBAN
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

// PunishRule - Validation Rules (true -> Permit)
type PunishRule struct {
	NoProfile bool
	Duplicate bool
	Cooldown  bool
	Offline   bool
}

func (i PunishLevel) String() string {
	switch i {
	case WARN:
		return "WARN"
	case KICK:
		return "KICK"
	case TEMPBAN:
		return "TEMPBAN"
	case PERMBAN:
		return "PERMBAN"
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
func SetPlayerPunishment(force bool, from, to PlayerIdentity, level PunishLevel, reason string, date, expire int64) (success bool, result PunishRule, err error) {
	// Error Status
	success = false
	result = PunishRule{
		NoProfile: false,
		Duplicate: false,
		Cooldown:  false,
		Offline:   false,
	}
	if _, err := GetMongoSession(); err != nil {
		return success, result, err
	}

	session := session.Copy()
	defer session.Close()

	punishColl := session.DB("systera").C("punishments")
	availableBan, err := GetPlayerPunishment(to.UUID, TEMPBAN, false)
	for _, ban := range availableBan {
		if ban.Level == PERMBAN {
			result.Duplicate = true
		}

		if level <= TEMPBAN && ban.Level == TEMPBAN {
			result.Cooldown = true
		}
	}

	playerData := PlayerData{}
	playerQuery := session.DB("systera").C("players").Find(bson.M{"uuid": to.UUID})
	playerCnt, playerCntErr := playerQuery.Count()
	if playerCntErr != nil {
		return success, result, playerCntErr
	}

	if playerCnt == 0 {
		result.NoProfile = true
	} else {
		playerQuery.One(&playerData)

		if playerData.Stats.CurrentServer == "" {
			result.Offline = true
		}
	}

	// Check
	// NoProfile
	if !force && result.NoProfile {
		return
	}

	// Duplicate
	if result.Duplicate {
		if level == TEMPBAN || level == PERMBAN {
			return
		}
	}

	// Cooldown
	if result.Cooldown {
		if level == TEMPBAN {
			return
		}
	}

	// Offline
	if !force && result.Offline {
		if level == WARN || level == KICK {
			return
		}
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
		return
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

	return true, result, err
}
